package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"newsbot/internal/autotrade"
	"newsbot/internal/binance"
	"newsbot/internal/filestore"
	"newsbot/internal/news"
	"newsbot/internal/trading"
	"newsbot/internal/websocket"
)

func buildAuto(cfg config, futures *binance.Futures, trader *trading.Trader, log *slog.Logger) (*autotrade.Service, error) {
	file := filestore.NewJSONFile[autotrade.State](cfg.AutoPath)

	saved, err := file.Load(autotrade.State{})
	if err != nil {
		return nil, fmt.Errorf("load auto trading state: %w", err)
	}

	return autotrade.NewService(autotrade.Deps{
		Prices: binance.Ticks{},
		Market: futures,
		Trader: trader,
		Store:  file,
		Log:    log,
	}, saved), nil
}

const (
	autoName       = "autotrade"
	candleRetry    = 5 * time.Second
	candleRetryMax = 5 * time.Minute
	candleSteady   = time.Minute
)

type candleFeed struct {
	futures *binance.Futures
	log     *slog.Logger
}

func (f candleFeed) Run(ctx context.Context, _ chan<- news.Event) {
	retry := websocket.Retry{Min: candleRetry, Max: candleRetryMax, Steady: candleSteady}

	retry.Keep(ctx, f.futures.Watch, warn(f.log, "candle stream"))
}
