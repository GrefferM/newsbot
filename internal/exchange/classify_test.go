package exchange

import (
	"slices"
	"testing"

	"newsbot/internal/news"
)

type classifyCase struct {
	title string
	cat   news.Category
	syms  []news.Symbol
	ok    bool
}

func check(t *testing.T, tc classifyCase, got Classification, ok bool) {
	t.Helper()

	if ok != tc.ok || got.Category != tc.cat || !slices.Equal(got.Symbols, tc.syms) {
		t.Errorf(
			"%q: got (%q, %v, %v), want (%q, %v, %v)",
			tc.title, got.Category, got.Symbols, ok, tc.cat, tc.syms, tc.ok,
		)
	}
}

func TestClassifyBinance(t *testing.T) {
	cases := []classifyCase{
		{
			title: "Binance Will Delist ICX, SCRT, STORJ on 2026-09-03",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"ICX", "SCRT", "STORJ"},
			ok:    true,
		},
		{
			title: "Binance Will Delist SNM, SRM and YFII on 2023-08-22",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"SNM", "SRM", "YFII"},
			ok:    true,
		},
		{
			title: "Binance Announced the First Batch of Vote to Delist Results and Will Delist BADGER, BAL, BETA on 2025-04-16",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"BADGER", "BAL", "BETA"},
			ok:    true,
		},
		{
			title: "Binance Futures Will Delist PTBUSDT and MITOUSDT USDⓈ-M Perpetual Contracts (2026-10-23)",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"PTB", "MITO"},
			ok:    true,
		},
		{
			title: "Binance Will Extend the Monitoring Tag to Include BICO & CVC on 2026-10-09",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"BICO", "CVC"},
			ok:    true,
		},
		{
			title: "Binance Will Extend the Monitoring Tag to Include WAN, Remove the Monitoring Tag for GPS & MBL, and Remove the Seed Tag for BERA on 2025-10-09",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"WAN"},
			ok:    true,
		},
		{
			title: "Binance Will Add Genius Terminal (GENIUS) and OpenGradient (OPG) on Earn, Buy Crypto, Convert, VIP Loan & Margin",
			cat:   news.ShortTrigger,
			syms:  []news.Symbol{"GENIUS", "OPG"},
			ok:    true,
		},
		{
			title: "Binance Will List Predict (PREDICT) with Seed Tag Applied",
			cat:   "",
			syms:  nil,
			ok:    false,
		},
		{
			title: "Binance Futures Will Launch USDⓈ-Margined PREDICTUSDT Perpetual Contract Pre-Market Trading (2026-10-25)",
			cat:   "",
			syms:  nil,
			ok:    false,
		},
		{
			title: "Binance Futures Will Launch Multiple TradFi USDⓈ-Margined Perpetual Contracts (2026-10-29)",
			cat:   "",
			syms:  nil,
			ok:    false,
		},
		{
			title: "Notice of Removal of Spot Trading Pairs - 2026-10-30",
			cat:   "",
			syms:  nil,
			ok:    false,
		},
		{
			title: "Binance Alpha Will Remove PTB, AA, OWL (2026-10-23)",
			cat:   "",
			syms:  nil,
			ok:    false,
		},
		{
			title: "Binance Margin Will Delist ABC on 2026-01-01",
			cat:   "",
			syms:  nil,
			ok:    false,
		},
	}

	for _, tc := range cases {
		got, ok := ClassifyBinance(tc.title)

		check(t, tc, got, ok)
	}
}

func TestBinanceSeverityFollowsPriceImpact(t *testing.T) {
	cases := map[string]news.Severity{
		"Binance Will Extend the Monitoring Tag to Include BICO & CVC on 2026-10-09":               news.Critical,
		"Binance Will Delist ICX, SCRT, STORJ on 2026-09-03":                                       news.Critical,
		"Binance Futures Will Delist PTBUSDT and MITOUSDT USDⓈ-M Perpetual Contracts (2026-10-23)": news.High,
		"Binance Will Add Hyperliquid (HYPE) on Earn, Buy Crypto, Convert, VIP Loan & Margin":      news.High,
	}

	for title, want := range cases {
		if got, _ := ClassifyBinance(title); got.Severity != want {
			t.Errorf("%q: got %v, want %v", title, got.Severity, want)
		}
	}
}
