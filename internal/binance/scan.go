package binance

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/watchlist"
)

const (
	productsURL   = "https://www.binance.com/bapi/asset/v2/public/asset-service/product/get-products?includeEtf=true"
	spotURL       = "https://api.binance.com"
	quoteAsset    = "USDT"
	tradingStatus = "TRADING"
	monitoringTag = "Monitoring"
	benchmark     = "BTCUSDT"
	volumeDays    = 30
	trendDays     = 90
	weekDays      = 7
	klineSpacing  = 150 * time.Millisecond
)

var ErrNoContracts = errors.New("Binance Futures не дав жодного криптоперпа")

type rawProduct struct {
	Symbol string   `json:"s"`
	Base   string   `json:"b"`
	Quote  string   `json:"q"`
	Status string   `json:"st"`
	Tags   []string `json:"tags"`
}

type Universe struct {
	http    *httpclient.Client
	futures *Futures
}

func NewUniverse(http *httpclient.Client, futures *Futures) *Universe {
	return &Universe{http: http, futures: futures}
}

func (u *Universe) Coins(ctx context.Context) ([]watchlist.Coin, error) {
	if err := u.futures.refresh(ctx); err != nil {
		return nil, err
	}

	contracts := slices.Sorted(maps.Keys(u.futures.cryptoContracts(ctx)))
	if len(contracts) == 0 {
		return nil, ErrNoContracts
	}

	spot, err := u.products(ctx)
	if err != nil {
		return nil, err
	}

	bench, err := u.closedBars(ctx, fapiURL+"/fapi/v1/klines", benchmark)
	if err != nil {
		return nil, err
	}

	coins := make([]watchlist.Coin, 0, len(contracts))

	for _, contract := range contracts {
		coin, err := u.coin(ctx, contract, spot, bench)
		if rejected(err) {
			continue
		}

		if err != nil {
			return nil, err
		}

		coins = append(coins, coin)

		if !pause(ctx, klineSpacing) {
			return nil, ctx.Err()
		}
	}

	return coins, nil
}

func (u *Universe) products(ctx context.Context) (map[string]rawProduct, error) {
	var resp struct {
		Data []rawProduct `json:"data"`
	}

	if err := u.http.JSON(ctx, productsURL, &resp); err != nil {
		return nil, err
	}

	out := make(map[string]rawProduct, len(resp.Data))

	for _, p := range resp.Data {
		if p.Quote == quoteAsset && p.Status == tradingStatus {
			out[p.Base] = p
		}
	}

	return out, nil
}

func (u *Universe) coin(ctx context.Context, contract string, spot map[string]rawProduct, bench []bar) (watchlist.Coin, error) {
	bars, err := u.closedBars(ctx, fapiURL+"/fapi/v1/klines", contract)
	if err != nil {
		return watchlist.Coin{}, err
	}

	ticker := u.futures.byContract(contract)
	asset := assetOf(contract)

	coin := watchlist.Coin{
		Asset:        asset,
		Volume:       meanVolume(bars, volumeDays),
		DayRange:     meanRange(bars, volumeDays),
		SellDay:      sellShare(bars, 1),
		SellWeek:     sellShare(bars, weekDays),
		Contract:     contract,
		Funding:      ticker.Funding,
		FundingHours: ticker.FundingHours,
	}

	if len(bars) > weekDays {
		last := bars[len(bars)-1]

		coin.Price = last.close
		coin.VolumeDay = last.volume
		coin.Change1d = change(bars[len(bars)-2].close, last.close)
		coin.Change7d = change(bars[len(bars)-1-weekDays].close, last.close)
		coin.HasChanges = true
	}

	coin.TrendVsBTC, coin.HasTrend = trendVs(bars, bench, trendDays)

	if product, listed := spot[asset]; listed {
		return u.withSpot(ctx, coin, product)
	}

	return coin, nil
}

func (u *Universe) withSpot(ctx context.Context, coin watchlist.Coin, product rawProduct) (watchlist.Coin, error) {
	coin.Tags = product.Tags
	coin.Tagged = slices.Contains(product.Tags, monitoringTag)

	bars, err := u.closedBars(ctx, spotURL+"/api/v3/klines", product.Symbol)
	if rejected(err) {
		return coin, nil
	}

	if err != nil {
		return watchlist.Coin{}, err
	}

	coin.SpotVolume = meanVolume(bars, volumeDays)
	coin.HasSpot = true

	return coin, nil
}

type bar struct {
	high   float64
	low    float64
	close  float64
	volume float64
	bought float64
}

func (u *Universe) closedBars(ctx context.Context, endpoint, symbol string) ([]bar, error) {
	url := fmt.Sprintf("%s?symbol=%s&interval=1d&limit=%d", endpoint, symbol, trendDays+2)

	rows, err := fetchKlines(ctx, u.http, url)
	if err != nil {
		return nil, err
	}

	if len(rows) > 0 {
		rows = rows[:len(rows)-1]
	}

	bars := make([]bar, 0, len(rows))

	for _, r := range rows {
		bars = append(bars, bar{
			high:   toFloat(r[2]),
			low:    toFloat(r[3]),
			close:  toFloat(r[4]),
			volume: toFloat(r[7]),
			bought: toFloat(r[10]),
		})
	}

	return bars, nil
}

func meanVolume(bars []bar, days int) float64 {
	recent := bars[max(0, len(bars)-days):]

	if len(recent) == 0 {
		return 0
	}

	var sum float64

	for _, b := range recent {
		sum += b.volume
	}

	return sum / float64(len(recent))
}

func sellShare(bars []bar, days int) float64 {
	recent := bars[max(0, len(bars)-days):]

	var volume, bought float64

	for _, b := range recent {
		volume += b.volume
		bought += b.bought
	}

	if volume <= 0 {
		return 0
	}

	return 1 - bought/volume
}

func meanRange(bars []bar, days int) float64 {
	recent := bars[max(0, len(bars)-days):]

	var (
		sum   float64
		count int
	)

	for _, b := range recent {
		if b.close > 0 {
			sum += (b.high - b.low) / b.close
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}

func trendVs(bars, bench []bar, days int) (float64, bool) {
	if len(bars) <= days || len(bench) <= days {
		return 0, false
	}

	start, end := bars[len(bars)-1-days].close, bars[len(bars)-1].close
	benchStart, benchEnd := bench[len(bench)-1-days].close, bench[len(bench)-1].close

	if start <= 0 || benchStart <= 0 || benchEnd <= 0 {
		return 0, false
	}

	return (end/start)/(benchEnd/benchStart) - 1, true
}

func change(from, to float64) float64 {
	if from <= 0 {
		return 0
	}

	return to/from - 1
}

func rejected(err error) bool {
	var se *httpclient.StatusError

	return errors.As(err, &se) && se.Code == http.StatusBadRequest
}
