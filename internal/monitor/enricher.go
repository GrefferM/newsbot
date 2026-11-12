package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

const (
	maxEnrichedSymbols = 5
	enrichTimeout      = 15 * time.Second
)

var enrichable = map[news.Category]bool{news.ShortTrigger: true}

type Enricher struct {
	market MarketData
	log    *slog.Logger
}

func NewEnricher(market MarketData, log *slog.Logger) *Enricher {
	return &Enricher{market: market, log: log}
}

func (en *Enricher) Enrich(ctx context.Context, e *news.Event) {
	if en.market == nil || !enrichable[e.Category] {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, enrichTimeout)
	defer cancel()

	for i, s := range e.Symbols {
		if i == maxEnrichedSymbols {
			e.Details = append(e.Details, fmt.Sprintf("… ще %d", len(e.Symbols)-maxEnrichedSymbols))

			return
		}

		lines, err := en.describe(ctx, s)
		if err != nil {
			en.log.Warn("enrich failed", "symbol", s, "err", err)

			return
		}

		e.Details = append(e.Details, lines...)
	}
}

func (en *Enricher) describe(ctx context.Context, s news.Symbol) ([]string, error) {
	t, ok, err := en.market.Ticker(ctx, s)
	if err != nil {
		return nil, err
	}

	if !ok {
		return []string{fmt.Sprintf("%s: перпа на Binance немає", s)}, nil
	}

	lines := []string{fmt.Sprintf("%s: %s", s, t.Summary())}

	if change, ok := en.market.OpenInterestChange(ctx, t.Contract, market.MinOpenInterestWindow); ok {
		lines = append(lines, fmt.Sprintf("%s: %s", s, market.OpenInterestLine(change, market.MinOpenInterestWindow)))
	}

	return lines, nil
}
