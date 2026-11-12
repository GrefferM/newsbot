package autopilot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

const (
	venue        = "plans"
	recentEvents = 100
	touchWindow  = 3 * time.Minute
)

var ErrNoWatch = errors.New("такого стеження немає")

type Prices interface {
	Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error)
	Extremes(ctx context.Context, contract string, since time.Time) (low, high float64, found bool)
}

type Events interface {
	Latest(n int) []news.Event
}

type Judge interface {
	Judge(ctx context.Context, w Watch, price float64, cause Cause) (Verdict, error)
}

type Desk interface {
	Book() trading.Book
	Place(ctx context.Context, draft trading.Draft) error
}

type Store interface {
	Save(ws Watches) error
}

type Deps struct {
	Prices   Prices
	Events   Events
	Judge    Judge
	Desk     Desk
	Store    Store
	Every    time.Duration
	Lifetime time.Duration
	Rest     time.Duration
	Hold     time.Duration
	AtOnce   int
	Log      *slog.Logger
}

type Service struct {
	deps    Deps
	mu      sync.Mutex
	watches Watches
	pending []news.Event
}

func NewService(deps Deps, saved Watches) *Service {
	return &Service{deps: deps, watches: saved}
}

func (s *Service) Of(asset string) []Watch {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.watches.Of(watchlist.Normalize(asset))
}

func (s *Service) Armed() []Watch {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := []Watch{}

	for _, w := range s.watches.Items {
		if w.State == Armed {
			out = append(out, w)
		}
	}

	return out
}

func (s *Service) Settled() []Watch {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := []Watch{}

	for _, w := range s.watches.Items {
		if w.State != Armed {
			out = append(out, w)
		}
	}

	slices.SortStableFunc(out, func(a, b Watch) int {
		return b.Checked.Compare(a.Checked)
	})

	return out
}

func (s *Service) Tally() Tally {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.watches.Tally()
}

func (s *Service) Arm(asset string, plan research.Scenario) (Watch, error) {
	return s.ArmFor(asset, plan, s.deps.Lifetime)
}

func (s *Service) ArmFor(asset string, plan research.Scenario, life time.Duration) (Watch, error) {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	next, w, err := s.watches.Arm(newID(), watchlist.Normalize(asset), plan, now, life)
	if err != nil {
		return Watch{}, err
	}

	if err := s.keep(next); err != nil {
		return Watch{}, err
	}

	return w, nil
}

func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, found := s.watches.Find(id)
	if !found || w.State != Armed {
		return ErrNoWatch
	}

	return s.keep(s.watches.With(w.Settled(Cancelled, "знято власником")))
}

func (s *Service) Amend(id string, numbers research.Scenario) (Watch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, found := s.watches.Find(id)
	if !found || w.State != Armed {
		return Watch{}, ErrNoWatch
	}

	amended, err := w.Amended(numbers, s.deps.Lifetime)
	if err != nil {
		return Watch{}, err
	}

	if err := s.keep(s.watches.With(amended)); err != nil {
		return Watch{}, err
	}

	return amended, nil
}

func (s *Service) keep(next Watches) error {
	if err := s.deps.Store.Save(next); err != nil {
		return err
	}

	s.watches = next

	return nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}
