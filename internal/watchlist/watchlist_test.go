package watchlist

import (
	"slices"
	"testing"
	"time"
)

func TestReasonsMarkTaggedAndThinCoins(t *testing.T) {
	rule := Rule{LowVolume: 300_000, ExcludedTags: []string{"stablecoin", "bStocks"}}

	cases := []struct {
		name    string
		coin    Coin
		reasons []string
	}{
		{"tagged and liquid", Coin{Asset: "STX", Tagged: true, HasSpot: true, SpotVolume: 1_700_000}, []string{ReasonTag}},
		{"tagged and thin", Coin{Asset: "MOVE", Tagged: true, HasSpot: true, SpotVolume: 100_000}, []string{ReasonTag, ReasonLowVolume}},
		{"thin only", Coin{Asset: "XVS", HasSpot: true, SpotVolume: 90_000}, []string{ReasonLowVolume}},
		{"liquid and untagged", Coin{Asset: "SOL", HasSpot: true, SpotVolume: 90_000_000}, nil},
		{"a perpetual without spot is not thin", Coin{Asset: "ARIA", Volume: 200_000_000}, nil},
		{"thin stablecoin", Coin{Asset: "EURI", Tags: []string{"stablecoin"}, HasSpot: true, SpotVolume: 50_000}, nil},
		{"thin tokenized stock", Coin{Asset: "NFLXB", Tags: []string{"bStocks"}, HasSpot: true, SpotVolume: 20_000}, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := rule.Reasons(c.coin); !slices.Equal(got, c.reasons) {
				t.Errorf("reasons %v, want %v", got, c.reasons)
			}
		})
	}
}

func TestTheScanKeepsOnlyTradableCoins(t *testing.T) {
	tradable := Coin{Asset: "SOL", Volume: 9e7, Contract: "SOLUSDT", HasTrend: true, HasChanges: true}

	noPerp := tradable
	noPerp.Asset, noPerp.Contract = "RAY", ""

	young := tradable
	young.Asset, young.HasTrend = "YOUNG", false

	noHistory := tradable
	noHistory.Asset, noHistory.HasChanges = "NEW", false

	noVolume := tradable
	noVolume.Asset, noVolume.Volume = "DEAD", 0

	scan := Rule{LowVolume: 300_000}.Build([]Coin{tradable, young, noPerp, noHistory, noVolume}, time.Time{})

	_, sol := scan.Lookup("SOL")
	_, young90 := scan.Lookup("YOUNG")

	if !sol || !young90 || len(scan.Coins) != 2 {
		t.Errorf("coins %+v", scan.Coins)
	}
}

func TestTheListAddsAndRemovesByHand(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

	list, added := List{}.Add(" move ", "тег і малий обіг", now)
	if !added || !list.Has("MOVE") || list.Picks[0].Note != "тег і малий обіг" {
		t.Fatalf("add: %+v", list)
	}

	if _, again := list.Add("MOVE", "", now); again {
		t.Error("added twice")
	}

	list, removed := list.Remove("move")
	if !removed || list.Has("MOVE") {
		t.Errorf("remove: %+v", list)
	}

	if _, gone := list.Remove("MOVE"); gone {
		t.Error("removed what was not there")
	}
}
