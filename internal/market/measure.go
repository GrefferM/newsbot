package market

import (
	"math"
	"time"
)

const (
	PulseInterval  = "5m"
	PulseBars      = 99
	WeekInterval   = "15m"
	WeekBars       = 7*96 + 8
	pulseHour      = 12
	pulseQuarter   = 3
	pulseLeg       = 48
	barMinutes     = 5
	hourMillis     = 60 * 60 * 1000
	hoursKept      = 74
	weekHour       = 4
	weekSlot       = 15 * 60 * 1000
	weekDays       = 7
	dayMillis      = 24 * 60 * 60 * 1000
	minWeekReturns = 400
)

var runHours = []int{24, 72}

type Candle struct {
	Open   int64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	Bought float64
}

type Week struct {
	At     time.Time
	hourly []Hour
	slots  map[int64]float64
	beta   float64
	sigma  float64
	runs   []Run
}

func WeekOf(bars []Candle, bench map[int64]float64) Week {
	week := Week{At: time.Now(), slots: make(map[int64]float64, len(bars))}

	for _, b := range bars {
		week.slots[b.Open] = b.Volume
	}

	week.hourly = hoursOf(bars)

	own := Returns(bars[:max(0, len(bars)-weekHour)])

	var n, sx, sy, sxx, sxy, syy float64

	for open, y := range own {
		x, paired := bench[open]
		if bench != nil && !paired {
			continue
		}

		n++
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
		syy += y * y
	}

	if n < minWeekReturns {
		return week
	}

	varX := sxx/n - (sx/n)*(sx/n)
	varY := syy/n - (sy/n)*(sy/n)
	cov := sxy/n - (sx/n)*(sy/n)

	if varX > 0 {
		week.beta = cov / varX
	}

	if rest := varY - week.beta*week.beta*varX; rest > 0 {
		week.sigma = math.Sqrt(rest) * math.Sqrt(weekHour)
	}

	for _, hours := range runHours {
		if run, ok := runOf(bars, bench, week, hours); ok {
			week.runs = append(week.runs, run)
		}
	}

	return week
}

func runOf(bars []Candle, bench map[int64]float64, week Week, hours int) (Run, bool) {
	span := hours * weekHour

	if len(bars) <= span || week.sigma <= 0 {
		return Run{}, false
	}

	window := bars[len(bars)-span-1:]
	first, price := window[0].Close, window[len(window)-1].Close

	if first <= 0 {
		return Run{}, false
	}

	drift, path := 0.0, 0.0

	for i := 1; i < len(window); i++ {
		drift += bench[window[i].Open]

		if i%weekHour == 0 && window[i-weekHour].Close > 0 {
			path += math.Abs(window[i].Close/window[i-weekHour].Close - 1)
		}
	}

	move := price/first - 1
	run := Run{Hours: hours, Move: move, Place: place(price, window[1:])}
	run.Sigmas = (move - week.beta*drift) / (week.sigma * math.Sqrt(float64(hours)))

	if path > 0 {
		run.Straight = math.Abs(move) / path
	}

	return run, true
}

func Returns(bars []Candle) map[int64]float64 {
	out := make(map[int64]float64, len(bars))

	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			out[bars[i].Open] = bars[i].Close/bars[i-1].Close - 1
		}
	}

	return out
}

func Measure(t Ticker, bars []Candle, week Week) (Pulse, bool) {
	if len(bars) < PulseBars {
		return Pulse{}, false
	}

	last := len(bars) - 1
	price := bars[last].Close
	hour, earlier := bars[len(bars)-pulseHour:], bars[:len(bars)-pulseHour]
	quarter := bars[len(bars)-pulseQuarter:]
	volume, bought := sumBars(hour)
	before, _ := sumBars(earlier)
	high, low := extremes(bars)

	pulse := Pulse{
		Contract:   t.Contract,
		Price:      price,
		Change15m:  change(bars[last-pulseQuarter].Close, price),
		Change1h:   change(bars[last-pulseHour].Close, price),
		Change4h:   change(bars[last-4*pulseHour].Close, price),
		Change24h:  t.Change24h / 100,
		VolumeHour: volume,
		BuyShare:   share(bought, volume),
		Funding:    t.Funding,
		VolumeDay:  t.QuoteVolume,
		Beta:       week.beta,
		Sigma:      week.sigma,
		Runs:       week.runs,
	}

	pulse.Hours = recentHours(week.hourly, hoursOf(bars))

	quarterAll, quarterBought := sumBars(quarter)
	pulse.QuarterBuyShare = share(quarterBought, quarterAll)

	closed := bars[last-pulseHour : last]
	closedVolume, _ := sumBars(closed)

	if usual := usualVolume(closed, week); usual > 0 {
		pulse.VolumeSurge, pulse.Known = closedVolume/usual, week.sigma > 0
	} else if before > 0 {
		pulse.VolumeSurge = volume / (before * pulseHour / float64(len(earlier)))
	}

	closedQuarter := bars[last-pulseQuarter : last]
	quarterVolume, _ := sumBars(closedQuarter)

	if usual := usualVolume(closedQuarter, week); usual > 0 {
		pulse.QuarterSurge = quarterVolume / usual
	}

	if high > low {
		pulse.RangePosition = (price - low) / (high - low)
	}

	pulse.LegMinutes, pulse.LegMove = leg(bars[len(bars)-pulseLeg:], pulse.Change1h)

	return pulse, true
}

func leg(bars []Candle, direction float64) (int, float64) {
	price := bars[len(bars)-1].Close
	from, edge := 0, bars[0].Low

	if direction < 0 {
		edge = bars[0].High
	}

	for i, b := range bars {
		if direction >= 0 && b.Low <= edge {
			from, edge = i, b.Low
		}

		if direction < 0 && b.High >= edge {
			from, edge = i, b.High
		}
	}

	if edge <= 0 || direction == 0 {
		return 0, 0
	}

	return (len(bars) - 1 - from) * barMinutes, price/edge - 1
}

func hoursOf(bars []Candle) []Hour {
	var hours []Hour

	for _, b := range bars {
		open := b.Open - b.Open%hourMillis

		if len(hours) == 0 || hours[len(hours)-1].Open != open {
			hours = append(hours, Hour{Open: open, High: b.High, Low: b.Low})
		}

		hour := &hours[len(hours)-1]
		hour.High, hour.Low = max(hour.High, b.High), min(hour.Low, b.Low)
		hour.Close = b.Close
		hour.Volume += b.Volume
	}

	return hours
}

func recentHours(kept, fresh []Hour) []Hour {
	if len(fresh) < 2 {
		return nil
	}

	fresh = fresh[1:]
	hours := make([]Hour, 0, len(kept)+len(fresh))

	for _, h := range kept {
		if h.Open < fresh[0].Open {
			hours = append(hours, h)
		}
	}

	hours = append(hours, fresh...)

	return hours[max(0, len(hours)-hoursKept):]
}

func share(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}

	return part / whole
}

func place(price float64, bars []Candle) float64 {
	high, low := extremes(bars)

	if high <= low {
		return 0
	}

	return (price - low) / (high - low)
}

func usualVolume(bars []Candle, week Week) float64 {
	var sum float64

	for day := 1; day <= weekDays; day++ {
		for _, b := range bars {
			v, seen := week.slots[b.Open-b.Open%weekSlot-int64(day)*dayMillis]
			if !seen {
				return 0
			}

			sum += v / pulseQuarter
		}
	}

	return sum / weekDays
}

func sumBars(bars []Candle) (float64, float64) {
	var volume, bought float64

	for _, b := range bars {
		volume += b.Volume
		bought += b.Bought
	}

	return volume, bought
}

func extremes(bars []Candle) (float64, float64) {
	high, low := bars[0].High, bars[0].Low

	for _, b := range bars {
		high, low = max(high, b.High), min(low, b.Low)
	}

	return high, low
}

func change(from, to float64) float64 {
	if from <= 0 {
		return 0
	}

	return to/from - 1
}
