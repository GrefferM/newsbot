package autotrade

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
)

type planner struct {
	mu    sync.Mutex
	asked []string
	plan  research.Plan
	err   error
}

func (p *planner) Event(_ context.Context, asset string, _ schedule.Entry) (research.Plan, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.asked = append(p.asked, asset)

	return p.plan, p.err
}

type shelf struct {
	mu   sync.Mutex
	kept []string
}

func (s *shelf) FixPlan(asset string, _ research.Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kept = append(s.kept, asset)

	return nil
}

type watcher struct {
	mu    sync.Mutex
	armed []research.Scenario
}

func (w *watcher) Arm(_ string, plan research.Scenario) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.armed = append(w.armed, plan)

	return nil
}

type agenda []schedule.Entry

func (a agenda) Upcoming(time.Time, time.Duration) []schedule.Entry {
	return a
}

type store struct {
	saved []State
}

func (s *store) Save(state State) error {
	s.saved = append(s.saved, state)

	return nil
}

func scenario(risk string, above float64) research.Scenario {
	return research.Scenario{
		Condition: "пробій після події " + risk, Action: "лонг", Risk: risk, Direction: research.DirectionLong,
		WakeAbove: above, SizePct: 1, Stop: above * 0.9, Take: above * 1.2,
	}
}

func service(p *planner, w *watcher, st *store, entries ...schedule.Entry) *Service {
	return NewService(Deps{
		Planner: p, Shelf: &shelf{}, Watcher: w, Agenda: agenda(entries), Store: st, Lead: 10 * time.Minute,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, State{})
}

func TestBeforeATimedCoinEventTheSafestPlanIsSetToBeWatchedOnce(t *testing.T) {
	now := time.Date(2026, 11, 19, 4, 0, 0, 0, time.UTC)
	unlock := schedule.Entry{ID: "unlock:KAITO", Time: now.Add(10 * time.Minute), Title: "Розблокування 7.29% обігу", Kind: schedule.Unlock, Assets: []string{"KAITO"}}
	p := &planner{plan: research.Plan{Scenarios: []research.Scenario{scenario(research.RiskHigh, 2), scenario(research.RiskMedium, 3), scenario(research.RiskMinimal, 4)}}}
	w, st := &watcher{}, &store{}
	s := service(p, w, st, unlock)

	s.Tick(context.Background(), now)

	if len(p.asked) != 0 || len(w.armed) != 0 {
		t.Fatal("switched off, yet a plan was asked for")
	}

	if err := s.Switch(true); err != nil || !s.Enabled() {
		t.Fatalf("switch on: %v", err)
	}

	s.Tick(context.Background(), now)
	s.Tick(context.Background(), now.Add(time.Minute))

	if len(p.asked) != 1 || len(w.armed) != 1 || w.armed[0].Risk != research.RiskMinimal {
		t.Fatalf("one question, the safest plan armed once: asked %v, armed %+v", p.asked, w.armed)
	}

	if w.armed[0].WakeAfter != "2026-11-19T04:10:00Z" || w.armed[0].WakeAbove != 4 {
		t.Errorf("the plan sleeps until the event and keeps its price: %+v", w.armed[0])
	}

	if len(s.pending) != 1 || !s.pending[0].Silent || s.pending[0].Kind != armedKind || s.pending[0].Category != news.Trade {
		t.Errorf("the feed is told quietly: %+v", s.pending)
	}

	if last := st.saved[len(st.saved)-1]; !last.Enabled || len(last.Planned) != 1 {
		t.Errorf("the switch and the claimed event are kept: %+v", last)
	}
}

func TestOnlyTimedCoinEventsArePlannedAndARiskyPlanAloneIsLeft(t *testing.T) {
	now := time.Date(2026, 11, 19, 4, 0, 0, 0, time.UTC)
	soon := now.Add(5 * time.Minute)
	entries := []schedule.Entry{
		{ID: "cpi", Time: soon, Title: "CPI", Kind: schedule.Inflation},
		{ID: "day", Time: soon, Title: "Розблокування", Kind: schedule.Unlock, Assets: []string{"ERA"}, AllDay: true},
		{ID: "earn", Time: soon, Title: "Звіт", Kind: schedule.Earnings, Assets: []string{"NVDA"}},
		{ID: "vote", Time: soon, Title: "Голосування", Kind: schedule.Coin, Assets: []string{"GRASS", "ZRO"}},
	}
	p := &planner{plan: research.Plan{Scenarios: []research.Scenario{scenario(research.RiskHigh, 2)}}}
	w := &watcher{}
	s := service(p, w, &store{}, entries...)
	_ = s.Switch(true)

	s.Tick(context.Background(), now)

	if len(p.asked) != 1 || p.asked[0] != "GRASS" {
		t.Fatalf("only the timed coin event, its first coin: %v", p.asked)
	}

	if len(w.armed) != 0 || len(s.pending) != 1 || s.pending[0].Kind != emptyKind {
		t.Errorf("a risky plan alone is not armed and the feed says so: %+v, %+v", w.armed, s.pending)
	}
}

func TestAFailedQuestionIsAskedOnceMoreThenToldAsAFailure(t *testing.T) {
	now := time.Date(2026, 11, 19, 4, 0, 0, 0, time.UTC)
	unlock := schedule.Entry{ID: "unlock:KAITO", Time: now.Add(10 * time.Minute), Title: "Розблокування", Kind: schedule.Unlock, Assets: []string{"KAITO"}}
	failing := &planner{err: errors.New("модель мовчить")}
	w := &watcher{}
	s := service(failing, w, &store{}, unlock)
	_ = s.Switch(true)

	s.Tick(context.Background(), now)

	if len(failing.asked) != 1 || len(s.pending) != 0 {
		t.Fatalf("the first failure is kept quiet for a second try: asked %v, told %+v", failing.asked, s.pending)
	}

	s.Tick(context.Background(), now.Add(30*time.Second))
	s.Tick(context.Background(), now.Add(time.Minute))

	if len(failing.asked) != 2 || len(w.armed) != 0 {
		t.Fatalf("asked twice and no more: %v", failing.asked)
	}

	if len(s.pending) != 1 || s.pending[0].Kind != failedKind || !s.pending[0].Silent {
		t.Errorf("the second failure is told quietly in the feed: %+v", s.pending)
	}
}

type held struct {
	started chan string
	release chan struct{}
}

func (h held) Event(_ context.Context, asset string, _ schedule.Entry) (research.Plan, error) {
	h.started <- asset
	<-h.release

	return research.Plan{}, nil
}

func TestEventsOfOneMinuteArePlannedSideBySide(t *testing.T) {
	now := time.Date(2026, 11, 15, 23, 50, 0, 0, time.UTC)
	at := now.Add(10 * time.Minute)
	entries := []schedule.Entry{
		{ID: "unlock:SIGN", Time: at, Title: "Розблокування", Kind: schedule.Unlock, Assets: []string{"SIGN"}},
		{ID: "unlock:LIGHT", Time: at, Title: "Розблокування", Kind: schedule.Unlock, Assets: []string{"LIGHT"}},
	}
	h := held{started: make(chan string, 2), release: make(chan struct{})}
	s := NewService(Deps{
		Planner: h, Shelf: &shelf{}, Watcher: &watcher{}, Agenda: agenda(entries), Store: &store{}, Lead: 10 * time.Minute,
		AtOnce: 2, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, State{Enabled: true})

	done := make(chan struct{})

	go func() {
		s.Tick(context.Background(), now)
		close(done)
	}()

	for range entries {
		select {
		case <-h.started:
		case <-time.After(2 * time.Second):
			t.Fatal("the second event waited for the first one's answer")
		}
	}

	close(h.release)
	<-done

	if len(s.pending) != 2 || !s.pending[0].Time.After(now) {
		t.Errorf("both events are told, stamped when their plans were ready: %+v", s.pending)
	}
}
