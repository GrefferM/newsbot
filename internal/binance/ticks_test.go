package binance

import (
	"testing"
	"time"
)

func TestMiniTickersBecomeTicksOfTheirAsset(t *testing.T) {
	message := []byte(`[
		{"e":"24hrMiniTicker","E":1794902917455,"s":"1000PEPEUSDT","c":"0.0038921","o":"0.0039","h":"0.004","l":"0.0037","v":"1","q":"1"},
		{"e":"24hrMiniTicker","E":1794902917455,"s":"BADUSDT","c":"0"}
	]`)

	ticks, err := parseTicks(message)
	if err != nil {
		t.Fatal(err)
	}

	if len(ticks) != 1 {
		t.Fatalf("ticks %+v", ticks)
	}

	tick := ticks[0]

	if tick.Contract != "1000PEPEUSDT" || tick.Asset != "PEPE" || tick.Price != 0.0038921 ||
		!tick.Time.Equal(time.UnixMilli(1794902917455)) {
		t.Errorf("tick %+v", tick)
	}
}
