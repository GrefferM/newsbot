package monitor

import (
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	"newsbot/internal/news"
)

type SourceStatus struct {
	LastOK    time.Time
	LastError string
	LastErrAt time.Time
	Items     int
	NewTotal  int
}

type Status struct {
	mu      sync.Mutex
	started time.Time
	sources map[string]SourceStatus
}

func NewStatus(started time.Time) *Status {
	return &Status{started: started, sources: map[string]SourceStatus{}}
}

func (s *Status) OK(name string, items, fresh int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.sources[name]
	st.LastOK = time.Now().UTC()
	st.Items = items
	st.NewTotal += fresh
	s.sources[name] = st
}

func (s *Status) Fail(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.sources[name]
	st.LastError = err.Error()
	st.LastErrAt = time.Now().UTC()
	s.sources[name] = st
}

func (s *Status) Snapshot() (time.Time, map[string]SourceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.started, maps.Clone(s.sources)
}

var ErrNoEvent = errors.New("такої події немає")

type RecentStore interface {
	Save(events []news.Event) error
}

type Recent struct {
	mu     sync.Mutex
	limit  int
	store  RecentStore
	events []news.Event
}

func NewRecent(limit int, store RecentStore, saved []news.Event) *Recent {
	return &Recent{limit: limit, store: store, events: slices.Clone(saved[max(0, len(saved)-limit):])}
}

func (r *Recent) Add(e news.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, e)

	if len(r.events) > r.limit {
		r.events = r.events[len(r.events)-r.limit:]
	}

	return r.save()
}

func (r *Recent) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	kept := slices.DeleteFunc(slices.Clone(r.events), func(e news.Event) bool {
		return e.ID == id
	})

	if len(kept) == len(r.events) {
		return ErrNoEvent
	}

	r.events = kept

	return r.save()
}

func (r *Recent) save() error {
	if r.store == nil {
		return nil
	}

	return r.store.Save(r.events)
}

func (r *Recent) Latest(n int) []news.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := slices.Clone(r.events[max(0, len(r.events)-n):])
	slices.Reverse(out)

	return out
}
