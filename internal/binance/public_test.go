package binance

import (
	"testing"
)

func TestOnlyPublicReadPathsAreAllowed(t *testing.T) {
	cases := map[string]bool{
		"/api/v3/ticker/24hr":            true,
		"/fapi/v1/premiumIndex":          true,
		"/futures/data/openInterestHist": true,
		"/sapi/v1/asset/assetDetail":     false,
		"/fapi/v1/../../sapi/v1/account": false,
		"/fapi/v1/order":                 true,
	}

	for path, want := range cases {
		if got := allowed(path); got != want {
			t.Errorf("%s: %v, want %v", path, got, want)
		}
	}
}
