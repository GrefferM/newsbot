package main

import (
	"slices"
	"time"

	"newsbot/internal/binance"
	"newsbot/internal/defillama"
	"newsbot/internal/httpclient"
	"newsbot/internal/monitor"
)

const (
	minute       = time.Minute
	tickerMaxAge = 90 * time.Second
)

func buildSources(cfg config, client *httpclient.Client) []monitor.Source {
	sources := []monitor.Source{
		defillama.NewHacks(client, cfg.Hacks, 2*minute),
		defillama.NewStablecoins(client, cfg.Depeg, 5*minute),
	}

	if cfg.BinanceKey != "" && cfg.BinanceSecret != "" {
		sources = append(sources, binance.NewStream(cfg.BinanceKey, cfg.BinanceSecret, cfg.Announcements))
	}

	return slices.DeleteFunc(sources, func(s monitor.Source) bool {
		return cfg.Disabled[s.Name()]
	})
}
