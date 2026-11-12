package autopilot

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/trading"
)

type fixedPrice float64

func (p fixedPrice) Ticker(context.Context, news.Symbol) (market.Ticker, bool, error) {
	return market.Ticker{Contract: "ZECUSDT", Price: float64(p)}, true, nil
}

func (p fixedPrice) Extremes(context.Context, string, time.Time) (float64, float64, bool) {
	return 0, 0, false
}

type scriptedJudge struct {
	verdicts []Verdict
	asked    int
	causes   []Cause
}

func (j *scriptedJudge) Judge(_ context.Context, _ Watch, _ float64, cause Cause) (Verdict, error) {
	j.asked++
	j.causes = append(j.causes, cause)

	if len(j.verdicts) == 0 {
		return Verdict{}, errors.New("мовчить")
	}

	verdict := j.verdicts[0]
	j.verdicts = j.verdicts[1:]

	return verdict, nil
}

type desk struct {
	book    trading.Book
	placed  []trading.Draft
	refusal error
}

func (d *desk) Book() trading.Book {
	return d.book
}

func (d *desk) Place(_ context.Context, draft trading.Draft) error {
	if d.refusal != nil {
		return d.refusal
	}

	d.placed = append(d.placed, draft)

	return nil
}

type memory struct {
	saved Watches
}

func (m *memory) Save(ws Watches) error {
	m.saved = ws

	return nil
}

func service(price float64, judge *scriptedJudge, d *desk) *Service {
	return NewService(Deps{
		Prices:   fixedPrice(price),
		Events:   &feed{},
		Judge:    judge,
		Desk:     d,
		Store:    &memory{},
		Every:    time.Hour,
		Lifetime: 24 * time.Hour,
		Rest:     15 * time.Minute,
		Hold:     time.Hour,
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Watches{})
}

func state(t *testing.T, s *Service) Watch {
	t.Helper()

	return s.Of("ZEC")[0]
}

func TestAPlanSleepsUntilItsPriceThenPlacesAnOrderOnAYes(t *testing.T) {
	judge := &scriptedJudge{verdicts: []Verdict{{Decision: Confirmed, Why: "закрилась вище"}}}
	d := &desk{book: trading.Book{}.Funded(1000, 0)}
	asleep := service(1100, judge, d)

	if _, err := asleep.Arm("zec", longPlan); err != nil {
		t.Fatal(err)
	}

	asleep.Tick(context.Background(), noon)

	if judge.asked != 0 || state(t, asleep).State != Armed {
		t.Fatalf("a sleeping plan was judged: %+v", state(t, asleep))
	}

	awake := service(1125, judge, d)
	_, _ = awake.Arm("ZEC", longPlan)
	awake.Tick(context.Background(), time.Now().UTC())

	if got := state(t, awake); got.State != Fired || len(d.placed) != 1 || d.placed[0].Side != trading.Long || d.placed[0].Stake != 70 {
		t.Fatalf("watch %+v, placed %+v", got, d.placed)
	}

	awake.Tick(context.Background(), time.Now().UTC())

	if len(d.placed) != 1 {
		t.Error("a fired plan placed a second order")
	}
}

func TestAPlanIsDroppedOnANoAtItsPriceAndAtItsDate(t *testing.T) {
	now := time.Now().UTC()
	refused := service(1125, &scriptedJudge{verdicts: []Verdict{{Decision: Denied, Why: "новина спростована"}}}, &desk{book: trading.Book{}.Funded(1000, 0)})
	_, _ = refused.Arm("ZEC", longPlan)
	refused.Tick(context.Background(), now)

	if got := state(t, refused); got.State != Refuted || got.Note != "новина спростована" {
		t.Errorf("after a no: %+v", got)
	}

	vote := longPlan
	vote.WakeAfter = now.Format(time.RFC3339)

	dated := &scriptedJudge{verdicts: []Verdict{{Decision: Denied, Why: "імпульс уже минув"}, {Decision: Confirmed}}}
	d := &desk{book: trading.Book{}.Funded(1000, 0)}
	late := service(1125, dated, d)
	_, _ = late.Arm("ZEC", vote)

	late.Tick(context.Background(), now.Add(time.Minute))
	late.Tick(context.Background(), now.Add(2*time.Hour))

	if got := state(t, late); dated.asked != 1 || got.State != Refuted || len(late.Of("ZEC")) != 1 || len(d.placed) != 0 {
		t.Errorf("a dated plan is asked once and taken off on a no: asked %d, %+v", dated.asked, got)
	}
}

func TestAPlanNeverEntersWhenTheOrderIsRefusedOrItIsCancelled(t *testing.T) {
	now := time.Now().UTC()
	blocked := service(1125, &scriptedJudge{verdicts: []Verdict{{Decision: Confirmed}}}, &desk{book: trading.Book{}.Funded(1000, 0), refusal: errors.New("експозиція")})
	_, _ = blocked.Arm("ZEC", longPlan)
	blocked.Tick(context.Background(), now)

	if got := state(t, blocked); got.State != Refuted {
		t.Errorf("an order the desk refused: %+v", got)
	}

	cancelled := service(1125, &scriptedJudge{}, &desk{book: trading.Book{}.Funded(1000, 0)})
	w, _ := cancelled.Arm("ZEC", longPlan)

	if armed := cancelled.Armed(); len(armed) != 1 || armed[0].ID != w.ID || len(blocked.Armed()) != 0 {
		t.Errorf("only a plan still watched is an active trigger: %+v, %+v", armed, blocked.Armed())
	}

	if err := cancelled.Cancel(w.ID); err != nil || state(t, cancelled).State != Cancelled || len(cancelled.Armed()) != 0 {
		t.Errorf("cancel: %v, %+v", err, state(t, cancelled))
	}

	if err := cancelled.Cancel(w.ID); !errors.Is(err, ErrNoWatch) {
		t.Errorf("a second cancel: %v", err)
	}

	late := service(1125, &scriptedJudge{}, &desk{book: trading.Book{}.Funded(1000, 0)})
	_, _ = late.Arm("ZEC", longPlan)
	late.Tick(context.Background(), now.Add(48*time.Hour))

	if state(t, late).State != Expired {
		t.Errorf("past its term: %+v", state(t, late))
	}
}

type feed struct {
	events []news.Event
}

func (f *feed) Latest(int) []news.Event {
	return f.events
}

func TestAPlanThatWaitsForADateOrAnAnnouncementIsAskedThenAndNotBefore(t *testing.T) {
	now := time.Now().UTC()
	vote := longPlan
	vote.WakeAfter = now.Add(48 * time.Hour).Format(time.RFC3339)

	judge := &scriptedJudge{verdicts: []Verdict{{Decision: Confirmed, Why: "рішення позитивне"}}}
	d := &desk{book: trading.Book{}.Funded(1000, 0)}
	s := service(1125, judge, d)
	w, _ := s.Arm("ZEC", vote)

	if !w.Expires.After(now.Add(48 * time.Hour)) {
		t.Errorf("a plan that waits two days lapses too early: %v", w.Expires)
	}

	s.Tick(context.Background(), now.Add(time.Hour))

	if judge.asked != 0 {
		t.Fatal("asked before the plan's date")
	}

	s.Tick(context.Background(), now.Add(49*time.Hour))

	if judge.asked != 1 || state(t, s).State != Fired || len(d.placed) != 1 {
		t.Errorf("at the plan's date: asked %d, %+v", judge.asked, state(t, s))
	}

	quiet := &scriptedJudge{verdicts: []Verdict{{Decision: Denied, Why: "делістинг"}}}
	news_ := &feed{}
	below := NewService(Deps{
		Prices: fixedPrice(1000), Events: news_, Judge: quiet, Desk: d, Store: &memory{}, Every: time.Hour,
		Lifetime: 24 * time.Hour, Rest: 15 * time.Minute, Hold: time.Hour,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Watches{})
	_, _ = below.Arm("ZEC", longPlan)
	below.Tick(context.Background(), now)

	if quiet.asked != 0 {
		t.Fatal("asked below the threshold with no announcement")
	}

	news_.events = []news.Event{
		{Category: news.Trade, Symbols: []news.Symbol{"ZEC"}, Time: now.Add(time.Minute)},
		{Category: news.Reminder, Symbols: []news.Symbol{"ZEC"}, Time: now.Add(time.Minute)},
	}
	below.Tick(context.Background(), now.Add(90*time.Second))

	if quiet.asked != 0 {
		t.Fatal("a trade or a reminder roused the plan")
	}

	news_.events = []news.Event{
		{Category: news.ShortTrigger, Symbols: []news.Symbol{"ZEC"}, Time: now.Add(time.Minute)},
	}
	below.Tick(context.Background(), now.Add(2*time.Minute))

	if quiet.asked != 1 || below.Of("ZEC")[0].State != Refuted {
		t.Errorf("an announcement about the coin did not rouse the plan: %+v", below.Of("ZEC")[0])
	}
}

func TestAPlanWithADateSleepsUntilItAndThenWakesAtItsPrice(t *testing.T) {
	now := time.Now().UTC()
	vote := longPlan
	vote.WakeAfter = now.Add(48 * time.Hour).Format(time.RFC3339)

	judge := &scriptedJudge{verdicts: []Verdict{{Decision: Confirmed, Why: "рішення позитивне"}}}
	d := &desk{book: trading.Book{}.Funded(1000, 0)}
	s := service(1125, judge, d)

	if _, err := s.Arm("ZEC", vote); err != nil {
		t.Fatal(err)
	}

	dateless := vote
	dateless.WakeAbove = 0

	if _, err := s.Arm("ZEC", dateless); err == nil {
		t.Fatal("a plan with a date and no price threshold was set to be watched")
	}

	s.Tick(context.Background(), now.Add(47*time.Hour))

	if judge.asked != 0 {
		t.Fatal("asked before the plan's date")
	}

	s.Tick(context.Background(), now.Add(49*time.Hour))
	s.Tick(context.Background(), now.Add(49*time.Hour+time.Minute))

	if got := state(t, s); judge.asked != 1 || judge.causes[0] != ByPrice || got.State != Fired || len(d.placed) != 1 {
		t.Fatalf("the price past the date called the agent once and a yes placed the order: asked %d, %+v", judge.asked, got)
	}
}

func TestACheckTheAgentFailedIsToldWithTheNextTry(t *testing.T) {
	now := time.Now().UTC()
	judge := &scriptedJudge{}
	s := service(1125, judge, &desk{book: trading.Book{}.Funded(1000, 0)})
	vote := longPlan
	vote.WakeAfter = now.Format(time.RFC3339)
	_, _ = s.Arm("ZEC", vote)

	s.Tick(context.Background(), now.Add(time.Minute))

	out := make(chan news.Event, 4)
	s.flush(context.Background(), out)
	close(out)

	var kinds []string
	var told news.Event

	for e := range out {
		kinds = append(kinds, e.Kind)
		told = e
	}

	if len(kinds) != 1 || kinds[0] != "план · "+checkedName {
		t.Fatalf("events %v", kinds)
	}

	if len(told.Details) != 4 || !strings.HasPrefix(told.Details[1], "перевірка 1: ") || !strings.HasPrefix(told.Details[2], "агент не відповів") || !strings.HasPrefix(told.Details[3], "наступна перевірка не раніше") {
		t.Errorf("a failed check told %+v", told.Details)
	}
}

func TestTheOwnerAmendsAWatchedPlansNumbersButNotItsCondition(t *testing.T) {
	s := service(1100, &scriptedJudge{}, &desk{book: trading.Book{}.Funded(1000, 0)})
	w, _ := s.Arm("ZEC", longPlan)

	numbers := research.Scenario{Condition: "інша умова", Direction: research.DirectionShort, WakeAbove: 1130, SizePct: 4, Stop: 1060, Take: 1300, WakeAfter: "2099-01-01"}

	amended, err := s.Amend(w.ID, numbers)
	if err != nil {
		t.Fatal(err)
	}

	plan := amended.Plan

	if plan.Condition != longPlan.Condition || plan.Direction != research.DirectionLong || plan.WakeAbove != 1130 || plan.SizePct != 4 {
		t.Errorf("amended plan %+v", plan)
	}

	if plan.Action != "лонг від 1130 не раніше 2099-01-01, 4% рахунку, стоп 1060, тейк 1300" || !amended.After.Equal(research.Moment("2099-01-01")) {
		t.Errorf("amended terms %q, after %v", plan.Action, amended.After)
	}

	if _, err := s.Amend(w.ID, research.Scenario{WakeAbove: 1130, SizePct: 4, Stop: 1400, Take: 1300}); err == nil {
		t.Error("a stop above the take passed for a long")
	}

	if state(t, s).Plan.Stop != 1060 {
		t.Error("a refused amendment changed the plan")
	}

	_ = s.Cancel(w.ID)

	if _, err := s.Amend(w.ID, numbers); !errors.Is(err, ErrNoWatch) {
		t.Errorf("amended a cancelled plan: %v", err)
	}
}

func TestSettingAndChangingAWatchTellsTheFeedNothing(t *testing.T) {
	s := service(1100, &scriptedJudge{}, &desk{book: trading.Book{}.Funded(1000, 0)})
	w, _ := s.Arm("ZEC", longPlan)

	_, _ = s.Amend(w.ID, research.Scenario{WakeAbove: 1130, SizePct: 4, Stop: 1060, Take: 1300})

	out := make(chan news.Event, 4)
	s.flush(context.Background(), out)
	close(out)

	if len(out) != 0 || len(s.Armed()) != 1 {
		t.Errorf("a watch set and changed is in the triggers, not in the feed: %d notices", len(out))
	}
}

func TestAnEntryTakesTheContractsOtherPlansOff(t *testing.T) {
	now := time.Now().UTC()
	judge := &scriptedJudge{verdicts: []Verdict{{Decision: Confirmed}, {Decision: Confirmed}}}
	placed := &desk{book: trading.Book{}.Funded(1000, 0)}
	s := service(1125, judge, placed)

	other := longPlan
	other.Condition = "інший вхід за тією самою ціною"

	first, _ := s.Arm("ZEC", longPlan)
	second, _ := s.Arm("ZEC", other)
	_, _ = s.Arm("ZEC", research.Scenario{Condition: "ручний", Action: "лонг", Risk: research.RiskMinimal, Direction: research.DirectionLong, WakeAbove: 2000, SizePct: 1, Stop: 1900, Take: 2200})

	s.Tick(context.Background(), now)

	states := map[string]State{}

	for _, w := range s.Of("ZEC") {
		states[w.ID] = w.State
	}

	if states[first.ID] != Fired || states[second.ID] != Cancelled || len(s.Armed()) != 0 {
		t.Fatalf("one entry, the contract's other plans off: %v", states)
	}

	if len(placed.placed) != 1 || judge.asked != 1 {
		t.Errorf("one order, the judge asked once: %+v, asked %d", placed.placed, judge.asked)
	}

	if tally := s.Tally(); tally.Fired != 1 || tally.Cancelled != 2 || tally.Armed != 0 {
		t.Errorf("every plan is counted, the owner's own too: %+v", tally)
	}
}

func TestTheSettledWatchesAreListedNewestFirstWithoutTheArmed(t *testing.T) {
	now := time.Now().UTC()
	s := service(1125, &scriptedJudge{verdicts: []Verdict{{Decision: Denied, Why: "ні"}}}, &desk{book: trading.Book{}.Funded(1000, 0)})
	_, _ = s.Arm("ZEC", longPlan)
	s.Tick(context.Background(), now)

	waiting := longPlan
	waiting.Condition, waiting.WakeAbove = "інша умова", 1200
	kept, _ := s.Arm("ZEC", waiting)
	_ = s.Cancel(kept.ID)
	_, _ = s.Arm("ZEC", waiting)

	got := s.Settled()

	if len(got) != 2 || got[0].State != Refuted || got[1].State != Cancelled {
		t.Errorf("settled: %+v", got)
	}
}

func TestAPlanWithADateAndAPriceSleepsUntilTheDateThenWaitsForItsPrice(t *testing.T) {
	now := time.Now().UTC()
	after := longPlan
	after.WakeAfter = now.Add(time.Hour).Format(time.RFC3339)

	judge := &scriptedJudge{verdicts: []Verdict{{Decision: Confirmed, Why: "пробій після події"}}}
	d := &desk{book: trading.Book{}.Funded(1000, 0)}
	prices := &moving{price: 1125}
	s := NewService(Deps{
		Prices: prices, Events: &feed{}, Judge: judge, Desk: d, Store: &memory{}, Every: time.Hour,
		Lifetime: 24 * time.Hour, Rest: 15 * time.Minute, Hold: time.Hour,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Watches{})
	_, _ = s.Arm("ZEC", after)

	s.Tick(context.Background(), now.Add(30*time.Minute))

	if judge.asked != 0 {
		t.Fatal("the price is over the threshold, but the plan's date has not come")
	}

	prices.price = 1100
	s.Tick(context.Background(), now.Add(2*time.Hour))

	if judge.asked != 0 || state(t, s).State != Armed {
		t.Fatalf("the date alone must not call the judge for a plan that waits for its price: asked %d", judge.asked)
	}

	prices.price = 1125
	s.Tick(context.Background(), now.Add(3*time.Hour))

	if judge.asked != 1 || judge.causes[0] != ByPrice || state(t, s).State != Fired {
		t.Errorf("after its date the plan wakes at its price: asked %d, %+v", judge.asked, state(t, s))
	}
}

type moving struct {
	price float64
}

func (m *moving) Ticker(context.Context, news.Symbol) (market.Ticker, bool, error) {
	return market.Ticker{Contract: "ZECUSDT", Price: m.price}, true, nil
}

func (m *moving) Extremes(context.Context, string, time.Time) (float64, float64, bool) {
	return 0, 0, false
}

type heldJudge struct {
	started chan string
	release chan struct{}
}

func (j heldJudge) Judge(_ context.Context, w Watch, _ float64, _ Cause) (Verdict, error) {
	j.started <- w.Asset
	<-j.release

	return Verdict{Decision: Denied, Why: "умова не виконана"}, nil
}

type anyPrice struct{}

func (anyPrice) Ticker(_ context.Context, s news.Symbol) (market.Ticker, bool, error) {
	return market.Ticker{Contract: string(s) + "USDT", Price: 1125}, true, nil
}

func (anyPrice) Extremes(context.Context, string, time.Time) (float64, float64, bool) {
	return 0, 0, false
}

func TestPlansOfDifferentContractsAreJudgedSideBySide(t *testing.T) {
	judge := heldJudge{started: make(chan string, 2), release: make(chan struct{})}
	s := NewService(Deps{
		Prices: anyPrice{}, Events: &feed{}, Judge: judge, Desk: &desk{book: trading.Book{}.Funded(1000, 0)},
		Store: &memory{}, Lifetime: 7 * 24 * time.Hour, Rest: 15 * time.Minute, AtOnce: 2,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Watches{})

	_, _ = s.Arm("ZEC", longPlan)
	_, _ = s.Arm("ETH", longPlan)

	done := make(chan struct{})

	go func() {
		s.Tick(context.Background(), time.Now().UTC())
		close(done)
	}()

	for range 2 {
		select {
		case <-judge.started:
		case <-time.After(2 * time.Second):
			t.Fatal("the second contract waited for the first one's verdict")
		}
	}

	close(judge.release)
	<-done

	if tally := s.Tally(); tally.Refuted != 2 {
		t.Errorf("both verdicts are kept: %+v", tally)
	}
}

type slowJudge struct {
	took time.Duration
}

func (j slowJudge) Judge(context.Context, Watch, float64, Cause) (Verdict, error) {
	time.Sleep(j.took)

	return Verdict{Decision: Denied, Why: "умова не виконана"}, nil
}

func TestWhatBecameOfAPlanIsStampedWithTheMomentOfTheVerdict(t *testing.T) {
	now := time.Now().UTC()
	s := NewService(Deps{
		Prices: fixedPrice(1125), Events: &feed{}, Judge: slowJudge{took: 50 * time.Millisecond},
		Desk: &desk{book: trading.Book{}.Funded(1000, 0)}, Store: &memory{}, Lifetime: 7 * 24 * time.Hour,
		Rest: 15 * time.Minute, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Watches{})

	_, _ = s.Arm("ZEC", longPlan)

	s.Tick(context.Background(), now)

	if len(s.pending) != 1 || s.pending[0].Time.Sub(now) < 50*time.Millisecond {
		t.Errorf("the notice is stamped when the judge answered, not when the check began: %+v", s.pending)
	}
}

type wick struct {
	price     float64
	low, high float64
	asked     []time.Time
}

func (w *wick) Ticker(context.Context, news.Symbol) (market.Ticker, bool, error) {
	return market.Ticker{Contract: "CUSDT", Price: w.price}, true, nil
}

func (w *wick) Extremes(_ context.Context, _ string, since time.Time) (float64, float64, bool) {
	w.asked = append(w.asked, since)

	return w.low, w.high, w.low > 0
}

func TestAPriceThatOnlyTouchedThePlansThresholdBetweenTwoLooksStillWakesIt(t *testing.T) {
	now := time.Now().UTC()
	prices := &wick{price: 0.0690}
	judge := &scriptedJudge{verdicts: []Verdict{{Decision: Denied, Why: "обсягу немає"}}}
	s := NewService(Deps{
		Prices: prices, Events: &feed{}, Judge: judge, Desk: &desk{book: trading.Book{}.Funded(1000, 0)}, Store: &memory{},
		Lifetime: 7 * 24 * time.Hour, Rest: 15 * time.Minute, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Watches{})
	short := research.Scenario{
		Condition: "пробій мінімуму доби", Action: "шорт", Risk: research.RiskMinimal, Direction: research.DirectionShort,
		WakeBelow: 0.0686, SizePct: 1, Stop: 0.0714, Take: 0.0658,
	}

	watch, _ := s.ArmFor("C", short, 48*time.Hour)

	if left := watch.Expires.Sub(watch.Armed); left != 48*time.Hour {
		t.Fatalf("a plan armed for an event lapses in its own time: %v", left)
	}

	s.Tick(context.Background(), now.Add(time.Minute))

	if judge.asked != 0 || len(prices.asked) != 1 || prices.asked[0].Before(watch.Armed) {
		t.Fatalf("no touch yet, and nothing before the plan was armed counts: asked %d, since %v", judge.asked, prices.asked)
	}

	prices.low, prices.high = 0.06842, 0.0692

	s.Tick(context.Background(), now.Add(2*time.Minute))

	if judge.asked != 1 || judge.causes[0] != ByPrice {
		t.Errorf("a wick through the threshold wakes the plan: asked %d, %v", judge.asked, judge.causes)
	}
}
