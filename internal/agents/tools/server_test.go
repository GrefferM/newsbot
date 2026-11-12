package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/radar"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

type stubEvents []news.Event

func (s stubEvents) Latest(int) []news.Event {
	return s
}

type stubAccount struct{}

func (stubAccount) Sized(_ context.Context, m trading.Market) trading.Market {
	m.MinSize = 0.005

	return m
}

func (stubAccount) Book() trading.Book {
	return trading.Book{}.Funded(1000, 0)
}

type stubCoins []watchlist.Kept

func (s stubCoins) Coins() []watchlist.Kept {
	return s
}

func (s stubCoins) Kept() []watchlist.Kept {
	return nil
}

type livePrice float64

func (p livePrice) Ticker(context.Context, news.Symbol) (market.Ticker, bool, error) {
	return market.Ticker{Contract: "BLURUSDT", Price: float64(p), Change24h: -9.5}, true, nil
}

type plans map[string]research.Plan

func (p plans) Plan(asset string) (research.Plan, bool) {
	plan, kept := p[asset]

	return plan, kept
}

type watched []autopilot.Watch

func (w watched) Of(string) []autopilot.Watch {
	return w
}

type board radar.Board

func (b board) Latest() radar.Board {
	return radar.Board(b)
}

func (board) LatestTradFi() radar.Board {
	return radar.Board{}
}

type journal []agents.Record

func (j journal) Last(string, string, int) ([]agents.Record, error) {
	return j, nil
}

type shelf struct {
	kept []research.Finding
}

func (s *shelf) Of(asset string) []research.Finding {
	var out []research.Finding

	for _, f := range s.kept {
		if f.Asset == asset {
			out = append(out, f)
		}
	}

	return out
}

func (s *shelf) Keep(f research.Finding) error {
	s.kept = append(s.kept, f)

	return nil
}

func use(t *testing.T, tool agents.Tool, args string, turn *agents.Turn) (string, error) {
	t.Helper()

	return tool.Call(context.Background(), json.RawMessage(args), turn)
}

func TestTheCoinCardCarriesMarketNewsAndBotEvents(t *testing.T) {
	found := &shelf{kept: []research.Finding{{ID: "n1", Asset: "BLUR", Published: "2026-10-12", Text: "Сайт закривається"}}}
	server := ServerTool{
		Recent: stubEvents{
			{Title: "Monitoring Tag", Kind: "тег", Symbols: []news.Symbol{"BLUR"}, Time: time.Now()},
			{Title: "Blur: злам на $9M", Kind: "злам", Category: news.Incident, Symbols: []news.Symbol{"BLUR"}, Time: time.Now()},
		},
		Trader:    stubAccount{},
		Watchlist: stubCoins{{Coin: watchlist.Coin{Asset: "BLUR", Price: 0.2, Tagged: true, OpenInterest: 3_000_000}}},
		Research:  found,
		Plans:     plans{"BLUR": {Summary: "Слабка", Scenarios: []research.Scenario{{Condition: "Делістинг"}}}},
		Watched:   watched{{Plan: research.Scenario{Condition: "Пробій"}, State: autopilot.Armed, Checks: 1}},
		Live:      livePrice(0.15),
		Journal: journal{{Ideas: []agents.Idea{
			{Asset: "BLUR", Direction: "шорт", Why: "падає на обігу", Cancel: "вище 0.2"},
			{Asset: "ZEC", Direction: "лонг", Why: "інша монета", Cancel: "x"},
		}}},
		Rules: DeskRules{
			Limits:    trading.Limits{MaxExposure: 1, MinVolume: 1e6, CrowdedFunding: -0.0005},
			Hold:      24 * time.Hour,
			WatchDays: 7,
		},
		Agenda: calendar{schedule.NewEntry("ФРС", time.Date(2026, 12, 9, 19, 0, 0, 0, time.UTC), "FOMC Meeting", "")},
		Radar: board{Coins: []radar.Coin{{
			Pulse:   market.Pulse{Asset: "BLUR", Change1h: -0.11, VolumeSurge: 3.5, Sigma: 0.006},
			Wake:    &radar.Wake{Minutes: 10, Interest: 0.04, Longs: -0.03},
			Stories: []radar.Story{{Title: "Blur closes its site", Outlet: "X · @blur_io"}},
		}}},
	}

	card, err := use(t, server, `{"what":"coin","asset":"blur"}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`"monitoring_tag":true`, `"instrument":"криптомонета"`, `"open_interest_usd":3000000`, `"news":[{`, "Сайт закривається", "Monitoring Tag",
		`"price_usd":0.15`, `"price_source":"live"`, `"perp_change_24h_fraction":-0.095`, "Делістинг", `"state":"armed"`, "Пробій",
		`"woke_up":{"minutes_ago":10`, `"usual_1h_move_fraction":0.006`,
		`"analyst_ideas":[{`, `"market_calendar":[{`, "FOMC Meeting", "падає на обігу", `"position_closed_by_time_after_hours":24`, `"plan_watched_for_days":7`,
		`"check":"Ліквідність","passes":false`, incidentMeasured, `"smallest_order_pct_of_account":0.5`,
	} {
		if !strings.Contains(card, want) {
			t.Errorf("card lacks %s: %s", want, card)
		}
	}
}

type calendar []schedule.Entry

func (c calendar) Upcoming(time.Time, time.Duration) []schedule.Entry {
	return c
}
