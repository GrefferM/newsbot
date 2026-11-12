package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/watchlist"
)

func TestTheScreenFiltersSortsAndCutsInOneCall(t *testing.T) {
	found := &shelf{kept: []research.Finding{
		{Asset: "SOL", Published: time.Now().UTC().Format(time.RFC3339), Text: "Суд"},
		{Asset: "ACT", Published: "2026-10-01", Text: "Давня новина"},
	}}
	tool := ScreenTool{Research: found, Live: liveQuotes{"JCT": {Price: 0.0064, Change24h: -31.6, Funding: -0.004, FundingHours: 4}}, Watchlist: stubCoins{
		{Coin: watchlist.Coin{Asset: "BLUR", Tagged: true, Volume: 600_000, DayRange: 0.08, Depth: 20_000}},
		{Coin: watchlist.Coin{Asset: "ACT", Tagged: true, Volume: 25_000_000, DayRange: 0.12, Depth: 90_000}},
		{Coin: watchlist.Coin{Asset: "SOL", Volume: 9e8, DayRange: 0.04, Depth: 5e6}},
		{Coin: watchlist.Coin{Asset: "JCT", Volume: 6e8, DayRange: 0.5, Depth: 37_000}},
	}}

	got, err := use(t, tool, `{"min":{"mean_day_range_30d_fraction":0.07,"perp_depth_within_1pct_usd":30000},"sort_by":"mean_day_range_30d_fraction","order":"desc","limit":1}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, `"matched":2`) || !strings.Contains(got, `"asset":"JCT"`) || strings.Contains(got, `"asset":"ACT"`) {
		t.Errorf("screen %s", got)
	}

	tagged, _ := use(t, tool, `{"only_tagged":true}`, &agents.Turn{})

	if !strings.Contains(tagged, `"matched":2`) || strings.Contains(tagged, `"asset":"SOL"`) {
		t.Errorf("tagged %s", tagged)
	}

	fresh, _ := use(t, tool, `{"fresh_news_only":true}`, &agents.Turn{})

	if !strings.Contains(fresh, `"matched":1`) || !strings.Contains(fresh, `"asset":"SOL"`) {
		t.Errorf("only the coin with a fresh finding is left: %s", fresh)
	}

	falling, _ := use(t, tool, `{"sort_by":"perp_change_24h_fraction","order":"asc","limit":1}`, &agents.Turn{})

	for _, want := range []string{`"asset":"JCT"`, `"perp_change_24h_fraction":-0.316`, `"funding_paid_every_hours":4`} {
		if !strings.Contains(falling, want) {
			t.Errorf("no %s in %s", want, falling)
		}
	}

	if _, err := use(t, tool, `{"sort_by":"beauty"}`, &agents.Turn{}); err == nil {
		t.Error("an unknown field passed")
	}
}

type liveQuotes map[news.Symbol]market.Ticker

func (l liveQuotes) Ticker(_ context.Context, s news.Symbol) (market.Ticker, bool, error) {
	ticker, ok := l[s]

	return ticker, ok, nil
}
