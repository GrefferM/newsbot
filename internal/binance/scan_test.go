package binance

import (
	"math"
	"testing"
)

func closes(values ...float64) []bar {
	out := make([]bar, 0, len(values))

	for _, v := range values {
		out = append(out, bar{close: v, volume: v * 10})
	}

	return out
}

func TestMeanVolumeTakesTheLastDays(t *testing.T) {
	cases := []struct {
		name string
		bars []bar
		days int
		want float64
	}{
		{"fewer bars than days", closes(1, 2, 3), 30, 20},
		{"only the last days", closes(100, 1, 3), 2, 20},
		{"no bars", nil, 30, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := meanVolume(c.bars, c.days); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestTrendVsComparesWithTheBenchmark(t *testing.T) {
	cases := []struct {
		name   string
		coin   []bar
		bench  []bar
		want   float64
		hasAny bool
	}{
		{"coin halves while the benchmark is flat", closes(2, 3, 1), closes(5, 5, 5), -0.5, true},
		{"both double", closes(1, 1, 2), closes(3, 3, 6), 0, true},
		{"too short", closes(1, 2), closes(1, 2, 3), 0, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := trendVs(c.coin, c.bench, 2)

			if ok != c.hasAny || math.Abs(got-c.want) > 1e-9 {
				t.Errorf("got %v %v, want %v %v", got, ok, c.want, c.hasAny)
			}
		})
	}
}

func TestDepthCountsOnlyWhatRestsNearThePriceAndTakesTheThinnerSide(t *testing.T) {
	bids := []level{{price: 99.9, quantity: 10}, {price: 99.2, quantity: 10}, {price: 98, quantity: 1000}}
	asks := []level{{price: 100.1, quantity: 5}, {price: 102, quantity: 1000}}

	if got := nearDepth(bids, asks); got != 100.1*5 {
		t.Errorf("depth %v", got)
	}
}

func TestTheMeanDayRangeIsAShareOfThePrice(t *testing.T) {
	bars := []bar{{high: 110, low: 90, close: 100}, {high: 105, low: 95, close: 100}, {close: 0}}

	if got := meanRange(bars, 30); got < 0.1499 || got > 0.1501 {
		t.Errorf("range %v", got)
	}
}

func TestTheSellShareIsWhatTakersDidNotBuy(t *testing.T) {
	bars := []bar{{volume: 100, bought: 60}, {volume: 100, bought: 30}, {volume: 200, bought: 60}}

	if day, week := sellShare(bars, 1), sellShare(bars, 7); day != 0.7 || week != 0.625 {
		t.Errorf("day %v, week %v", day, week)
	}

	if got := sellShare(nil, 7); got != 0 {
		t.Errorf("no bars: %v", got)
	}
}

func TestFundingIsBroughtToEightHoursWhateverItsPeriod(t *testing.T) {
	hourly := toTicker(rawTicker{Symbol: "INITUSDT"}, rawPremium{LastFundingRate: "-0.02"}, 1)
	usual := toTicker(rawTicker{Symbol: "BTCUSDT"}, rawPremium{LastFundingRate: "0.0001"}, 0)

	if hourly.Funding != -0.16 || hourly.FundingHours != 1 {
		t.Errorf("hourly %+v", hourly)
	}

	if usual.Funding != 0.0001 || usual.FundingHours != 8 {
		t.Errorf("usual %+v", usual)
	}
}
