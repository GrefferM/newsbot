package watchlist

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"newsbot/internal/news"
)

const (
	watchlistSource     = "watchlist"
	watchlistRetry      = 5 * time.Minute
	openInterestSpacing = 100 * time.Millisecond
	depthSpacing        = 150 * time.Millisecond
)

var (
	ErrUnknownCoin = errors.New("такої монети немає серед перпів Binance")
	ErrAlreadyKept = errors.New("монета вже у вішлісті")
	ErrNotKept     = errors.New("монети немає у вішлісті")
)

type Deps struct {
	Universe     Universe
	OpenInterest OpenInterest
	Depth        Depth
	Store        ListStore
	Scans        ScanStore
	Rule         Rule
	Every        time.Duration
	Journal      Journal
	Status       Health
	Log          *slog.Logger
}

type Kept struct {
	Pick    Pick
	Coin    Coin
	Reasons []string
	Known   bool
}

type Service struct {
	deps Deps
	mu   sync.Mutex
	scan Scan
	list List
}

func NewService(deps Deps, list List, last Scan) *Service {
	scan := deps.Rule.Build(slices.Collect(maps.Values(last.Coins)), last.Built)

	return &Service{deps: deps, list: list, scan: scan}
}

func (w *Service) Name() string {
	return watchlistSource
}

func (w *Service) Run(ctx context.Context) {
	for {
		delay := w.deps.Every

		if err := w.Rebuild(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}

			delay = watchlistRetry

			w.fail(err)
		}

		if !sleep(ctx, delay) {
			return
		}
	}
}

func (w *Service) Rebuild(ctx context.Context) error {
	coins, err := w.deps.Universe.Coins(ctx)
	if err != nil {
		return fmt.Errorf("watchlist coins: %w", err)
	}

	scan := w.deps.Rule.Build(coins, time.Now().UTC())

	w.mu.Lock()
	carryMeasured(scan, w.scan)
	w.scan = scan
	w.mu.Unlock()

	w.enrichAll(ctx, scan)

	w.mu.Lock()
	err = w.deps.Scans.Save(scan)
	w.mu.Unlock()

	if err != nil {
		w.deps.Log.Warn("save scan", "err", err)
	}

	w.deps.Status.OK(watchlistSource, len(scan.Coins), 0)
	w.deps.Log.Info("watchlist scanned", "coins", len(coins))

	return nil
}

func (w *Service) Coins() []Kept {
	w.mu.Lock()
	defer w.mu.Unlock()

	out := make([]Kept, 0, len(w.scan.Coins))

	for _, c := range w.scan.Coins {
		out = append(out, Kept{Coin: c, Reasons: w.deps.Rule.Reasons(c), Known: true})
	}

	slices.SortFunc(out, func(a, b Kept) int {
		return cmp.Compare(a.Coin.Asset, b.Coin.Asset)
	})

	return out
}

func (w *Service) scanned() Scan {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.scan
}

func (w *Service) List() List {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.list
}

func (w *Service) Kept() []Kept {
	w.mu.Lock()
	defer w.mu.Unlock()

	out := make([]Kept, 0, len(w.list.Picks))

	for _, p := range w.list.Picks {
		coin, known := w.scan.Lookup(p.Asset)

		var reasons []string

		if known {
			reasons = w.deps.Rule.Reasons(coin)
		}

		out = append(out, Kept{Pick: p, Coin: coin, Reasons: reasons, Known: known})
	}

	return out
}

func (w *Service) Add(asset, note string) error {
	asset = Normalize(asset)

	scan := w.scanned()

	if len(scan.Coins) > 0 {
		if _, known := scan.Lookup(asset); !known {
			return ErrUnknownCoin
		}
	}

	w.mu.Lock()
	list, added := w.list.Add(asset, note, time.Now().UTC())
	w.mu.Unlock()

	if !added {
		return ErrAlreadyKept
	}

	return w.keep(list)
}

func (w *Service) Remove(asset string) error {
	w.mu.Lock()
	list, removed := w.list.Remove(asset)
	w.mu.Unlock()

	if !removed {
		return ErrNotKept
	}

	return w.keep(list)
}

func (w *Service) keep(list List) error {
	if err := w.deps.Store.Save(list); err != nil {
		return fmt.Errorf("save watchlist: %w", err)
	}

	w.mu.Lock()
	w.list = list
	w.mu.Unlock()

	return nil
}

func (w *Service) enrichAll(ctx context.Context, scan Scan) {
	w.mu.Lock()
	coins := slices.Collect(maps.Values(scan.Coins))
	w.mu.Unlock()

	for _, coin := range coins {
		if coin.Contract == "" {
			continue
		}

		if value, ok := w.deps.OpenInterest.OpenInterestValue(ctx, coin.Contract); ok {
			coin.OpenInterest = value
		}

		if !sleep(ctx, openInterestSpacing) {
			return
		}

		if value, ok := w.deps.Depth.DepthValue(ctx, coin.Contract); ok {
			coin.Depth = value
		}

		w.mu.Lock()
		scan.Coins[coin.Asset] = coin
		w.mu.Unlock()

		if !sleep(ctx, depthSpacing) {
			return
		}
	}
}

func carryMeasured(next, last Scan) {
	for asset, coin := range next.Coins {
		if old, ok := last.Lookup(asset); ok && old.Contract == coin.Contract {
			coin.OpenInterest = old.OpenInterest
			coin.Depth = old.Depth
			next.Coins[asset] = coin
		}
	}
}

func (w *Service) fail(err error) {
	w.deps.Status.Fail(watchlistSource, err)
	w.deps.Journal.Fail(news.Failure{
		At:     time.Now().UTC(),
		Source: watchlistSource,
		Err:    err,
	})
	w.deps.Log.Warn("watchlist failed", "err", err, "retry_in", watchlistRetry)
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
