package radar

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/research"
)

type pulses struct {
	now []market.Pulse
	err error
}

func (p *pulses) Pulse(context.Context) ([]market.Pulse, error) {
	return p.now, p.err
}

func woken(interest, longs []float64) market.Pulse {
	return market.Pulse{Crowd: &market.Crowd{Interest: interest, Longs: longs}}
}

func TestAContractWakesWhenOpenInterestAndTheLongShareShiftTogether(t *testing.T) {
	rule := Rule{Interest: 0.02, Longs: 0.02}
	cases := []struct {
		name    string
		pulse   market.Pulse
		woke    bool
		minutes int
	}{
		{"both shift in the newest row", woken([]float64{0, 0, 0.03}, []float64{0, 0, -0.025}), true, 0},
		{"both shifted two rows ago", woken([]float64{-0.04, 0, 0}, []float64{0.03, 0, 0}), true, 10},
		{"the newest of two wakes is told", woken([]float64{0.05, 0.03, 0}, []float64{0.05, 0.02, 0}), true, 5},
		{"open interest alone", woken([]float64{0, 0, 0.05}, []float64{0, 0, 0.01}), false, 0},
		{"the long share alone", woken([]float64{0, 0, 0.01}, []float64{0, 0, 0.05}), false, 0},
		{"the shifts fell on different rows", woken([]float64{0.05, 0, 0}, []float64{0, 0, 0.05}), false, 0},
		{"nothing read about the crowd", market.Pulse{}, false, 0},
	}

	for _, c := range cases {
		wake, woke := rule.Woke(c.pulse)

		if woke != c.woke || wake.Minutes != c.minutes {
			t.Errorf("%s: woke %v, %d minutes ago", c.name, woke, wake.Minutes)
		}
	}

	if _, woke := (Rule{}).Woke(woken([]float64{0.5}, []float64{0.5})); woke {
		t.Error("a rule without thresholds marks nothing")
	}

	if wake, _ := rule.Woke(woken([]float64{-0.04}, []float64{0.03})); wake.Interest != -0.04 || wake.Longs != 0.03 {
		t.Errorf("the wake keeps both shifts with their signs: %+v", wake)
	}
}

func TestASteadyTrendIsLargeStraightAndStillAtItsEdge(t *testing.T) {
	day := market.Run{Hours: 24, Move: 0.2, Sigmas: 4, Straight: 0.5, Place: 0.9}
	days := market.Run{Hours: 72, Move: 0.4, Sigmas: 2.5, Straight: 0.3, Place: 0.8}

	if got, steady := Trend(market.Pulse{Runs: []market.Run{day, days}}); !steady || got.Hours != 72 {
		t.Errorf("the longer steady run names the trend: %+v", got)
	}

	fall := market.Run{Hours: 24, Move: -0.2, Sigmas: -4, Straight: 0.5, Place: 0.1}

	if got, steady := Trend(market.Pulse{Runs: []market.Run{fall}}); !steady || got.Move != -0.2 {
		t.Errorf("a fall near its low is a trend too: %+v", got)
	}

	winding, small, pulledBack := day, day, day
	winding.Straight, small.Sigmas, pulledBack.Place = 0.2, 2, 0.5

	for _, run := range []market.Run{winding, small, pulledBack} {
		if _, steady := Trend(market.Pulse{Runs: []market.Run{run}}); steady {
			t.Errorf("not a steady trend: %+v", run)
		}
	}
}

type shelf map[string][]research.Finding

func (s shelf) Of(asset string) []research.Finding {
	return s[asset]
}

type health struct {
	fresh  int
	failed error
}

func (h *health) OK(_ string, _, fresh int) {
	h.fresh = fresh
}

func (h *health) Fail(_ string, err error) {
	h.failed = err
}

func quiet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAMarkedContractShowsTheFreshNewsKeptAboutItAndAsksNoOne(t *testing.T) {
	mover := woken([]float64{0.03}, []float64{0.04})
	mover.Asset = "JCT"
	gold := woken([]float64{-0.03}, []float64{0.02})
	gold.Asset = "XAU"
	seen := &health{}
	s := NewService(Deps{
		Coins:  &pulses{now: []market.Pulse{mover, {Asset: "ZEC", Change1h: 0.001}}},
		TradFi: &pulses{now: []market.Pulse{gold}},
		Rule:   Rule{Interest: 0.02, Longs: 0.02},
		Shelf: shelf{"JCT": {
			{Published: "2026-11-11T07:50:00Z", Text: "Проєкт запустив мейннет", Outlet: "The Block", Source: "https://example.org/a"},
			{Published: "2026-11-01", Text: "стара новина"},
		}},
		Health: seen,
		Log:    quiet(),
	})

	s.Refresh(context.Background(), time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC))

	board := s.Latest()

	if seen.fresh != 1 || len(board.Coins) != 2 || !board.Coins[0].Active() || len(board.Coins[0].Stories) != 1 || board.Coins[0].Stories[0].Title != "Проєкт запустив мейннет" {
		t.Errorf("a marked coin shows the fresh news kept about it: %+v", board.Coins[0])
	}

	if tradfi := s.LatestTradFi(); len(tradfi.Coins) != 1 || !tradfi.Coins[0].Active() {
		t.Errorf("the TradFi board is marked by the same rule: %+v", tradfi.Coins)
	}
}

func TestAContractOutsideTheNearOnesIsShownOnlyWhenMarked(t *testing.T) {
	calm := market.Pulse{Asset: "CALM", Wide: true}
	risen := market.Pulse{Asset: "AKE", Wide: true, Hours: append(quietHours(50), market.Hour{High: 1.16, Low: 1.0, Close: 1.15, Volume: 5000})}
	steady := market.Pulse{Asset: "GRAM", Wide: true, Runs: []market.Run{{Hours: 72, Move: 0.2, Sigmas: 3, Straight: 0.5, Place: 0.9}}}
	s := NewService(Deps{
		Coins:  &pulses{now: []market.Pulse{{Asset: "BTC"}, calm, risen, steady}},
		Rule:   Rule{LaunchTimes: 10, LaunchMove: 0.05},
		Shelf:  shelf{},
		Health: &health{},
		Log:    quiet(),
	})

	s.Refresh(context.Background(), time.Now())

	board := s.Latest()

	if len(board.Coins) != 3 || board.Coins[0].Pulse.Asset != "BTC" || board.Coins[1].Launch == nil || board.Coins[2].Trend == nil {
		t.Errorf("the near contracts and the marked ones from beyond them: %+v", board.Coins)
	}
}

func TestAFailedReadingIsReportedAndLeavesTheLastBoard(t *testing.T) {
	market := &pulses{now: []market.Pulse{{Asset: "ZEC", Change1h: 0.001, Known: true, Sigma: 0.005}}}
	seen := &health{}
	s := NewService(Deps{Coins: market, Shelf: shelf{}, Health: seen, Log: quiet()})

	s.Refresh(context.Background(), time.Now())

	market.err = errors.New("binance мовчить")
	s.Refresh(context.Background(), time.Now())

	if seen.failed == nil || len(s.Latest().Coins) != 1 {
		t.Error("a failed reading must be reported and leave the last board")
	}
}

func quietHours(n int) []market.Hour {
	hours := make([]market.Hour, 0, n+1)

	for i := range n {
		price := 1 + 0.001*float64(i%2)
		hours = append(hours, market.Hour{Open: int64(i), High: price + 0.001, Low: price - 0.001, Close: price, Volume: 100})
	}

	return hours
}

func TestAContractLaunchesWhenAnHourDwarfsTwoQuietDays(t *testing.T) {
	rule := Rule{LaunchTimes: 10, LaunchMove: 0.05}
	rose := market.Hour{High: 1.16, Low: 1.0, Close: 1.15, Volume: 5000}
	cases := []struct {
		name     string
		hours    []market.Hour
		launched bool
		ago      int
	}{
		{"the hour still open towers over the quiet base", append(quietHours(50), rose), true, 0},
		{"the launch was three hours ago", append(append(quietHours(50), rose), quietHours(3)...), true, 3},
		{"a launch of five hours ago is no longer told", append(append(quietHours(50), rose), quietHours(5)...), false, 0},
		{"the candle gave half of itself back", append(quietHours(50), market.Hour{High: 1.3, Low: 1.0, Close: 1.15, Volume: 5000}), false, 0},
		{"the volume did not come", append(quietHours(50), market.Hour{High: 1.16, Low: 1.0, Close: 1.15, Volume: 400}), false, 0},
		{"a rise too small to count", append(quietHours(50), market.Hour{High: 1.041, Low: 1.0, Close: 1.04, Volume: 5000}), false, 0},
		{"a fall of the same size", append(quietHours(50), market.Hour{High: 1.0, Low: 0.84, Close: 0.85, Volume: 5000}), false, 0},
		{"too little history for a base", append(quietHours(30), rose), false, 0},
	}

	for _, c := range cases {
		launch, launched := rule.Launched(market.Pulse{Hours: c.hours})

		if launched != c.launched || launch.Hours != c.ago {
			t.Errorf("%s: launched %v, %d hours ago", c.name, launched, launch.Hours)
		}
	}

	wild := quietHours(50)
	wild[20].High = 1.4

	if _, launched := rule.Launched(market.Pulse{Hours: append(wild, rose)}); launched {
		t.Error("a base that was not quiet launched")
	}

	if _, launched := (Rule{}).Launched(market.Pulse{Hours: append(quietHours(50), rose)}); launched {
		t.Error("a rule without thresholds marks nothing")
	}

	if launch, _ := rule.Launched(market.Pulse{Hours: append(quietHours(50), rose)}); launch.Move < 0.14 || launch.Times < 100 {
		t.Errorf("the launch keeps its size and how many usual hours it is: %+v", launch)
	}
}
