package main

import (
	"context"
	"log/slog"
	"time"

	"newsbot/internal/binance"
	"newsbot/internal/market"
	"newsbot/internal/monitor"
	"newsbot/internal/radar"
)

const radarEvery = time.Minute

func buildRadar(cfg config, futures *binance.Futures, status *monitor.Status, shelf radar.Shelf, log *slog.Logger) *radar.Service {
	return radar.NewService(radar.Deps{
		Coins:  futures,
		TradFi: tradfiMarket{futures: futures},
		Rule:   cfg.Radar,
		Health: status,
		Shelf:  shelf,
		Every:  radarEvery,
		Log:    log,
	})
}

type tradfiMarket struct {
	futures *binance.Futures
}

func (m tradfiMarket) Pulse(ctx context.Context) ([]market.Pulse, error) {
	return m.futures.TradFi(ctx)
}
