package monitor

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"newsbot/internal/news"
)

type fakeSource struct {
	mu    sync.Mutex
	polls [][]string
	calls int
}

func (f *fakeSource) Name() string {
	return "fake"
}

func (f *fakeSource) Interval() time.Duration {
	return 10 * time.Millisecond
}

func (f *fakeSource) Poll(context.Context) ([]news.Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ids := f.polls[min(f.calls, len(f.polls)-1)]
	f.calls++

	var out []news.Event

	for _, id := range ids {
		e := news.Event{ID: id, Title: "title " + id, Category: news.Incident}

		if strings.HasPrefix(id, "muted") {
			e.Muted = "не цікавий"
		}

		out = append(out, e)
	}

	return out, nil
}

type fakePublisher struct {
	mu   sync.Mutex
	sent []string
}

func (p *fakePublisher) Publish(_ context.Context, post Post) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sent = append(p.sent, post.Event.ID)

	return nil
}

func (p *fakePublisher) ids() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]string(nil), p.sent...)
}

type memorySeen struct {
	mu     sync.Mutex
	seen   map[string]bool
	seeded map[string]bool
	saves  int
}

func newMemorySeen() *memorySeen {
	return &memorySeen{seen: map[string]bool{}, seeded: map[string]bool{}}
}

func (m *memorySeen) MarkNew(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.seen[id] {
		return false
	}

	m.seen[id] = true

	return true
}

func (m *memorySeen) IsSeeded(source string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.seeded[source]
}

func (m *memorySeen) MarkSeeded(source string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seeded[source] = true
}

func (m *memorySeen) Prune(time.Duration) {}

func (m *memorySeen) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saves++

	return nil
}

type memoryJournal struct {
	mu       sync.Mutex
	outcomes map[string]string
}

func newMemoryJournal() *memoryJournal {
	return &memoryJournal{outcomes: map[string]string{}}
}

func (j *memoryJournal) Record(o news.Observation) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.outcomes[o.Event.ID] = o.Outcome
}

func (j *memoryJournal) Fail(f news.Failure) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.outcomes[f.Source] = "помилка: " + f.Err.Error()
}

func newTestService(seen SeenRepository, journal Journal, publisher Publisher) *Service {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	status := NewStatus(time.Now())
	dispatcher := NewDispatcher(DispatcherDeps{
		Publisher: publisher,
		Journal:   journal,
		Enricher:  NewEnricher(nil, log),
		Recent:    NewRecent(10, nil, nil),
		Status:    status,
		Log:       log,
	})

	return NewService(ServiceDeps{
		Collector:  NewCollector(seen, journal, status, log),
		Dispatcher: dispatcher,
		Watchlist:  idleScanner{},
		Trader:     idleTrader{},
		Seen:       seen,
		Log:        log,
	})
}

func TestRunSeedsSilentlyThenSendsOnlyNew(t *testing.T) {
	seen := newMemorySeen()
	publisher := &fakePublisher{}
	source := &fakeSource{polls: [][]string{{"old1", "old2"}, {"old1", "old2", "new1"}, {"old2", "new1", "new2"}}}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	newTestService(seen, newMemoryJournal(), publisher).Run(ctx, []Source{source})

	got := publisher.ids()

	if len(got) != 2 || got[0] != "new1" || got[1] != "new2" {
		t.Fatalf("sent %v", got)
	}

	if !seen.IsSeeded("fake") || seen.saves == 0 {
		t.Fatal("source not seeded or state not saved on shutdown")
	}
}

func TestOnceRespectsLimit(t *testing.T) {
	publisher := &fakePublisher{}
	source := &fakeSource{polls: [][]string{{"a", "b", "c"}}}

	service := newTestService(newMemorySeen(), newMemoryJournal(), publisher)

	service.Once(context.Background(), []Source{source}, 2)

	if got := publisher.ids(); len(got) != 2 {
		t.Fatalf("sent %v", got)
	}
}

func TestRunJournalsMutedEventsWithoutSending(t *testing.T) {
	publisher := &fakePublisher{}
	journal := newMemoryJournal()
	source := &fakeSource{polls: [][]string{{"old"}, {"old", "muted1", "new1"}}}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	newTestService(newMemorySeen(), journal, publisher).Run(ctx, []Source{source})

	if got := publisher.ids(); len(got) != 1 || got[0] != "new1" {
		t.Fatalf("sent %v", got)
	}

	journal.mu.Lock()
	defer journal.mu.Unlock()

	if journal.outcomes["muted1"] != "не цікавий" ||
		journal.outcomes["new1"] != "надіслано" ||
		journal.outcomes["old"] != "" {
		t.Fatalf("journal %v", journal.outcomes)
	}
}

type brokenSource struct{}

func (brokenSource) Name() string {
	return "broken"
}

func (brokenSource) Interval() time.Duration {
	return 10 * time.Millisecond
}

func (brokenSource) Poll(context.Context) ([]news.Event, error) {
	return nil, errors.New("HTTP 403")
}

func TestRunJournalsSourceFailures(t *testing.T) {
	journal := newMemoryJournal()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	newTestService(newMemorySeen(), journal, &fakePublisher{}).Run(ctx, []Source{brokenSource{}})

	journal.mu.Lock()
	defer journal.mu.Unlock()

	if journal.outcomes["broken"] != "помилка: HTTP 403" {
		t.Fatalf("journal %v", journal.outcomes)
	}
}

type idleScanner struct{}

func (idleScanner) Run(ctx context.Context) {
	<-ctx.Done()
}

type idleTrader struct{}

func (idleTrader) Run(ctx context.Context, _ chan<- news.Event) {
	<-ctx.Done()
}

func TestRecentRemovesAnEventOnce(t *testing.T) {
	recent := NewRecent(10, nil, []news.Event{{ID: "a"}, {ID: "b"}})

	if err := recent.Remove("a"); err != nil {
		t.Fatal(err)
	}

	if left := recent.Latest(10); len(left) != 1 || left[0].ID != "b" {
		t.Fatalf("left %v", left)
	}

	if err := recent.Remove("a"); !errors.Is(err, ErrNoEvent) {
		t.Fatalf("second removal: %v", err)
	}
}
