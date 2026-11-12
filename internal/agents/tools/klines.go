package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	openInterestPath = "/futures/data/openInterestHist"
	longAccountsPath = "/futures/data/globalLongShortAccountRatio"
)

type candle struct {
	open                         time.Time
	o, h, l, c, quote, boughtUSD float64
}

func klines(ctx context.Context, m Market, contract, interval string, limit int) ([]candle, error) {
	body, err := m.Get(ctx, "futures", fmt.Sprintf("/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", contract, interval, limit))
	if err != nil {
		return nil, err
	}

	var rows [][]any

	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}

	out := make([]candle, 0, len(rows))

	for _, r := range rows {
		if len(r) < 11 {
			continue
		}

		out = append(out, candle{
			open:      time.UnixMilli(int64(numeric(r[0]))).UTC(),
			o:         numeric(r[1]),
			h:         numeric(r[2]),
			l:         numeric(r[3]),
			c:         numeric(r[4]),
			quote:     numeric(r[7]),
			boughtUSD: numeric(r[10]),
		})
	}

	if len(out) == 0 {
		return nil, errors.New("свічок немає")
	}

	return out, nil
}

func history(ctx context.Context, m Market, path, contract, period string, limit int) ([]map[string]any, error) {
	body, err := m.Get(ctx, "futures", fmt.Sprintf("%s?symbol=%s&period=%s&limit=%d", path, contract, period, limit))
	if err != nil {
		return nil, err
	}

	var rows []map[string]any

	return rows, json.Unmarshal(body, &rows)
}
