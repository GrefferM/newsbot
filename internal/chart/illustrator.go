package chart

import (
	"context"

	"newsbot/internal/binance"
	"newsbot/internal/news"
)

const (
	chartInterval = "5m"
	chartBars     = 288
	chartLabel    = "5M  24H"
	maxTried      = 3
)

type Illustrator struct {
	futures *binance.Futures
}

func NewIllustrator(futures *binance.Futures) *Illustrator {
	return &Illustrator{futures: futures}
}

func (il *Illustrator) Illustrate(ctx context.Context, e news.Event) ([]byte, bool) {
	for _, s := range e.Symbols[:min(len(e.Symbols), maxTried)] {
		if png, ok := il.render(ctx, s, e); ok {
			return png, true
		}
	}

	return nil, false
}

func (il *Illustrator) render(ctx context.Context, s news.Symbol, e news.Event) ([]byte, bool) {
	t, ok, err := il.futures.Ticker(ctx, s)
	if err != nil || !ok {
		return nil, false
	}

	bars, err := il.futures.Bars(ctx, t.Contract, chartInterval, chartBars)
	if err != nil {
		return nil, false
	}

	png, err := Render(Spec{
		Contract: t.Contract,
		Interval: chartLabel,
		Bars:     bars,
		Marker:   e.Time,
		Trend:    e.Trend,
	})
	if err != nil {
		return nil, false
	}

	return png, true
}
