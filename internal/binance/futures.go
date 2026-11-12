package binance

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/market"
	"newsbot/internal/news"
)

const (
	fapiURL            = "https://fapi.binance.com"
	depthLevels        = 500
	depthBand          = 0.01
	fundingBase        = 8
	intervalsMaxAge    = time.Hour
	kindsMaxAge        = 6 * time.Hour
	cryptoContract     = "PERPETUAL"
	cryptoUnderlying   = "COIN"
	tradfiContract     = "TRADIFI_PERPETUAL"
	kindCrypto         = "crypto"
	kindCommodity      = "COMMODITY"
	kindEquity         = "EQUITY"
	openInterestPeriod = "5m"
	openInterestStep   = 5 * time.Minute
	minuteInterval     = "1m"
	minutesBack        = 5
)

var contractScales = []string{"", "1000", "1000000"}

type Futures struct {
	http    *httpclient.Client
	maxAge  time.Duration
	mu      sync.Mutex
	fetched time.Time
	byName  map[string]market.Ticker

	intervals        map[string]int
	intervalsFetched time.Time

	kinds        map[string]string
	kindsFetched time.Time

	pulse  []market.Pulse
	pulsed time.Time

	tradfi   []market.Pulse
	tradfiAt time.Time

	weeks   map[string]market.Week
	crowds  map[string]crowdRead
	bench   map[int64]float64
	benchAt time.Time

	follow func() []string
}

func NewFutures(http *httpclient.Client, maxAge time.Duration) *Futures {
	return &Futures{
		http:   http,
		maxAge: maxAge,
		byName: map[string]market.Ticker{},
		weeks:  map[string]market.Week{},
		crowds: map[string]crowdRead{},
	}
}

type rawTicker struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	QuoteVolume        string `json:"quoteVolume"`
	PriceChangePercent string `json:"priceChangePercent"`
}

type rawPremium struct {
	Symbol          string `json:"symbol"`
	LastFundingRate string `json:"lastFundingRate"`
}

type rawOpenInterest struct {
	Contracts string `json:"sumOpenInterest"`
	Value     string `json:"sumOpenInterestValue"`
}

func (f *Futures) refresh(ctx context.Context) error {
	var raws []rawTicker
	if err := f.http.JSON(ctx, fapiURL+"/fapi/v1/ticker/24hr", &raws); err != nil {
		return err
	}

	premiums, err := f.premiums(ctx)
	if err != nil {
		return err
	}

	intervals := f.fundingIntervals(ctx)
	byName := make(map[string]market.Ticker, len(raws))

	for _, r := range raws {
		byName[r.Symbol] = toTicker(r, premiums[r.Symbol], intervals[r.Symbol])
	}

	f.mu.Lock()
	f.byName, f.fetched = byName, time.Now()
	f.mu.Unlock()

	return nil
}

func (f *Futures) byContract(contract string) market.Ticker {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.byName[contract]
}

func (f *Futures) Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error) {
	if f.stale() {
		if err := f.refresh(ctx); err != nil {
			return market.Ticker{}, false, err
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, scale := range contractScales {
		if t, ok := f.byName[scale+string(s)+"USDT"]; ok {
			return t, true, nil
		}
	}

	return market.Ticker{}, false, nil
}

func (f *Futures) Bars(ctx context.Context, contract, interval string, limit int) ([]market.Bar, error) {
	rows, err := f.klines(ctx, contract, interval, limit)
	if err != nil {
		return nil, err
	}

	bars := make([]market.Bar, 0, len(rows))

	for _, r := range rows {
		bars = append(bars, market.Bar{
			Open:        time.UnixMilli(toInt(r[0])).UTC(),
			Close:       toFloat(r[4]),
			QuoteVolume: toFloat(r[7]),
		})
	}

	return bars, nil
}

func (f *Futures) OpenInterestChange(ctx context.Context, contract string, window time.Duration) (float64, bool) {
	raws, ok := f.openInterestHistory(ctx, contract, int(window/openInterestStep)+1)
	if !ok {
		return 0, false
	}

	first, last := parse(raws[0].Contracts), parse(raws[len(raws)-1].Contracts)

	if first <= 0 {
		return 0, false
	}

	return last/first - 1, true
}

func (f *Futures) OpenInterestValue(ctx context.Context, contract string) (float64, bool) {
	raws, ok := f.openInterestHistory(ctx, contract, 1)
	if !ok {
		return 0, false
	}

	return parse(raws[len(raws)-1].Value), true
}

func (f *Futures) DepthValue(ctx context.Context, contract string) (float64, bool) {
	var book struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}

	url := fmt.Sprintf("%s/fapi/v1/depth?symbol=%s&limit=%d", fapiURL, contract, depthLevels)

	if err := f.http.JSON(ctx, url, &book); err != nil || len(book.Bids) == 0 || len(book.Asks) == 0 {
		return 0, false
	}

	return nearDepth(levels(book.Bids), levels(book.Asks)), true
}

type level struct {
	price    float64
	quantity float64
}

func levels(rows [][]string) []level {
	out := make([]level, 0, len(rows))

	for _, r := range rows {
		if len(r) >= 2 {
			out = append(out, level{price: parse(r[0]), quantity: parse(r[1])})
		}
	}

	return out
}

func nearDepth(bids, asks []level) float64 {
	if len(bids) == 0 || len(asks) == 0 {
		return 0
	}

	mid := (bids[0].price + asks[0].price) / 2

	return min(resting(bids, mid), resting(asks, mid))
}

func resting(side []level, mid float64) float64 {
	var sum float64

	for _, l := range side {
		if math.Abs(l.price/mid-1) <= depthBand {
			sum += l.price * l.quantity
		}
	}

	return sum
}

func (f *Futures) openInterestHistory(ctx context.Context, contract string, limit int) ([]rawOpenInterest, bool) {
	url := fmt.Sprintf(
		"%s/futures/data/openInterestHist?symbol=%s&period=%s&limit=%d",
		fapiURL, contract, openInterestPeriod, limit,
	)

	var raws []rawOpenInterest
	if err := f.http.JSON(ctx, url, &raws); err != nil || len(raws) < min(limit, 2) {
		return nil, false
	}

	return raws, true
}

func (f *Futures) Extremes(ctx context.Context, contract string, since time.Time) (float64, float64, bool) {
	rows, err := f.klines(ctx, contract, minuteInterval, minutesBack)
	if err != nil {
		return 0, 0, false
	}

	low, high, found := 0.0, 0.0, false

	for _, c := range candlesOf(rows) {
		if c.Open < since.UnixMilli() {
			continue
		}

		if !found || c.Low < low {
			low = c.Low
		}

		high, found = max(high, c.High), true
	}

	return low, high, found && low > 0
}

func (f *Futures) klines(ctx context.Context, contract, interval string, limit int) ([][]any, error) {
	url := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", fapiURL, contract, interval, limit)

	return fetchKlines(ctx, f.http, url)
}

func fetchKlines(ctx context.Context, http *httpclient.Client, url string) ([][]any, error) {
	var rows [][]any

	if err := http.JSON(ctx, url, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func (f *Futures) premiums(ctx context.Context) (map[string]rawPremium, error) {
	var raws []rawPremium
	if err := f.http.JSON(ctx, fapiURL+"/fapi/v1/premiumIndex", &raws); err != nil {
		return nil, err
	}

	out := make(map[string]rawPremium, len(raws))

	for _, p := range raws {
		out[p.Symbol] = p
	}

	return out, nil
}

func (f *Futures) stale() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	return time.Since(f.fetched) > f.maxAge
}

func toTicker(r rawTicker, p rawPremium, hours int) market.Ticker {
	if hours <= 0 {
		hours = fundingBase
	}

	return market.Ticker{
		Contract:     r.Symbol,
		Price:        parse(r.LastPrice),
		QuoteVolume:  parse(r.QuoteVolume),
		Change24h:    parse(r.PriceChangePercent),
		Funding:      parse(p.LastFundingRate) * fundingBase / float64(hours),
		FundingHours: hours,
	}
}

func (f *Futures) fundingIntervals(ctx context.Context) map[string]int {
	f.mu.Lock()
	known, fresh := f.intervals, time.Since(f.intervalsFetched) < intervalsMaxAge
	f.mu.Unlock()

	if fresh {
		return known
	}

	var raws []struct {
		Symbol string `json:"symbol"`
		Hours  int    `json:"fundingIntervalHours"`
	}

	if err := f.http.JSON(ctx, fapiURL+"/fapi/v1/fundingInfo", &raws); err != nil {
		return known
	}

	intervals := make(map[string]int, len(raws))

	for _, r := range raws {
		intervals[r.Symbol] = r.Hours
	}

	f.mu.Lock()
	f.intervals, f.intervalsFetched = intervals, time.Now()
	f.mu.Unlock()

	return intervals
}

func (f *Futures) cryptoContracts(ctx context.Context) map[string]bool {
	return of(f.contractKinds(ctx), kindCrypto)
}

func of(kinds map[string]string, kind string) map[string]bool {
	out := map[string]bool{}

	for symbol, k := range kinds {
		if k == kind {
			out[symbol] = true
		}
	}

	return out
}

func (f *Futures) contractKinds(ctx context.Context) map[string]string {
	f.mu.Lock()
	known, fresh := f.kinds, time.Since(f.kindsFetched) < kindsMaxAge
	f.mu.Unlock()

	if fresh {
		return known
	}

	var info struct {
		Symbols []struct {
			Symbol     string `json:"symbol"`
			Status     string `json:"status"`
			Quote      string `json:"quoteAsset"`
			Contract   string `json:"contractType"`
			Underlying string `json:"underlyingType"`
		} `json:"symbols"`
	}

	if err := f.http.JSON(ctx, fapiURL+"/fapi/v1/exchangeInfo", &info); err != nil {
		return known
	}

	kinds := make(map[string]string, len(info.Symbols))

	for _, s := range info.Symbols {
		if s.Status != tradingStatus || s.Quote != quoteAsset {
			continue
		}

		switch {
		case s.Contract == cryptoContract && s.Underlying == cryptoUnderlying:
			kinds[s.Symbol] = kindCrypto
		case s.Contract == tradfiContract && (s.Underlying == kindCommodity || s.Underlying == kindEquity):
			kinds[s.Symbol] = s.Underlying
		}
	}

	f.mu.Lock()
	f.kinds, f.kindsFetched = kinds, time.Now()
	f.mu.Unlock()

	return kinds
}

func parse(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)

	return v
}

func toFloat(v any) float64 {
	s, _ := v.(string)

	return parse(s)
}

func toInt(v any) int64 {
	n, _ := v.(float64)

	return int64(n)
}
