package news

import (
	"slices"
	"testing"
)

func TestSymbolsFromList(t *testing.T) {
	cases := map[string][]Symbol{
		"ICX, SCRT and STORJ":       {"ICX", "SCRT", "STORJ"},
		"BICO & CVC":                {"BICO", "CVC"},
		"Vanar (VANRY), not-a-coin": {"VANRY"},
	}

	for in, want := range cases {
		if got := SymbolsFromList(in); !slices.Equal(got, want) {
			t.Errorf("%q: got %v, want %v", in, got, want)
		}
	}
}

func TestSymbolsFromUSDTPairsDropsScalePrefix(t *testing.T) {
	got := SymbolsFromUSDTPairs("USDⓈ-M PROMPTUSDT, 1000PEPEUSDT and 1000000BOBUSDT")
	want := []Symbol{"PROMPT", "PEPE", "BOB"}

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFormat(t *testing.T) {
	cases := map[string]string{
		FormatUSD(2_500_000_000): "$2.50B",
		FormatUSD(959_000):       "$959K",
		FormatUSD(60_400_000):    "$60.4M",
		FormatPrice(75743):       "$75743",
		FormatPrice(1.2345):      "$1.234",
		FormatPrice(0.02565):     "$0.02565",
		FormatPercent(-0.57):     "-0.57%",
	}

	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
