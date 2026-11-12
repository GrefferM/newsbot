package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/market"
	"newsbot/internal/trading"
)

type hourlyMarket struct{}

func (hourlyMarket) Get(_ context.Context, _, address string) ([]byte, error) {
	start := time.Now().UTC().Truncate(time.Hour).Add(-weekHours * time.Hour)

	switch {
	case strings.HasPrefix(address, "/fapi/v1/klines"):
		var rows [][]any

		for i := range weekHours + 1 {
			price, quote := 1.0, 100.0

			if i == weekHours {
				price, quote = 0.9, 400
			}

			rows = append(rows, []any{start.Add(time.Duration(i) * time.Hour).UnixMilli(), fmt.Sprint(price), fmt.Sprint(price * 1.01), fmt.Sprint(price * 0.99), fmt.Sprint(price), "0", 0, fmt.Sprint(quote), 1, "0", fmt.Sprint(quote * 0.3), "0"})
		}

		return json.Marshal(rows)
	case strings.HasPrefix(address, openInterestPath):
		return json.Marshal([]map[string]any{{"sumOpenInterestValue": "1000"}, {"sumOpenInterestValue": "1100"}, {"sumOpenInterestValue": "1210"}})
	default:
		return json.Marshal([]map[string]any{{"longAccount": "0.5"}, {"longAccount": "0.52"}, {"longAccount": "0.55"}})
	}
}

func TestThePlanSheetGivesTheCardAndTheRangesToSetLevelsFrom(t *testing.T) {
	server := ServerTool{
		Recent: stubEvents{}, Trader: stubAccount{}, Watchlist: stubCoins{}, Research: &shelf{}, Plans: plans{}, Watched: watched{},
		Live: liveQuotes{"MOVE": market.Ticker{Contract: "MOVEUSDT", Price: 0.9, QuoteVolume: 5e6}}, Radar: board{}, Agenda: calendar{},
		Journal: journal{}, Rules: DeskRules{Limits: trading.Limits{MaxExposure: 1}},
	}

	got, err := use(t, SheetTool{Server: server, Market: hourlyMarket{}}, `{"asset":"move"}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	var sheet planSheet

	if err := json.Unmarshal([]byte(got), &sheet); err != nil || sheet.Market == nil {
		t.Fatalf("sheet %s", got)
	}

	m := sheet.Market

	if sheet.Card.Contract != "MOVEUSDT" || len(m.Levels) != len(levelHours) || m.Levels[0].Low != 0.891 || m.Levels[3].High != 1.01 {
		t.Errorf("the card and the ranges: %+v", sheet)
	}

	if m.VolumeHour != 400 || m.UsualHour != 100 || m.BuyHour != 0.3 || m.Change1h > -0.099 {
		t.Errorf("the last hour against the usual: %+v", m)
	}

	if m.Interest != 1210 || m.Interest1h < 0.099 || m.Longs24h < 0.049 {
		t.Errorf("positioning and its change: %+v", m)
	}

	empty, _ := use(t, SheetTool{Server: ServerTool{Recent: stubEvents{}, Trader: stubAccount{}, Watchlist: stubCoins{}, Research: &shelf{}, Plans: plans{}, Watched: watched{},
		Live: liveQuotes{}, Radar: board{}, Agenda: calendar{}, Journal: journal{}}, Market: hourlyMarket{}}, `{"asset":"GONE"}`, &agents.Turn{})

	if !strings.Contains(empty, "перп не знайдено") || strings.Contains(empty, `"market"`) {
		t.Errorf("an instrument without a perpetual has no market to set levels from: %s", empty)
	}
}
