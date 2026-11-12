package binance

import (
	"math"
	"strconv"
)

type contractRules struct {
	step        float64
	tick        float64
	minNotional float64
}

func (r contractRules) smallest(price float64) float64 {
	if price <= 0 || r.step <= 0 {
		return r.minNotional
	}

	return math.Ceil(r.minNotional/price/r.step-1e-9) * r.step * price
}

type exchangeInfo struct {
	Symbols []struct {
		Symbol  string `json:"symbol"`
		Status  string `json:"status"`
		Filters []struct {
			Type     string `json:"filterType"`
			Step     string `json:"stepSize"`
			Tick     string `json:"tickSize"`
			Notional string `json:"notional"`
		} `json:"filters"`
	} `json:"symbols"`
}

func (info exchangeInfo) rules() map[string]contractRules {
	out := make(map[string]contractRules, len(info.Symbols))

	for _, s := range info.Symbols {
		if s.Status != tradingStatus {
			continue
		}

		var r contractRules

		for _, f := range s.Filters {
			switch f.Type {
			case "LOT_SIZE":
				r.step = parse(f.Step)
			case "PRICE_FILTER":
				r.tick = parse(f.Tick)
			case "MIN_NOTIONAL":
				r.minNotional = parse(f.Notional)
			}
		}

		out[s.Symbol] = r
	}

	return out
}

func floorTo(v, step float64) float64 {
	if step <= 0 {
		return v
	}

	return math.Floor(v/step+1e-9) * step
}

func roundTo(v, step float64) float64 {
	if step <= 0 {
		return v
	}

	return math.Round(v/step) * step
}

func decimal(v, step float64) string {
	places := 0

	for step > 0 && step < 1 && places < 10 {
		step *= 10
		places++
	}

	return strconv.FormatFloat(v, 'f', places, 64)
}
