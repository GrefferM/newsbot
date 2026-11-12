package radar

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/research"
)

const statusName = "pulse"

type Market interface {
	Pulse(ctx context.Context) ([]market.Pulse, error)
}

type Shelf interface {
	Of(asset string) []research.Finding
}

type Health interface {
	OK(name string, seen, fresh int)
	Fail(name string, err error)
}

type Deps struct {
	Coins  Market
	TradFi Market
	Rule   Rule
	Shelf  Shelf
	Health Health
	Every  time.Duration
	Log    *slog.Logger
}

type Service struct {
	deps   Deps
	mu     sync.Mutex
	board  Board
	tradfi Board
}

func NewService(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) Latest() Board {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.board
}

func (s *Service) LatestTradFi() Board {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.tradfi
}

func (s *Service) Name() string {
	return statusName
}

func (s *Service) Run(ctx context.Context, _ chan<- news.Event) {
	ticker := time.NewTicker(s.deps.Every)
	defer ticker.Stop()

	for {
		s.Refresh(ctx, time.Now().UTC())

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) Refresh(ctx context.Context, now time.Time) {
	board, err := s.snapshot(ctx, s.deps.Coins, now)
	if err != nil {
		s.deps.Health.Fail(statusName, err)
		s.deps.Log.Warn("pulse failed", "err", err)
	} else {
		s.mu.Lock()
		s.board = board
		s.mu.Unlock()

		s.deps.Health.OK(statusName, len(board.Coins), board.Movers())
	}

	if s.deps.TradFi == nil {
		return
	}

	tradfi, err := s.snapshot(ctx, s.deps.TradFi, now)
	if err != nil {
		s.deps.Log.Warn("tradfi pulse failed", "err", err)

		return
	}

	s.mu.Lock()
	s.tradfi = tradfi
	s.mu.Unlock()
}

func (s *Service) snapshot(ctx context.Context, source Market, now time.Time) (Board, error) {
	pulses, err := source.Pulse(ctx)
	if err != nil {
		return Board{}, err
	}

	board := Board{At: now, Bench: Bench(pulses), Coins: make([]Coin, 0, len(pulses))}

	for _, p := range pulses {
		coin := s.coin(p, now)

		if !p.Wide || coin.Active() || coin.Trend != nil {
			board.Coins = append(board.Coins, coin)
		}
	}

	return board, nil
}

func (s *Service) coin(p market.Pulse, now time.Time) Coin {
	coin := Coin{Pulse: p}

	if wake, woke := s.deps.Rule.Woke(p); woke {
		coin.Wake = &wake
	}

	if launch, launched := s.deps.Rule.Launched(p); launched {
		coin.Launch = &launch
	}

	if run, steady := Trend(p); steady {
		coin.Trend = &run
	}

	if coin.Active() {
		coin.Stories = Fresh(s.deps.Shelf.Of(p.Asset), now)
	}

	return coin
}
