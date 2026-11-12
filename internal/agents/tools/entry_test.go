package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
	"newsbot/internal/trading"
)

type sheetMarket struct {
	event time.Time
}

func (m sheetMarket) Get(_ context.Context, _, address string) ([]byte, error) {
	start := m.event.Add(-2 * time.Hour).Truncate(barWidth)

	switch {
	case strings.HasPrefix(address, "/fapi/v1/klines"):
		var rows [][]any

		for i := range 36 {
			open := start.Add(time.Duration(i) * barWidth)
			price, quote := 1.0, 1000.0

			if !open.Before(m.event) {
				price, quote = 0.95, 3000
			}

			rows = append(rows, []any{open.UnixMilli(), fmt.Sprint(price), fmt.Sprint(price * 1.001), fmt.Sprint(price * 0.999), fmt.Sprint(price), "0", 0, fmt.Sprint(quote), 1, "0", fmt.Sprint(quote * 0.4), "0"})
		}

		return json.Marshal(rows)
	case strings.HasPrefix(address, "/futures/data/openInterestHist"):
		return json.Marshal([]map[string]any{
			{"timestamp": m.event.Add(-time.Hour).UnixMilli(), "sumOpenInterestValue": "1000000"},
			{"timestamp": m.event.Add(time.Hour).UnixMilli(), "sumOpenInterestValue": "1200000"},
		})
	default:
		return json.Marshal([]map[string]any{
			{"timestamp": m.event.Add(-time.Hour).UnixMilli(), "longAccount": "0.55"},
			{"timestamp": m.event.Add(time.Hour).UnixMilli(), "longAccount": "0.60"},
		})
	}
}

func TestThePlanCheckTellsWhatTheMarketDidSinceThePlansEvent(t *testing.T) {
	event := time.Now().UTC().Add(-50 * time.Minute)
	plan := research.Scenario{Direction: research.DirectionShort, WakeBelow: 0.97, SizePct: 100, Stop: 1.02, Take: 0.9}
	busy := trading.Book{}.Funded(1000, 0)
	busy.Positions = []trading.Position{{Size: 0.5}}

	tool := EntryTool{
		Market:   sheetMarket{event: event},
		Research: &shelf{kept: []research.Finding{{Asset: "UNLK", Published: "2026-11-14", Text: "Розблокування перенесено"}}},
		Live: liveQuotes{"UNLK": market.Ticker{
			Contract: "UNLKUSDT", Price: 0.95, Funding: 0.0001, FundingHours: 8, QuoteVolume: 5e6,
		}},
		Watched: watched{
			{Asset: "UNLK", Plan: plan, State: autopilot.Armed, Armed: event.Add(-time.Hour), After: event},
			{Asset: "UNLK", Plan: plan, State: autopilot.Refuted},
		},
		Trader: bookOf(busy),
		Recent: stubEvents{{Title: "Binance Will Delist UNLK", Symbols: []news.Symbol{"UNLK"}, Time: event.Add(time.Minute)}},
		Agenda: calendar{schedule.NewEntry("ФРС", event.Add(time.Hour), "FOMC Meeting", "")},
		Rules:  DeskRules{Limits: trading.Limits{MaxExposure: 1, MinVolume: 1e6, CrowdedFunding: -0.0005}},
	}

	got, err := use(t, tool, `{"asset":"unlk"}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	var sheets []checkSheet

	if err := json.Unmarshal([]byte(got), &sheets); err != nil || len(sheets) != 1 {
		t.Fatalf("one armed plan, one sheet: %s", got)
	}

	s := sheets[0]

	if s.SinceIs != sinceEvent || s.PriceThen != 1 || s.Move > -0.049 || !s.Crossed || s.Past <= 0 || s.BuyShare != 0.4 {
		t.Errorf("the move since the event: %+v", s)
	}

	if s.VolumeHour < 2.5*s.VolumeUsual || s.InterestThen != 1e6 || s.InterestNow != 1.2e6 || s.LongsThen != 0.55 {
		t.Errorf("volume and positioning since the event: %+v", s)
	}

	if s.Checks[1].Passes || len(s.Events) != 1 || len(s.News) != 1 || len(s.Calendar) != 1 || len(s.Bars) != checkRecent {
		t.Errorf("the plan's own size does not fit the free exposure, and the bot's events and the calendar are told: %+v", s)
	}

	if object(map[string]any{})["required"] != nil {
		t.Error("a tool without required arguments names no list of them: a null there is refused by the model")
	}

	if _, err := use(t, EntryTool{Watched: watched{}}, `{"asset":"ZEC"}`, &agents.Turn{}); err == nil {
		t.Error("a coin with no watched plan has no sheet")
	}
}
