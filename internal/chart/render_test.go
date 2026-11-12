package chart

import (
	"bytes"
	"image/png"
	"math"
	"testing"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

func bars(n int) []market.Bar {
	start := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	out := make([]market.Bar, n)

	for i := range out {
		out[i] = market.Bar{
			Open:        start.Add(time.Duration(i) * 5 * time.Minute),
			Close:       0.3 + 0.02*math.Sin(float64(i)/10),
			QuoteVolume: float64(1000 + i%17*300),
		}
	}

	return out
}

func TestRenderProducesPNG(t *testing.T) {
	b := bars(288)

	data, err := Render(Spec{
		Contract: "CYBERUSDT",
		Interval: "5M  24H",
		Bars:     b,
		Marker:   b[270].Open,
		Trend:    news.Down,
	})
	if err != nil {
		t.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Fatalf("size %v", img.Bounds())
	}
}

func TestRenderRejectsTooFewBars(t *testing.T) {
	if _, err := Render(Spec{Bars: bars(3)}); err == nil {
		t.Fatal("rendered chart from 3 bars")
	}
}
