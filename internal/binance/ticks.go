package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/websocket"
)

const (
	ticksURL      = "wss://fstream.binance.com/market/ws/!miniTicker@arr"
	ticksSilence  = time.Minute
	ticksLifetime = 23 * time.Hour
)

type Ticks struct{}

func (Ticks) Stream(ctx context.Context, each func([]market.Tick)) error {
	return websocket.Listen(ctx, websocket.Session{
		URL:      ticksURL,
		Silence:  ticksSilence,
		Lifetime: ticksLifetime,
		Each: func(message []byte) error {
			ticks, err := parseTicks(message)
			if err != nil {
				return err
			}

			each(ticks)

			return nil
		},
	})
}

type rawMiniTicker struct {
	Type   string `json:"e"`
	Event  int64  `json:"E"`
	Symbol string `json:"s"`
	Close  string `json:"c"`
}

func parseTicks(message []byte) ([]market.Tick, error) {
	var raws []rawMiniTicker
	if err := json.Unmarshal(message, &raws); err != nil {
		return nil, fmt.Errorf("mini tickers: %w", err)
	}

	ticks := make([]market.Tick, 0, len(raws))

	for _, r := range raws {
		price := parse(r.Close)
		if price <= 0 {
			continue
		}

		ticks = append(ticks, market.Tick{
			Contract: r.Symbol,
			Asset:    assetOf(r.Symbol),
			Price:    price,
			Time:     time.UnixMilli(r.Event).UTC(),
		})
	}

	return ticks, nil
}
