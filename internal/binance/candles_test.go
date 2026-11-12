package binance

import (
	"fmt"
	"testing"
	"time"

	"newsbot/internal/market"
)

func kline(open time.Time, price string) []byte {
	return fmt.Appendf(nil,
		`{"e":"kline","E":1794986622400,"s":"MOVEUSDT","k":{"t":%d,"T":%d,"s":"MOVEUSDT","i":"5m","f":100,"L":120,"o":"%s","c":"%s","h":"%s","l":"%s","v":"10","n":3,"x":false,"q":"20","V":"4","Q":"8","B":"0"}}`,
		open.UnixMilli(), open.Add(candleStep).UnixMilli()-1, price, price, price, price,
	)
}

func history(from time.Time, count int) []market.Candle {
	var out []market.Candle

	for i := range count {
		open := from.Add(time.Duration(i) * candleStep)
		out = append(out, market.Candle{Open: open.UnixMilli(), High: 1.1, Low: 0.9, Close: 1, Volume: 5, Bought: 2})
	}

	return out
}

func TestLiveCandlesServeTheReadOnceSeededAndFollowTheStream(t *testing.T) {
	f := NewFutures(nil, time.Minute)
	now := time.Date(2026, 11, 18, 10, 2, 0, 0, time.UTC)
	bucket := now.Truncate(candleStep)

	if _, live, watching := f.fromLive("MOVEUSDT", 3, now); live || watching {
		t.Fatal("without the stream the candles are read from Binance")
	}

	f.watch(true)

	if _, live, watching := f.fromLive("MOVEUSDT", 3, now); live || !watching {
		t.Fatal("an unseeded contract is read from Binance once")
	}

	f.seed("MOVEUSDT", history(bucket.Add(-4*candleStep), 5))
	f.candle(kline(bucket, "1.5"))

	kept, live, _ := f.fromLive("MOVEUSDT", 3, now)

	if !live || len(kept) != 3 || kept[2].Open != bucket.UnixMilli() || kept[2].Close != 1.5 || kept[2].Volume != 20 || kept[2].Bought != 8 {
		t.Fatalf("kept %+v", kept)
	}

	later := now.Add(2 * candleStep)
	kept, live, _ = f.fromLive("MOVEUSDT", 3, later)

	if !live || kept[2].Open != later.Truncate(candleStep).UnixMilli() || kept[2].Close != 1.5 || kept[2].Volume != 0 {
		t.Fatalf("five minutes without a trade are a flat candle: %+v", kept)
	}

	f.watch(false)

	if _, live, watching := f.fromLive("MOVEUSDT", 3, now); live || watching {
		t.Error("a dropped stream forgets its candles")
	}
}

func TestMergeKeepsOneRowPerCandleAndTheNewestOfIt(t *testing.T) {
	from := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	older := history(from, 3)
	newer := history(from.Add(2*candleStep), 2)
	newer[0].Close = 2

	got := merge(older, newer)

	if len(got) != 4 || got[2].Close != 2 {
		t.Fatalf("got %+v", got)
	}

	if long := merge(history(from, keptCandles+10), nil); len(long) != keptCandles {
		t.Errorf("kept %d", len(long))
	}
}
