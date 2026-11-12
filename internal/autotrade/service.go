package autotrade

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
)

const (
	sourceName = "autotrade"
	armedKind  = "автотрейдинг · план на стеженні"
	emptyKind  = "автотрейдинг · без плану"
	failedKind = "автотрейдинг · помилка"
	forgetIn   = 24 * time.Hour
)

type Planner interface {
	Event(ctx context.Context, asset string, entry schedule.Entry) (research.Plan, error)
}

type Shelf interface {
	FixPlan(asset string, plan research.Plan) error
}

type Watcher interface {
	Arm(asset string, plan research.Scenario) error
}

type Agenda interface {
	Upcoming(now time.Time, within time.Duration) []schedule.Entry
}

type Store interface {
	Save(state State) error
}

type Deps struct {
	Planner Planner
	Shelf   Shelf
	Watcher Watcher
	Agenda  Agenda
	Store   Store
	Lead    time.Duration
	Every   time.Duration
	AtOnce  int
	Log     *slog.Logger
}

type Service struct {
	deps    Deps
	mu      sync.Mutex
	state   State
	failed  map[string]bool
	pending []news.Event
}

func NewService(deps Deps, saved State) *Service {
	if saved.Planned == nil {
		saved.Planned = map[string]time.Time{}
	}

	return &Service{deps: deps, state: saved, failed: map[string]bool{}}
}

func (s *Service) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.Enabled
}

func (s *Service) Switch(on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.state
	next.Enabled = on

	if err := s.deps.Store.Save(next); err != nil {
		return err
	}

	s.state = next

	return nil
}

func (s *Service) Run(ctx context.Context, out chan<- news.Event) {
	ticker := time.NewTicker(s.deps.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx, time.Now().UTC())
			s.flush(ctx, out)
		}
	}
}

func (s *Service) Tick(ctx context.Context, now time.Time) {
	if !s.Enabled() {
		return
	}

	var due []schedule.Entry

	for _, entry := range s.deps.Agenda.Upcoming(now, s.deps.Lead) {
		if Planned(entry) && s.claim(entry.ID, entry.Time, now) {
			due = append(due, entry)
		}
	}

	slots := make(chan struct{}, max(s.deps.AtOnce, 1))

	var wg sync.WaitGroup

	for _, entry := range due {
		wg.Go(func() {
			slots <- struct{}{}

			defer func() {
				<-slots
			}()

			s.prepare(ctx, entry, now)
		})
	}

	wg.Wait()
}

func (s *Service) prepare(ctx context.Context, entry schedule.Entry, began time.Time) {
	asset := entry.Assets[0]
	started := time.Now()

	plan, err := s.deps.Planner.Event(ctx, asset, entry)
	now := began.Add(time.Since(started))

	if err != nil {
		s.deps.Log.Warn("auto plan failed", "asset", asset, "err", err)

		if !s.retried(entry.ID) {
			s.tell(notice(entry, failedKind, "ризик-менеджер двічі не відповів: "+err.Error(), now))
		}

		return
	}

	if err := s.deps.Shelf.FixPlan(asset, plan); err != nil {
		s.deps.Log.Warn("auto plan not kept", "asset", asset, "err", err)
	}

	scenario, found := Safest(plan)
	if !found {
		s.tell(notice(entry, emptyKind, "ризик-менеджер не дав плану з числами", now))

		return
	}

	scenario = Timed(scenario, entry)

	if err := s.deps.Watcher.Arm(asset, scenario); err != nil {
		s.tell(notice(entry, emptyKind, "план не став на стеження: "+err.Error(), now))

		return
	}

	s.tell(notice(entry, armedKind, scenario.Condition, now))
}

func (s *Service) claim(id string, at, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, planned := s.state.Planned[id]; planned {
		return false
	}

	for id, at := range s.state.Planned {
		if now.Sub(at) > forgetIn {
			delete(s.state.Planned, id)
			delete(s.failed, id)
		}
	}

	s.state.Planned[id] = at

	if err := s.deps.Store.Save(s.state); err != nil {
		s.deps.Log.Warn("save auto trading state", "err", err)
	}

	return true
}

func (s *Service) retried(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failed[id] {
		return false
	}

	s.failed[id] = true
	delete(s.state.Planned, id)

	if err := s.deps.Store.Save(s.state); err != nil {
		s.deps.Log.Warn("save auto trading state", "err", err)
	}

	return true
}

func (s *Service) tell(e news.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending = append(s.pending, e)
}

func (s *Service) flush(ctx context.Context, out chan<- news.Event) {
	s.mu.Lock()
	events := s.pending
	s.pending = nil
	s.mu.Unlock()

	for _, e := range events {
		select {
		case out <- e:
		case <-ctx.Done():
			return
		}
	}
}

func notice(entry schedule.Entry, kind, detail string, now time.Time) news.Event {
	return news.Event{
		ID:       "auto:" + entry.ID,
		Source:   sourceName,
		Category: news.Trade,
		Kind:     kind,
		Title:    entry.Assets[0] + ": " + entry.Title,
		Time:     now,
		Symbols:  []news.Symbol{news.Symbol(entry.Assets[0])},
		Severity: news.Normal,
		Details:  []string{detail, "подія " + entry.Time.UTC().Format("02.01 15:04 UTC")},
		Silent:   true,
	}
}
