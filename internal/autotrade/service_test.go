package autotrade

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/trading"
)

type listed map[string]bool

func (l listed) CryptoContracts(context.Context) map[string]bool {
	return l
}

type desk struct {
	book trading.Book
}

func (d desk) Book() trading.Book {
	return d.book
}

type memory struct {
	saved []State
	err   error
}

func (m *memory) Save(state State) error {
	if m.err != nil {
		return m.err
	}

	m.saved = append(m.saved, state)

	return nil
}

type sample struct {
	take  string
	seen  []market.Tick
	armed []bool
}

func (s *sample) describe() Strategy {
	return Strategy{ID: "sample", Name: "Зразок", Terms: []Term{{Name: "Тейк", Value: s.take}}, Rule: s.take}
}

func (s *sample) retune(raw []byte) ([]byte, func(), error) {
	take := strings.TrimSpace(string(raw))
	if take == "" {
		return nil, nil, errors.New("порожнє правило")
	}

	return []byte(take), func() {
		s.take = take
	}, nil
}

func (s *sample) observe(_ context.Context, ticks []market.Tick, on bool) {
	s.seen = append(s.seen, ticks...)
	s.armed = append(s.armed, on)
}

func serviceWith(store *memory, book trading.Book, strategies ...strategy) *Service {
	s := NewService(Deps{
		Market: listed{"MOVEUSDT": true},
		Trader: desk{book: book},
		Store:  store,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, State{})

	s.strategies = strategies
	s.List(context.Background())

	return s
}

func TestWithoutStrategiesNothingIsListedSwitchedOrTuned(t *testing.T) {
	store := &memory{}
	s := serviceWith(store, trading.Book{})

	if listed := s.Strategies(); len(listed) != 0 {
		t.Errorf("listed %+v", listed)
	}

	if err := s.Switch("none", true); !errors.Is(err, ErrNoStrategy) {
		t.Errorf("switch: %v", err)
	}

	if err := s.Tune("none", []byte(`{}`)); !errors.Is(err, ErrNoStrategy) {
		t.Errorf("tune: %v", err)
	}

	if len(store.saved) != 0 {
		t.Errorf("saved %+v", store.saved)
	}
}

func TestAStrategyIsListedWithItsSwitchAndItsTrades(t *testing.T) {
	book := trading.Book{Positions: []trading.Position{
		{ID: "A", Strategy: "sample", ClosedAt: time.Unix(1, 0), PnL: 2.5},
		{ID: "B", Strategy: "other", ClosedAt: time.Unix(1, 0), PnL: -1},
	}}
	store := &memory{}
	s := serviceWith(store, book, &sample{take: "6%"})

	if err := s.Switch("sample", true); err != nil {
		t.Fatal(err)
	}

	listed := s.Strategies()

	if len(listed) != 1 || !listed[0].On || listed[0].Name != "Зразок" || listed[0].Terms[0].Value != "6%" || listed[0].Record.PnL != 2.5 {
		t.Errorf("listed %+v", listed)
	}

	if len(store.saved) != 1 || !store.saved[0].On["sample"] {
		t.Errorf("saved %+v", store.saved)
	}
}

func TestAStrategySeesOnlyCryptoPricesAndItsSwitch(t *testing.T) {
	watched := &sample{take: "6%"}
	s := serviceWith(&memory{}, trading.Book{}, watched)

	ticks := []market.Tick{{Contract: "MOVEUSDT", Price: 1}, {Contract: "GOLDUSDT", Price: 2}}

	s.Observe(context.Background(), ticks)

	if err := s.Switch("sample", true); err != nil {
		t.Fatal(err)
	}

	s.Observe(context.Background(), ticks)
	s.Observe(context.Background(), ticks[1:])

	if len(watched.seen) != 2 || watched.seen[0].Contract != "MOVEUSDT" || len(watched.armed) != 2 || watched.armed[0] || !watched.armed[1] {
		t.Errorf("seen %+v, armed %v", watched.seen, watched.armed)
	}
}

func TestATunedRuleIsAppliedAndSavedOnlyWhenTheStrategyTakesIt(t *testing.T) {
	tuned := &sample{take: "6%"}
	store := &memory{}
	s := serviceWith(store, trading.Book{}, tuned)

	if err := s.Tune("sample", []byte(" ")); err == nil {
		t.Fatal("an empty rule was taken")
	}

	if tuned.take != "6%" || len(store.saved) != 0 {
		t.Fatalf("a refused rule changed something: %q, %+v", tuned.take, store.saved)
	}

	if err := s.Tune("sample", []byte("8%")); err != nil {
		t.Fatal(err)
	}

	if tuned.take != "8%" || len(store.saved) != 1 || string(store.saved[0].Rules["sample"]) != "8%" {
		t.Errorf("take %q, saved %+v", tuned.take, store.saved)
	}

	store.err = errors.New("диск повний")

	if err := s.Tune("sample", []byte("9%")); err == nil || tuned.take != "8%" {
		t.Errorf("a rule that was not saved is applied: %v, %q", err, tuned.take)
	}
}
