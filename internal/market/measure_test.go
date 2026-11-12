package market

import (
	"math"
	"testing"
)

func TestThePulseReadsTheLastHourAgainstTheHoursBefore(t *testing.T) {
	bars := make([]Candle, 0, PulseBars)

	for range PulseBars - pulseHour {
		bars = append(bars, Candle{High: 101, Low: 99, Close: 100, Volume: 1000, Bought: 500})
	}

	for i := range pulseHour {
		price := 100 + float64(i+1)
		bought := 800.0

		if i < pulseQuarter {
			bought = 400
		}

		bars = append(bars, Candle{High: price, Low: price - 1, Close: price, Volume: 1000 * float64(1+i/pulseQuarter), Bought: bought * float64(1+i/pulseQuarter)})
	}

	pulse, ok := Measure(Ticker{Contract: "1000PEPEUSDT", Change24h: 12, Funding: 0.0001, QuoteVolume: 5e6}, bars, Week{})
	if !ok {
		t.Fatal("no pulse from a full window")
	}

	near := func(got, want float64) bool {
		return math.Abs(got-want) < 1e-9
	}

	if pulse.Price != 112 || !near(pulse.Change15m, 112.0/109-1) || !near(pulse.Change1h, 0.12) || !near(pulse.Change4h, 0.12) {
		t.Errorf("moves %+v", pulse)
	}

	if !near(pulse.VolumeSurge, 2.5) || !near(pulse.BuyShare, 0.76) || pulse.RangePosition != 1 || !near(pulse.Change24h, 0.12) || pulse.Known {
		t.Errorf("volume and range %+v", pulse)
	}

	if !near(pulse.QuarterBuyShare, 0.8) {
		t.Errorf("takers of the last quarter %+v", pulse)
	}

	if _, ok := Measure(Ticker{Contract: "NEWUSDT"}, bars[:10], Week{}); ok {
		t.Error("a pulse from a short window")
	}
}

func TestAWeekGivesTheBetaTheUsualMoveAndTheUsualVolumeOfEachHour(t *testing.T) {
	const step = int64(15 * 60 * 1000)

	var coin, btc []Candle

	price, bench := 100.0, 100.0

	for i := range WeekBars {
		x := 0.002 * math.Sin(float64(i))
		noise := 0.001 * math.Cos(float64(i)*1.7)
		bench *= 1 + x
		price *= 1 + 2*x + noise
		open := int64(i) * step
		btc = append(btc, Candle{Open: open, Close: bench})
		coin = append(coin, Candle{Open: open, Close: price, High: price, Low: price, Volume: float64(1000 + i%96)})
	}

	week := WeekOf(coin, Returns(btc))

	if math.Abs(week.beta-2) > 0.05 || week.sigma <= 0 || week.sigma > 0.004 {
		t.Fatalf("beta %v sigma %v", week.beta, week.sigma)
	}

	var hour []Candle

	want := 0.0

	for _, b := range coin[len(coin)-weekHour:] {
		want += float64(1000 + int((b.Open/step)%96))

		for part := range int64(pulseQuarter) {
			hour = append(hour, Candle{Open: b.Open + part*step/pulseQuarter})
		}
	}

	if got := usualVolume(hour, week); math.Abs(got-want) > 1e-6 {
		t.Errorf("usual volume of this hour %v, want %v", got, want)
	}

	if usualVolume(hour, Week{}) != 0 {
		t.Error("a usual volume without a week of history")
	}
}

func TestTheVolumeIsComparedOnClosedBarsForTheHourAndForItsLastQuarter(t *testing.T) {
	const step = int64(5 * 60 * 1000)

	week := Week{slots: map[int64]float64{}, sigma: 0.01}
	bars := make([]Candle, 0, PulseBars)

	for i := range PulseBars {
		open := int64(10*288+i) * step
		bars = append(bars, Candle{Open: open, High: 101, Low: 99, Close: 100, Volume: 2000})
	}

	bars[len(bars)-1].Volume = 50

	for _, i := range []int{2, 3, 4} {
		bars[len(bars)-i].Volume = 5000
	}

	for day := int64(1); day <= 7; day++ {
		for _, b := range bars {
			week.slots[b.Open-b.Open%weekSlot-day*dayMillis] = 3000
		}
	}

	pulse, ok := Measure(Ticker{Contract: "ZECUSDT"}, bars, week)
	if !ok || !pulse.Known || math.Abs(pulse.VolumeSurge-2.75) > 1e-9 || math.Abs(pulse.QuarterSurge-5) > 1e-9 {
		t.Errorf("a bar just begun pulled the volume down: %+v", pulse)
	}
}

func TestAWeekWithoutABenchmarkMeasuresTheOwnMoveAlone(t *testing.T) {
	var bars []Candle

	price := 100.0

	for i := range WeekBars {
		price *= 1 + 0.002*math.Sin(float64(i))
		bars = append(bars, Candle{Open: int64(i) * 900000, Close: price, Volume: 1000})
	}

	week := WeekOf(bars, nil)

	if week.beta != 0 || week.sigma <= 0 || week.sigma > 0.004 {
		t.Errorf("beta %v sigma %v", week.beta, week.sigma)
	}
}

func TestTheRunSoFarIsMeasuredFromTheEdgeItLeft(t *testing.T) {
	bars := make([]Candle, 0, pulseLeg)

	for range pulseLeg - 6 {
		bars = append(bars, Candle{High: 101, Low: 99, Close: 100})
	}

	bars[20].Low = 95

	for _, price := range []float64{96, 99, 102, 104, 107, 110} {
		bars = append(bars, Candle{High: price + 1, Low: price - 1, Close: price})
	}

	bars[pulseLeg-6].Low = 95

	minutes, move := leg(bars, 0.1)

	if minutes != 25 || math.Abs(move-(110.0/95-1)) > 1e-9 {
		t.Errorf("a rise is counted from its latest Low: %d min, %v", minutes, move)
	}

	if minutes, move := leg(bars, -0.01); minutes != 0 || math.Abs(move-(110.0/111-1)) > 1e-9 {
		t.Errorf("a fall is counted from the High: %d min, %v", minutes, move)
	}

	if minutes, move := leg(bars, 0); minutes != 0 || move != 0 {
		t.Errorf("no direction, no run: %d min, %v", minutes, move)
	}
}

func TestAWeekMeasuresTheRunOfTheLastDayAndThreeDays(t *testing.T) {
	var bars []Candle

	price := 100.0

	for i := range WeekBars {
		price *= 1 + 0.001*math.Sin(float64(i))

		if i >= WeekBars-24*weekHour {
			price *= 1.001
		}

		bars = append(bars, Candle{Open: int64(i) * weekSlot, Close: price, High: price, Low: price, Volume: 1000})
	}

	week := WeekOf(bars, nil)

	if len(week.runs) != 2 || week.runs[0].Hours != 24 || week.runs[1].Hours != 72 {
		t.Fatalf("runs %+v", week.runs)
	}

	day := week.runs[0]

	if day.Move < 0.09 || day.Move > 0.11 || day.Sigmas < 3 || day.Straight < 0.8 || day.Place < 0.99 {
		t.Errorf("a day of steady rise %+v", day)
	}

	if short := WeekOf(bars[:50], nil); len(short.runs) != 0 {
		t.Errorf("runs from too little history %+v", short.runs)
	}
}

func TestThePulseKeepsHourlyBarsFromTheWeekAndTheFreshBars(t *testing.T) {
	hour := int64(hourMillis)
	week := hoursOf([]Candle{
		{Open: 0, High: 2, Low: 1, Close: 1.5, Volume: 10},
		{Open: hour / 4, High: 3, Low: 1.2, Close: 2, Volume: 5},
		{Open: hour, High: 2.5, Low: 1.8, Close: 2.2, Volume: 7},
		{Open: 2 * hour, High: 9, Low: 9, Close: 9, Volume: 1},
	})

	if len(week) != 3 || week[0].High != 3 || week[0].Low != 1 || week[0].Close != 2 || week[0].Volume != 15 {
		t.Fatalf("bars are folded into their hours: %+v", week)
	}

	fresh := []Hour{{Open: hour}, {Open: 2 * hour, Close: 2.4}, {Open: 3 * hour, Close: 2.6}}
	hours := recentHours(week, fresh)

	if len(hours) != 4 || hours[1].Close != 2.2 || hours[2].Close != 2.4 || hours[3].Open != 3*hour {
		t.Errorf("fresh hours stand in for the kept ones, the first fresh one may be partial: %+v", hours)
	}

	if recentHours(week, fresh[:1]) != nil {
		t.Error("one fresh hour is not enough to tell which are whole")
	}
}
