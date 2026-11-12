package agenda

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"newsbot/internal/news"
	"newsbot/internal/schedule"
)

const statusName = "market-calendar"

type Source interface {
	Name() string
	Entries(ctx context.Context) ([]schedule.Entry, error)
}

type Health interface {
	OK(name string, seen, fresh int)
	Fail(name string, err error)
}

type Kept interface {
	Dates(zone *time.Location) []schedule.Entry
}

type Deps struct {
	Sources []Source
	Coins   Kept
	Health  Health
	Every   time.Duration
	Log     *slog.Logger
}

type Service struct {
	deps  Deps
	mu    sync.Mutex
	found map[string][]schedule.Entry
}

func NewService(deps Deps) *Service {
	return &Service{deps: deps, found: map[string][]schedule.Entry{}}
}

func (s *Service) Run(ctx context.Context, _ chan<- news.Event) {
	ticker := time.NewTicker(s.deps.Every)
	defer ticker.Stop()

	for {
		s.Refresh(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) Refresh(ctx context.Context) {
	var failed []error

	total := 0

	for _, source := range s.deps.Sources {
		entries, err := source.Entries(ctx)
		if err != nil {
			failed = append(failed, err)
			s.deps.Log.Warn("market calendar source failed", "source", source.Name(), "err", err)

			continue
		}

		total += len(entries)

		s.mu.Lock()
		s.found[source.Name()] = entries
		s.mu.Unlock()
	}

	if len(failed) > 0 {
		s.deps.Health.Fail(statusName, errors.Join(failed...))

		return
	}

	s.deps.Health.OK(statusName, total, 0)
}

func (s *Service) Upcoming(now time.Time, within time.Duration) []schedule.Entry {
	return s.Between(now, now.Add(within))
}

func (s *Service) On(day string, zone *time.Location) []schedule.Entry {
	start, err := time.ParseInLocation(time.DateOnly, day, zone)
	if err != nil {
		return nil
	}

	return s.Between(start, start.AddDate(0, 0, 1).Add(-time.Nanosecond))
}

func (s *Service) Between(from, to time.Time) []schedule.Entry {
	s.mu.Lock()

	var found []schedule.Entry

	for _, source := range s.deps.Sources {
		found = append(found, s.found[source.Name()]...)
	}

	s.mu.Unlock()

	if s.deps.Coins != nil {
		found = append(found, slices.DeleteFunc(s.deps.Coins.Dates(news.Kyiv), func(kept schedule.Entry) bool {
			return slices.ContainsFunc(found, kept.Repeats)
		})...)
	}

	out := []schedule.Entry{}

	for _, e := range found {
		if slices.ContainsFunc(found, e.Clocked) {
			continue
		}

		if !e.Time.Before(from) && !e.Time.After(to) {
			out = append(out, e)
		}
	}

	slices.SortFunc(out, func(a, b schedule.Entry) int {
		if c := a.Time.Compare(b.Time); c != 0 {
			return c
		}

		return cmp.Compare(a.ID, b.ID)
	})

	return out
}
