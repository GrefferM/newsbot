package autotrade

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"sync"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/trading"
	"newsbot/internal/websocket"
)

const (
	listEvery    = time.Hour
	streamRetry  = 5 * time.Second
	streamMaxGap = 5 * time.Minute
	streamSteady = time.Minute
)

type Prices interface {
	Stream(ctx context.Context, each func([]market.Tick)) error
}

type Market interface {
	CryptoContracts(ctx context.Context) map[string]bool
}

type Trader interface {
	Book() trading.Book
}

type Store interface {
	Save(state State) error
}

type State struct {
	On    map[string]bool            `json:"on,omitempty"`
	Rules map[string]json.RawMessage `json:"rules,omitempty"`
}

type Deps struct {
	Prices Prices
	Market Market
	Trader Trader
	Store  Store
	Log    *slog.Logger
}

type Service struct {
	deps       Deps
	mu         sync.Mutex
	state      State
	crypto     map[string]bool
	strategies []strategy
}

func NewService(deps Deps, saved State) *Service {
	if saved.On == nil {
		saved.On = map[string]bool{}
	}

	return &Service{deps: deps, state: saved}
}

func (s *Service) Run(ctx context.Context, _ chan<- news.Event) {
	if len(s.strategies) == 0 {
		<-ctx.Done()

		return
	}

	var streaming sync.WaitGroup

	retry := websocket.Retry{Min: streamRetry, Max: streamMaxGap, Steady: streamSteady}

	s.List(ctx)

	streaming.Go(func() {
		retry.Keep(ctx, s.quotes, s.failed)
	})

	ticker := time.NewTicker(listEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			streaming.Wait()

			return
		case <-ticker.C:
			s.List(ctx)
		}
	}
}

func (s *Service) failed(err error) {
	s.deps.Log.Warn("price stream", "err", err)
}

func (s *Service) List(ctx context.Context) {
	crypto := s.deps.Market.CryptoContracts(ctx)
	if len(crypto) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.crypto = crypto
}

func (s *Service) quotes(ctx context.Context) error {
	return s.deps.Prices.Stream(ctx, func(ticks []market.Tick) {
		s.Observe(ctx, ticks)
	})
}

func (s *Service) Observe(ctx context.Context, ticks []market.Tick) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var traded []market.Tick

	for _, t := range ticks {
		if s.crypto[t.Contract] {
			traded = append(traded, t)
		}
	}

	if len(traded) == 0 {
		return
	}

	for _, each := range s.strategies {
		each.observe(ctx, traded, s.state.On[each.describe().ID])
	}
}

func (s *Service) Strategies() []Strategy {
	book := s.deps.Trader.Book()

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Strategy, 0, len(s.strategies))

	for _, each := range s.strategies {
		card := each.describe()
		card.On = s.state.On[card.ID]
		card.Record = book.Of(card.ID).Record()

		out = append(out, card)
	}

	return out
}

func (s *Service) named(id string) (strategy, bool) {
	for _, each := range s.strategies {
		if each.describe().ID == id {
			return each, true
		}
	}

	return nil, false
}

func (s *Service) Switch(id string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := s.named(id); !known {
		return ErrNoStrategy
	}

	next := s.state
	next.On = maps.Clone(s.state.On)
	next.On[id] = on

	if err := s.deps.Store.Save(next); err != nil {
		return err
	}

	s.state = next

	return nil
}

func (s *Service) Tune(id string, raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tuned, known := s.named(id)
	if !known {
		return ErrNoStrategy
	}

	kept, apply, err := tuned.retune(raw)
	if err != nil {
		return err
	}

	next := s.state
	next.Rules = maps.Clone(s.state.Rules)

	if next.Rules == nil {
		next.Rules = map[string]json.RawMessage{}
	}

	next.Rules[id] = kept

	if err := s.deps.Store.Save(next); err != nil {
		return err
	}

	s.state = next

	apply()

	return nil
}
