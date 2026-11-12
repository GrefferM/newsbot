package monitor

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"newsbot/internal/news"
)

const (
	sourceStagger  = 3 * time.Second
	persistEvery   = 30 * time.Second
	seenRetention  = 90 * 24 * time.Hour
	outboxCapacity = 200
)

type ServiceDeps struct {
	Collector  *Collector
	Dispatcher *Dispatcher
	Watchlist  Runner
	Trader     Trader
	Seen       SeenRepository
	Log        *slog.Logger
}

type Service struct {
	collector  *Collector
	dispatcher *Dispatcher
	watchlist  Runner
	trader     Trader
	seen       SeenRepository
	log        *slog.Logger
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		collector:  deps.Collector,
		dispatcher: deps.Dispatcher,
		watchlist:  deps.Watchlist,
		trader:     deps.Trader,
		seen:       deps.Seen,
		log:        deps.Log,
	}
}

func (s *Service) Run(ctx context.Context, sources []Source, reporters ...Reporter) {
	outbox := make(chan news.Event, outboxCapacity)

	var wg sync.WaitGroup

	wg.Go(func() {
		s.dispatcher.Run(ctx, outbox)
	})

	wg.Go(func() {
		s.trader.Run(ctx, outbox)
	})

	wg.Go(func() {
		s.persist(ctx)
	})

	if s.watchlist != nil {
		wg.Go(func() {
			s.watchlist.Run(ctx)
		})
	}

	for _, reporter := range reporters {
		wg.Go(func() {
			reporter.Run(ctx, outbox)
		})
	}

	for i, src := range sources {
		wg.Go(func() {
			s.collector.Run(ctx, src, time.Duration(i)*sourceStagger, outbox)
		})
	}

	wg.Wait()

	s.save()
}

func (s *Service) Once(ctx context.Context, sources []Source, limit int) {
	for _, src := range sources {
		events, err := src.Poll(ctx)
		if err != nil {
			s.log.Warn("poll failed", "source", src.Name(), "err", err)

			continue
		}

		s.log.Info("polled", "source", src.Name(), "items", len(events))

		events = slices.DeleteFunc(events, isMuted)

		for _, e := range events[:min(limit, len(events))] {
			if !s.dispatcher.Dispatch(ctx, e) {
				return
			}
		}
	}
}

func isMuted(e news.Event) bool {
	return e.Muted != ""
}

func (s *Service) persist(ctx context.Context) {
	ticker := time.NewTicker(persistEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.seen.Prune(seenRetention)
			s.save()
		}
	}
}

func (s *Service) save() {
	if err := s.seen.Save(); err != nil {
		s.log.Error("save state", "err", err)
	}
}
