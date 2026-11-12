package radar

import (
	"cmp"
	"math"
	"slices"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/research"
)

const (
	headlinesKept = 3
	freshNews     = 72 * time.Hour
)

type Story struct {
	Published string `json:"published"`
	Title     string `json:"title"`
	Outlet    string `json:"outlet"`
	URL       string `json:"url"`
}

const (
	benchmark  = "BTC"
	rowMinutes = 5
)

const (
	launchBase   = 48
	launchWithin = 4
	launchRange  = 0.12
	launchVolume = 5
	launchPlace  = 0.7
)

type Rule struct {
	Interest    float64
	Longs       float64
	LaunchTimes float64
	LaunchMove  float64
}

type Launch struct {
	Hours int
	Move  float64
	Times float64
}

func (r Rule) Launched(p market.Pulse) (Launch, bool) {
	if r.LaunchTimes <= 0 || r.LaunchMove <= 0 {
		return Launch{}, false
	}

	last := len(p.Hours) - 1

	for i := last; i > last-launchWithin && i > launchBase; i-- {
		if move, times, rose := r.rise(p.Hours[i-launchBase-1:i], p.Hours[i]); rose {
			return Launch{Hours: last - i, Move: move, Times: times}, true
		}
	}

	return Launch{}, false
}

func (r Rule) rise(before []market.Hour, candle market.Hour) (float64, float64, bool) {
	base := before[1:]
	open := base[len(base)-1].Close
	high, low := base[0].High, base[0].Low
	moves := make([]float64, 0, len(base))
	volumes := make([]float64, 0, len(base))

	for i, h := range base {
		high, low = max(high, h.High), min(low, h.Low)
		moves = append(moves, math.Abs(h.Close/before[i].Close-1))
		volumes = append(volumes, h.Volume)
	}

	slices.Sort(moves)
	slices.Sort(volumes)

	usual, volume := moves[len(moves)/2], volumes[len(volumes)/2]
	quiet := low > 0 && high/low-1 <= launchRange

	if !quiet || open <= 0 || usual <= 0 || candle.High <= candle.Low {
		return 0, 0, false
	}

	move := candle.Close/open - 1
	strong := move >= r.LaunchMove && move >= r.LaunchTimes*usual && candle.Volume >= launchVolume*volume
	held := (candle.Close-candle.Low)/(candle.High-candle.Low) >= launchPlace

	return move, move / usual, strong && held
}

type Wake struct {
	Minutes  int
	Interest float64
	Longs    float64
}

func (r Rule) Woke(p market.Pulse) (Wake, bool) {
	if p.Crowd == nil || r.Interest <= 0 || r.Longs <= 0 {
		return Wake{}, false
	}

	rows := min(len(p.Crowd.Interest), len(p.Crowd.Longs))

	for i := rows - 1; i >= 0; i-- {
		interest, longs := p.Crowd.Interest[i], p.Crowd.Longs[i]

		if math.Abs(interest) >= r.Interest && math.Abs(longs) >= r.Longs {
			return Wake{Minutes: (rows - 1 - i) * rowMinutes, Interest: interest, Longs: longs}, true
		}
	}

	return Wake{}, false
}

type trendLimit struct {
	sigmas   float64
	straight float64
}

const trendPlace = 0.75

var trends = map[int]trendLimit{
	24: {sigmas: 3, straight: 0.4},
	72: {sigmas: 2, straight: 0.25},
}

func Trend(p market.Pulse) (market.Run, bool) {
	var found market.Run

	for _, run := range p.Runs {
		limit := trends[run.Hours]
		steady := math.Abs(run.Sigmas) >= limit.sigmas && run.Straight >= limit.straight

		if run.Sigmas < 0 {
			run.Place = 1 - run.Place
		}

		if steady && run.Place >= trendPlace && run.Hours > found.Hours {
			found = run
		}
	}

	return found, found.Hours > 0
}

func Bench(pulses []market.Pulse) float64 {
	for _, p := range pulses {
		if p.Asset == benchmark {
			return p.Change1h
		}
	}

	return 0
}

type Coin struct {
	Pulse   market.Pulse
	Wake    *Wake
	Launch  *Launch
	Trend   *market.Run
	Stories []Story
}

func (c Coin) Active() bool {
	return c.Wake != nil || c.Launch != nil
}

type Board struct {
	At    time.Time
	Bench float64
	Coins []Coin
}

func (b Board) Of(asset string) (Coin, bool) {
	for _, c := range b.Coins {
		if c.Pulse.Asset == asset {
			return c, true
		}
	}

	return Coin{}, false
}

func (b Board) Movers() int {
	count := 0

	for _, c := range b.Coins {
		if c.Active() {
			count++
		}
	}

	return count
}

func Fresh(findings []research.Finding, now time.Time) []Story {
	since := now.Add(-freshNews).Format(time.DateOnly)
	fresh := make([]Story, 0, len(findings))

	for _, f := range findings {
		if f.Published >= since {
			fresh = append(fresh, Story{Published: f.Published, Title: f.Text, Outlet: f.Outlet, URL: f.Source})
		}
	}

	slices.SortStableFunc(fresh, func(a, b Story) int {
		return cmp.Compare(b.Published, a.Published)
	})

	return fresh[:min(headlinesKept, len(fresh))]
}
