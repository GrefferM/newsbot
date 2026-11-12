package binance

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"newsbot/internal/market"
)

const (
	pulseContracts = 150
	crowdPeriod    = "5m"
	crowdPoints    = 16
	crowdHour      = 12
	crowdQuarter   = 3
	crowdRow       = 5 * time.Minute
	crowdDelay     = 150 * time.Second
	crowdRetry     = time.Minute
	pulseMaxAge    = 45 * time.Second
	pulseWorkers   = 8
	weeksPerPass   = 60
	weekMaxAge     = time.Hour
	benchContract  = "BTCUSDT"
	tradfiEquities = 20
)

func (f *Futures) Pulse(ctx context.Context) ([]market.Pulse, error) {
	f.mu.Lock()
	cached, fresh := f.pulse, time.Since(f.pulsed) <= pulseMaxAge
	f.mu.Unlock()

	if fresh {
		return cached, nil
	}

	if err := f.refresh(ctx); err != nil {
		return nil, err
	}

	bench, err := f.benchmark(ctx)
	if err != nil {
		return nil, err
	}

	every := f.liquid(math.MaxInt, f.cryptoContracts(ctx))
	picked := every[:min(pulseContracts, len(every))]
	near := append(slices.Clone(picked), f.followed(picked)...)

	out, err := f.pulses(ctx, append(near, apart(every, near)...), len(near), bench)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.pulse, f.pulsed = out, time.Now()
	f.mu.Unlock()

	return out, nil
}

func (f *Futures) Follow(contracts func() []string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.follow = contracts
}

func (f *Futures) followed(picked []market.Ticker) []market.Ticker {
	f.mu.Lock()
	follow := f.follow
	f.mu.Unlock()

	if follow == nil {
		return nil
	}

	contracts := follow()

	f.mu.Lock()
	defer f.mu.Unlock()

	taken := map[string]bool{}

	for _, t := range picked {
		taken[t.Contract] = true
	}

	var out []market.Ticker

	for _, name := range contracts {
		t, found := f.byName[name]

		if !found || t.Price <= 0 || taken[name] {
			continue
		}

		taken[name] = true
		out = append(out, t)
	}

	return out
}

func (f *Futures) TradFi(ctx context.Context) ([]market.Pulse, error) {
	f.mu.Lock()
	cached, fresh := f.tradfi, time.Since(f.tradfiAt) <= pulseMaxAge
	f.mu.Unlock()

	if fresh {
		return cached, nil
	}

	if err := f.refresh(ctx); err != nil {
		return nil, err
	}

	kinds := f.contractKinds(ctx)
	picked := append(f.liquid(len(kinds), of(kinds, kindCommodity)), f.liquid(tradfiEquities, of(kinds, kindEquity))...)

	out, err := f.pulses(ctx, picked, len(picked), nil)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.tradfi, f.tradfiAt = out, time.Now()
	f.mu.Unlock()

	return out, nil
}

func (f *Futures) Equities(ctx context.Context) (map[string]bool, error) {
	if err := f.refresh(ctx); err != nil {
		return nil, err
	}

	out := map[string]bool{}

	for _, t := range f.liquid(tradfiEquities, of(f.contractKinds(ctx), kindEquity)) {
		out[assetOf(t.Contract)] = true
	}

	return out, nil
}

func (f *Futures) benchmark(ctx context.Context) (map[int64]float64, error) {
	f.mu.Lock()
	known, fresh := f.bench, time.Since(f.benchAt) < weekMaxAge
	f.mu.Unlock()

	if fresh {
		return known, nil
	}

	rows, err := f.klines(ctx, benchContract, market.WeekInterval, market.WeekBars)
	if err != nil {
		return nil, err
	}

	bench := market.Returns(candlesOf(rows))

	f.mu.Lock()
	f.bench, f.benchAt = bench, time.Now()
	f.mu.Unlock()

	return bench, nil
}

func apart(every, taken []market.Ticker) []market.Ticker {
	named := map[string]bool{}

	for _, t := range taken {
		named[t.Contract] = true
	}

	var rest []market.Ticker

	for _, t := range every {
		if !named[t.Contract] {
			rest = append(rest, t)
		}
	}

	return rest
}

func (f *Futures) pulses(ctx context.Context, tickers []market.Ticker, near int, bench map[int64]float64) ([]market.Pulse, error) {
	found := make([]*market.Pulse, len(tickers))
	failed := make([]error, len(tickers))
	slots := make(chan struct{}, pulseWorkers)

	var weeks atomic.Int32

	weeks.Store(weeksPerPass)

	var wg sync.WaitGroup

	for i, ticker := range tickers {
		wg.Go(func() {
			slots <- struct{}{}

			defer func() {
				<-slots
			}()

			wide := i >= near

			pulse, err := f.pulseOfContract(ctx, ticker, bench, wide, &weeks)
			if err != nil && !wide {
				failed[i] = err
			}

			if err == nil {
				found[i] = pulse
			}
		})
	}

	wg.Wait()

	if err := errors.Join(failed...); err != nil {
		return nil, err
	}

	out := make([]market.Pulse, 0, len(found))

	for _, pulse := range found {
		if pulse != nil {
			out = append(out, *pulse)
		}
	}

	return out, nil
}

func (f *Futures) pulseOfContract(
	ctx context.Context, ticker market.Ticker, bench map[int64]float64, wide bool, weeks *atomic.Int32,
) (*market.Pulse, error) {
	f.mu.Lock()
	week, known := f.weeks[ticker.Contract]
	f.mu.Unlock()

	stale := !known || time.Since(week.At) >= weekMaxAge

	if stale && weeks.Add(-1) >= 0 {
		rows, err := f.klines(ctx, ticker.Contract, market.WeekInterval, market.WeekBars)
		if err != nil {
			return nil, err
		}

		week = market.WeekOf(candlesOf(rows), bench)

		f.mu.Lock()
		f.weeks[ticker.Contract] = week
		f.mu.Unlock()
	}

	rows, err := f.klines(ctx, ticker.Contract, market.PulseInterval, market.PulseBars)
	if err != nil {
		return nil, err
	}

	pulse, ok := market.Measure(ticker, candlesOf(rows), week)
	if !ok {
		return nil, nil
	}

	pulse.Asset, pulse.Wide = assetOf(ticker.Contract), wide

	if wide {
		return &pulse, nil
	}

	if crowd, known := f.Crowd(ctx, ticker.Contract); known {
		pulse.Crowd = &crowd
	}

	return &pulse, nil
}

func (f *Futures) liquid(count int, allowed map[string]bool) []market.Ticker {
	f.mu.Lock()
	defer f.mu.Unlock()

	var tickers []market.Ticker

	for name, t := range f.byName {
		if strings.HasSuffix(name, quoteAsset) && t.Price > 0 && (allowed == nil || allowed[name]) {
			tickers = append(tickers, t)
		}
	}

	slices.SortFunc(tickers, func(a, b market.Ticker) int {
		return cmp.Compare(b.QuoteVolume, a.QuoteVolume)
	})

	return tickers[:min(count, len(tickers))]
}

func candlesOf(rows [][]any) []market.Candle {
	bars := make([]market.Candle, 0, len(rows))

	for _, r := range rows {
		bars = append(bars, market.Candle{
			Open:   toInt(r[0]),
			High:   toFloat(r[2]),
			Low:    toFloat(r[3]),
			Close:  toFloat(r[4]),
			Volume: toFloat(r[7]),
			Bought: toFloat(r[10]),
		})
	}

	return bars
}

type crowdRead struct {
	crowd market.Crowd
	stamp time.Time
	asked time.Time
}

func (r crowdRead) fresh(now time.Time) bool {
	return now.Sub(r.stamp) < crowdRow+crowdDelay || now.Sub(r.asked) < crowdRetry
}

func (f *Futures) Crowd(ctx context.Context, contract string) (market.Crowd, bool) {
	now := time.Now()

	f.mu.Lock()
	kept, known := f.crowds[contract]
	f.mu.Unlock()

	if known && kept.fresh(now) {
		return kept.crowd, !kept.stamp.IsZero()
	}

	read, ok := f.readCrowd(ctx, contract)
	if !ok {
		read = kept
	}

	read.asked = now

	f.mu.Lock()
	f.crowds[contract] = read
	f.mu.Unlock()

	return read.crowd, !read.stamp.IsZero()
}

func (f *Futures) readCrowd(ctx context.Context, contract string) (crowdRead, bool) {
	var shares []struct {
		Long string `json:"longAccount"`
		Time int64  `json:"timestamp"`
	}

	var interest []struct {
		Open string `json:"sumOpenInterest"`
	}

	query := fmt.Sprintf("?symbol=%s&period=%s&limit=%d", contract, crowdPeriod, crowdPoints)

	if err := f.http.JSON(ctx, fapiURL+"/futures/data/globalLongShortAccountRatio"+query, &shares); err != nil || len(shares) < crowdPoints {
		return crowdRead{}, false
	}

	if err := f.http.JSON(ctx, fapiURL+"/futures/data/openInterestHist"+query, &interest); err != nil || len(interest) < crowdPoints {
		return crowdRead{}, false
	}

	longs := make([]float64, 0, len(shares))
	open := make([]float64, 0, len(interest))

	for i := range shares {
		longs = append(longs, parse(shares[i].Long))
		open = append(open, parse(interest[i].Open))
	}

	last := len(longs) - 1
	crowd := market.Crowd{LongShare: longs[last], HourChange: longs[last] - longs[last-crowdHour]}

	for i := last - crowdHour + 1; i <= last; i++ {
		crowd.Longs = append(crowd.Longs, longs[i]-longs[i-crowdQuarter])
		crowd.Interest = append(crowd.Interest, change(open[i-1], open[i]))
	}

	return crowdRead{crowd: crowd, stamp: time.UnixMilli(shares[last].Time)}, longs[last] > 0 && longs[last-crowdHour] > 0
}

func assetOf(contract string) string {
	asset := strings.TrimSuffix(contract, quoteAsset)

	for _, scale := range slices.Backward(contractScales) {
		if scale != "" && strings.HasPrefix(asset, scale) {
			return strings.TrimPrefix(asset, scale)
		}
	}

	return asset
}
