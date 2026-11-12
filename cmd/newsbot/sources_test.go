package main

import (
	"slices"
	"testing"

	"newsbot/internal/httpclient"
	"newsbot/internal/monitor"
)

func names(sources []monitor.Source) []string {
	out := make([]string, 0, len(sources))

	for _, s := range sources {
		out = append(out, s.Name())
	}

	return out
}

func TestAnnouncementsNeedTheKeyAndADisabledSourceIsLeftOut(t *testing.T) {
	client := httpclient.New()

	bare := names(buildSources(config{Disabled: map[string]bool{}}, client))
	if !slices.Equal(bare, []string{"defillama-hacks", "stablecoin-depeg"}) {
		t.Errorf("without a key %v", bare)
	}

	keyed := names(buildSources(config{BinanceKey: "k", BinanceSecret: "s", Disabled: map[string]bool{"stablecoin-depeg": true}}, client))
	if !slices.Equal(keyed, []string{"defillama-hacks", "binance-stream"}) {
		t.Errorf("with a key and a disabled source %v", keyed)
	}
}
