package binance

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/websocket"
)

const (
	candlesURL       = "wss://fstream.binance.com/market/ws"
	candleInterval   = market.PulseInterval
	candleStep       = 5 * time.Minute
	keptCandles      = 160
	seedCandles      = market.PulseBars
	candleStreams    = 1000
	candlesPerAsk    = 200
	candleAskPause   = 250 * time.Millisecond
	candleSilence    = time.Minute
	candlesLifetime  = 23 * time.Hour
	candleStreamName = "@kline_5m"
)

type rawKline struct {
	Type   string `json:"e"`
	Event  int64  `json:"E"`
	Symbol string `json:"s"`
	K      struct {
		FirstTrade  int64  `json:"f"`
		LastTrade   int64  `json:"L"`
		Open        int64  `json:"t"`
		Close       int64  `json:"T"`
		OpenPrice   string `json:"o"`
		High        string `json:"h"`
		Low         string `json:"l"`
		ClosePrice  string `json:"c"`
		Volume      string `json:"v"`
		QuoteVolume string `json:"q"`
		Trades      int64  `json:"n"`
		BoughtBase  string `json:"V"`
		BoughtQuote string `json:"Q"`
	} `json:"k"`
}

func (f *Futures) Watch(ctx context.Context) error {
	if err := f.refresh(ctx); err != nil {
		return err
	}

	contracts := f.liquid(candleStreams, nil)

	f.watch(true)
	defer f.watch(false)

	return websocket.Listen(ctx, websocket.Session{
		URL:      candlesURL,
		Silence:  candleSilence,
		Lifetime: candlesLifetime,
		Opened: func(conn *websocket.Conn) error {
			return subscribe(ctx, conn, contracts)
		},
		Each: func(message []byte) error {
			f.candle(message)

			return nil
		},
	})
}

func subscribe(ctx context.Context, conn *websocket.Conn, contracts []market.Ticker) error {
	for i := 0; i < len(contracts); i += candlesPerAsk {
		var streams []string

		for _, t := range contracts[i:min(i+candlesPerAsk, len(contracts))] {
			streams = append(streams, strings.ToLower(t.Contract)+candleStreamName)
		}

		ask, err := json.Marshal(map[string]any{"method": "SUBSCRIBE", "params": streams, "id": i/candlesPerAsk + 1})
		if err != nil {
			return err
		}

		if err := conn.WriteText(ask); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(candleAskPause):
		}
	}

	return nil
}

func (f *Futures) watch(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.watching = on
	f.live = map[string][]market.Candle{}
}

func (f *Futures) candle(message []byte) {
	var raw rawKline

	if json.Unmarshal(message, &raw) != nil || raw.Type != "kline" {
		return
	}

	fresh := market.Candle{
		Open:   raw.K.Open,
		High:   parse(raw.K.High),
		Low:    parse(raw.K.Low),
		Close:  parse(raw.K.ClosePrice),
		Volume: parse(raw.K.QuoteVolume),
		Bought: parse(raw.K.BoughtQuote),
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.watching {
		f.live[raw.Symbol] = merge(f.live[raw.Symbol], []market.Candle{fresh})
	}
}

func (f *Futures) fiveMinutes(ctx context.Context, contract string, limit int) ([]market.Candle, error) {
	kept, live, watching := f.fromLive(contract, limit, time.Now())
	if live {
		return kept, nil
	}

	wanted := limit
	if watching {
		wanted = max(limit, seedCandles)
	}

	rows, err := fetchKlines(ctx, f.http, klinesURL(contract, candleInterval, wanted))
	if err != nil {
		return nil, err
	}

	candles := candlesOf(rows)

	f.seed(contract, candles)

	return candles[max(0, len(candles)-limit):], nil
}

func (f *Futures) seed(contract string, candles []market.Candle) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.watching {
		f.live[contract] = merge(candles, f.live[contract])
	}
}

func (f *Futures) fromLive(contract string, limit int, now time.Time) ([]market.Candle, bool, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	kept := f.live[contract]

	if !f.watching || len(kept) == 0 {
		return nil, false, f.watching
	}

	filled := flatUntil(kept, now.Truncate(candleStep).UnixMilli())

	if len(filled) < limit {
		return nil, false, true
	}

	return filled[len(filled)-limit:], true, true
}

func merge(older, newer []market.Candle) []market.Candle {
	out := make([]market.Candle, 0, len(older)+len(newer))

	for _, c := range older {
		if len(newer) == 0 || c.Open < newer[0].Open {
			out = append(out, c)
		}
	}

	for _, c := range newer {
		if last := len(out) - 1; last >= 0 && out[last].Open >= c.Open {
			if out[last].Open == c.Open {
				out[last] = c
			}

			continue
		}

		out = append(out, c)
	}

	if len(out) > keptCandles {
		out = out[len(out)-keptCandles:]
	}

	return out
}

func flatUntil(kept []market.Candle, bucket int64) []market.Candle {
	out := append([]market.Candle(nil), kept...)
	step := candleStep.Milliseconds()

	for {
		last := out[len(out)-1]
		next := last.Open + step

		if next > bucket {
			return out
		}

		out = append(out, market.Candle{Open: next, High: last.Close, Low: last.Close, Close: last.Close})
	}
}
