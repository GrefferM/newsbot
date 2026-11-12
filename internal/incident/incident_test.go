package incident

import (
	"strconv"
	"testing"
	"time"

	"newsbot/internal/news"
)

var (
	start  = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	origin = news.Origin{Name: "DefiLlama", URL: "https://defillama.com/stablecoins"}
)

func TestHackPolicy(t *testing.T) {
	policy := HackPolicy{MinAmount: 1e6, MaxAge: 72 * time.Hour}
	hacks := []Hack{
		{Name: "Big", Date: start.Add(-time.Hour), Amount: 40e6, AmountKnown: true, Chains: []string{"Ethereum"}},
		{Name: "Small", Date: start.Add(-time.Hour), Amount: 0.5e6, AmountKnown: true},
		{Name: "Unknown", Date: start.Add(-time.Hour)},
		{Name: "Old", Date: start.Add(-100 * time.Hour), Amount: 40e6, AmountKnown: true},
	}

	events := policy.Events(origin, hacks, start)

	if len(events) != 3 || events[0].Title != "Big" || events[0].Muted != "" {
		t.Fatalf("got %+v", events)
	}

	if events[1].Muted == "" || events[2].Muted == "" {
		t.Fatalf("small and unknown hacks must be muted: %+v", events[1:])
	}

	if events[0].ID != "hack:Big:"+strconv.FormatInt(start.Add(-time.Hour).Unix(), 10) {
		t.Errorf("unexpected id %q", events[0].ID)
	}
}

var depegRules = DepegRules{MinSupply: 300e6, Threshold: 0.01, Dead: 0.5, Cooldown: 6 * time.Hour}

func TestDepegDetectorAlertsOnceWhenPegBreaks(t *testing.T) {
	d := NewDepegDetector(depegRules, origin)
	coins := []Stablecoin{
		{Name: "Tether", Symbol: "USDT", Price: 0.9987, Supply: 176e9},
		{Name: "Ethena", Symbol: "USDe", Price: 0.97, Supply: 4.9e9},
		{Name: "Tiny", Symbol: "TINY", Price: 0.5, Supply: 1e6},
		{Name: "Neutrino", Symbol: "USDN", Price: 0.012, Supply: 600e6},
	}

	first := d.Detect(coins, start)

	if len(first) != 1 || first[0].Symbols[0] != "USDe" {
		t.Fatalf("got %+v", first)
	}

	if later := d.Detect(coins, start.Add(7*time.Hour)); len(later) != 0 {
		t.Fatalf("still off peg reported again: %+v", later)
	}
}

func TestDepegDetectorAlertsAgainAfterRecoveryAndCooldown(t *testing.T) {
	d := NewDepegDetector(depegRules, origin)
	off := []Stablecoin{{Name: "Ethena", Symbol: "USDe", Price: 0.97, Supply: 4.9e9}}
	back := []Stablecoin{{Name: "Ethena", Symbol: "USDe", Price: 0.999, Supply: 4.9e9}}

	d.Detect(off, start)
	d.Detect(back, start.Add(time.Hour))

	if flap := d.Detect(off, start.Add(2*time.Hour)); len(flap) != 0 {
		t.Fatalf("cooldown ignored: %+v", flap)
	}

	d.Detect(back, start.Add(3*time.Hour))

	if again := d.Detect(off, start.Add(7*time.Hour)); len(again) != 1 {
		t.Fatalf("new depeg missed: %+v", again)
	}
}
