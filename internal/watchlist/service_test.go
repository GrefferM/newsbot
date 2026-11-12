package watchlist

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"newsbot/internal/news"
)

type fakeUniverse struct {
	coins []Coin
	err   error
}

func (u *fakeUniverse) Coins(context.Context) ([]Coin, error) {
	return u.coins, u.err
}

type fakeOpenInterest map[string]float64

func (f fakeOpenInterest) OpenInterestValue(_ context.Context, contract string) (float64, bool) {
	v, ok := f[contract]

	return v, ok
}

type memoryScans struct {
	mu    sync.Mutex
	saved []Scan
}

func (m *memoryScans) Save(scan Scan) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saved = append(m.saved, scan)

	return nil
}

type memoryLists struct {
	mu    sync.Mutex
	saved []List
}

func (m *memoryLists) Save(list List) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saved = append(m.saved, list)

	return nil
}

func newTestWatchlist(universe *fakeUniverse, journal Journal, store *memoryLists) *Service {
	return NewService(Deps{
		Universe:     universe,
		OpenInterest: fakeOpenInterest{"BLURUSDT": 3_000_000, "SOLUSDT": 9e8},
		Depth:        fakeDepth{"BLURUSDT": 40_000},
		Store:        store,
		Scans:        &memoryScans{},
		Rule:         Rule{LowVolume: 300_000},
		Every:        time.Hour,
		Journal:      journal,
		Status:       quietHealth{},
		Log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, List{}, Scan{})
}

func spotMarket() *fakeUniverse {
	return &fakeUniverse{coins: []Coin{
		{Asset: "BLUR", Tagged: true, Volume: 600_000, HasSpot: true, SpotVolume: 600_000, Contract: "BLURUSDT", HasTrend: true, HasChanges: true},
		{Asset: "QKC", Tagged: true, Volume: 130_000, HasSpot: true, SpotVolume: 130_000, Contract: "QKCUSDT", HasTrend: true, HasChanges: true},
		{Asset: "SOL", Volume: 90_000_000, HasSpot: true, SpotVolume: 90_000_000, Contract: "SOLUSDT", HasTrend: true, HasChanges: true},
		{Asset: "RAY", Volume: 3_000_000, HasTrend: true, HasChanges: true},
	}}
}

func TestTheScanListsEveryCoinWithOpenInterest(t *testing.T) {
	w := newTestWatchlist(spotMarket(), newMemoryJournal(), &memoryLists{})

	if err := w.Rebuild(context.Background()); err != nil {
		t.Fatal(err)
	}

	coins := w.Coins()

	if len(coins) != 3 || coins[0].Coin.Asset != "BLUR" || coins[2].Coin.Asset != "SOL" {
		t.Fatalf("coins %+v", coins)
	}

	if coins[0].Coin.Depth != 40_000 {
		t.Errorf("depth %v", coins[0].Coin.Depth)
	}

	if coins[0].Coin.OpenInterest != 3_000_000 || coins[2].Coin.OpenInterest != 9e8 || len(coins[2].Reasons) != 0 {
		t.Errorf("open interest for every perpetual, reasons only for marked coins: %+v", coins)
	}
}

func TestTheOwnerKeepsAnyKnownCoin(t *testing.T) {
	store := &memoryLists{}
	w := newTestWatchlist(spotMarket(), newMemoryJournal(), store)

	if err := w.Rebuild(context.Background()); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		asset string
		want  error
	}{
		{"sol", nil},
		{"SOL", ErrAlreadyKept},
		{"NOPE", ErrUnknownCoin},
	}

	for _, c := range cases {
		if err := w.Add(c.asset, "ліквідний"); !errors.Is(err, c.want) {
			t.Errorf("add %s: %v, want %v", c.asset, err, c.want)
		}
	}

	kept := w.Kept()

	if len(kept) != 1 || !kept[0].Known || kept[0].Coin.OpenInterest != 9e8 || kept[0].Pick.Note != "ліквідний" {
		t.Fatalf("kept %+v", kept)
	}

	if err := w.Remove("sol"); err != nil || len(w.Kept()) != 0 || len(store.saved) != 2 {
		t.Errorf("remove: %v, kept %d, saves %d", err, len(w.Kept()), len(store.saved))
	}

	if err := w.Remove("SOL"); !errors.Is(err, ErrNotKept) {
		t.Errorf("second remove: %v", err)
	}
}

func TestAFailedScanKeepsTheLastOneAndIsJournaled(t *testing.T) {
	universe := spotMarket()
	journal := newMemoryJournal()
	w := newTestWatchlist(universe, journal, &memoryLists{})

	if err := w.Rebuild(context.Background()); err != nil {
		t.Fatal(err)
	}

	universe.err = errors.New("binance down")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	w.Run(ctx)

	if len(w.Coins()) != 3 {
		t.Errorf("coins %+v", w.Coins())
	}

	if !slices.Contains(journal.failures, watchlistSource) {
		t.Error("failure was not journaled")
	}
}

func TestAScanStartsFromTheLastOneAndKeepsItsOpenInterest(t *testing.T) {
	last := Scan{Coins: map[string]Coin{
		"BLUR": {Asset: "BLUR", Contract: "BLURUSDT", OpenInterest: 1, HasTrend: true, HasChanges: true, Volume: 1},
		"RAY":  {Asset: "RAY", HasTrend: true, HasChanges: true, Volume: 1},
	}}

	w := NewService(Deps{
		Universe:     spotMarket(),
		OpenInterest: fakeOpenInterest{},
		Depth:        fakeDepth{},
		Store:        &memoryLists{},
		Scans:        &memoryScans{},
		Rule:         Rule{LowVolume: 300_000},
		Every:        time.Hour,
		Journal:      newMemoryJournal(),
		Status:       quietHealth{},
		Log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, List{}, last)

	if coins := w.Coins(); len(coins) != 1 || coins[0].Coin.Asset != "BLUR" {
		t.Fatalf("before the first scan %+v", coins)
	}

	if err := w.Rebuild(context.Background()); err != nil {
		t.Fatal(err)
	}

	if coin, _ := w.scanned().Lookup("BLUR"); coin.OpenInterest != 1 {
		t.Errorf("open interest not carried: %+v", coin)
	}
}

type memoryJournal struct {
	mu       sync.Mutex
	failures []string
}

func newMemoryJournal() *memoryJournal {
	return &memoryJournal{}
}

func (j *memoryJournal) Fail(f news.Failure) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.failures = append(j.failures, f.Source)
}

type quietHealth struct{}

func (quietHealth) OK(string, int, int) {}

func (quietHealth) Fail(string, error) {}

type fakeDepth map[string]float64

func (f fakeDepth) DepthValue(_ context.Context, contract string) (float64, bool) {
	value, ok := f[contract]

	return value, ok
}
