package tools

import (
	"testing"

	"newsbot/internal/agents"
	"newsbot/internal/watchlist"
)

func TestFinalistsAreCoinsWithAPerpAndNoMoreThanTheLimit(t *testing.T) {
	tool := PickTool{Limit: 2, Watchlist: stubCoins{
		{Coin: watchlist.Coin{Asset: "WLD"}}, {Coin: watchlist.Coin{Asset: "ZRO"}}, {Coin: watchlist.Coin{Asset: "BLUR"}},
	}}
	turn := &agents.Turn{}

	refused := []string{
		`{"symbol":"NOPE","direction":"шорт","why":"слабка"}`,
		`{"symbol":"WLD","direction":"вбік","why":"слабка"}`,
		`{"symbol":"WLD","direction":"шорт","why":" "}`,
	}

	for _, args := range refused {
		if _, err := use(t, tool, args, turn); err == nil {
			t.Errorf("passed: %s", args)
		}
	}

	for _, symbol := range []string{"wld", "WLD", "ZRO", "BLUR"} {
		_, _ = use(t, tool, `{"symbol":"`+symbol+`","direction":"шорт","why":"слабка і з новиною"}`, turn)
	}

	if len(turn.Picks) != 2 || turn.Picks[0].Asset != "WLD" || turn.Picks[1].Asset != "ZRO" {
		t.Errorf("picks %+v", turn.Picks)
	}
}
