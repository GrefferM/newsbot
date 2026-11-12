package binance

import (
	"testing"

	"newsbot/internal/market"
)

func TestThePulseListsOnlyCryptoContractsOfTheMostLiquid(t *testing.T) {
	futures := &Futures{byName: map[string]market.Ticker{
		"BTCUSDT":     {Contract: "BTCUSDT", Price: 70000, QuoteVolume: 9e9},
		"CLUSDT":      {Contract: "CLUSDT", Price: 60, QuoteVolume: 8e9},
		"INJUSDT":     {Contract: "INJUSDT", Price: 7, QuoteVolume: 2e8},
		"AGTUSDT":     {Contract: "AGTUSDT", Price: 0.05, QuoteVolume: 1e8},
		"BTCUSDC":     {Contract: "BTCUSDC", Price: 70000, QuoteVolume: 7e9},
		"DEADUSDT":    {Contract: "DEADUSDT", QuoteVolume: 5e9},
		"CRWVUSDT":    {Contract: "CRWVUSDT", Price: 90, QuoteVolume: 3e9},
		"ARIAUSDT":    {Contract: "ARIAUSDT", Price: 0.27, QuoteVolume: 1e7},
		"UNKNOWNUSDT": {Contract: "UNKNOWNUSDT", Price: 1, QuoteVolume: 6e9},
	}}
	crypto := map[string]bool{"BTCUSDT": true, "INJUSDT": true, "AGTUSDT": true, "ARIAUSDT": true, "CLUSDT": false, "CRWVUSDT": false}

	var got []string

	for _, ticker := range futures.liquid(3, crypto) {
		got = append(got, ticker.Contract)
	}

	if len(got) != 3 || got[0] != "BTCUSDT" || got[1] != "INJUSDT" || got[2] != "AGTUSDT" {
		t.Errorf("the three most liquid crypto contracts, most liquid first: %v", got)
	}
}

func TestTheFollowedContractsJoinThePulseOnce(t *testing.T) {
	futures := &Futures{byName: map[string]market.Ticker{
		"BTCUSDT":  {Contract: "BTCUSDT", Price: 70000, QuoteVolume: 9e9},
		"ARIAUSDT": {Contract: "ARIAUSDT", Price: 0.27, QuoteVolume: 1e7},
		"DEADUSDT": {Contract: "DEADUSDT", QuoteVolume: 5e9},
	}}
	picked := futures.liquid(1, nil)

	if extra := futures.followed(picked); extra != nil {
		t.Fatalf("nothing followed yet: %v", extra)
	}

	futures.Follow(func() []string {
		return []string{"BTCUSDT", "ARIAUSDT", "ARIAUSDT", "DEADUSDT", "GONEUSDT"}
	})

	if extra := futures.followed(picked); len(extra) != 1 || extra[0].Contract != "ARIAUSDT" {
		t.Errorf("only a followed contract that trades and is not yet picked joins, once: %v", extra)
	}
}

func TestContractsAreSortedIntoTheirKinds(t *testing.T) {
	kinds := map[string]string{"BTCUSDT": kindCrypto, "XAUUSDT": kindCommodity, "NVDAUSDT": kindEquity, "ETHUSDT": kindCrypto}

	if got := of(kinds, kindCrypto); len(got) != 2 || !got["BTCUSDT"] || got["XAUUSDT"] {
		t.Errorf("crypto %v", got)
	}

	if got := of(kinds, kindEquity); len(got) != 1 || !got["NVDAUSDT"] {
		t.Errorf("equities %v", got)
	}
}

func TestTheContractsBeyondTheNearOnesKeepTheirOrder(t *testing.T) {
	every := []market.Ticker{{Contract: "BTCUSDT"}, {Contract: "ZECUSDT"}, {Contract: "AKEUSDT"}, {Contract: "BARDUSDT"}}
	rest := apart(every, []market.Ticker{{Contract: "BTCUSDT"}, {Contract: "AKEUSDT"}})

	if len(rest) != 2 || rest[0].Contract != "ZECUSDT" || rest[1].Contract != "BARDUSDT" {
		t.Errorf("the rest: %+v", rest)
	}
}

func TestAContractsScaleIsStrippedFromItsAsset(t *testing.T) {
	if assetOf("1000000BOBUSDT") != "BOB" || assetOf("1000PEPEUSDT") != "PEPE" || assetOf("ZECUSDT") != "ZEC" {
		t.Error("contract scale not stripped")
	}
}
