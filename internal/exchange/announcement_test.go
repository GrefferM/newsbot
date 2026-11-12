package exchange

import (
	"testing"
	"time"
)

var (
	announced = time.Date(2026, 11, 9, 5, 0, 0, 0, time.UTC)
	policy    = AnnouncementPolicy{MaxAge: 10 * time.Minute}
)

const delistTitle = "Binance Will Delist ICX, SCRT, STORJ on 2026-09-03"

func TestAnnouncementPolicyPassesFreshTrigger(t *testing.T) {
	published := announced.Add(-2 * time.Minute)
	e := policy.Event(Announcement{Title: delistTitle, URL: "u", Published: published}, announced)

	if e.Muted != "" ||
		e.Kind != "Повний делістинг Binance" ||
		!e.Time.Equal(published) ||
		e.URL != "u" {
		t.Fatalf("got %+v", e)
	}
}

func TestAnnouncementPolicyMutesStaleAndIrrelevant(t *testing.T) {
	stale := policy.Event(Announcement{Title: delistTitle, Published: announced.Add(-6 * time.Hour)}, announced)
	maintenance := Announcement{Title: "Wallet Maintenance for Ethereum Network (ETH)", Published: announced}
	other := policy.Event(maintenance, announced)

	if stale.Muted != MutedStale || other.Muted != MutedIrrelevant {
		t.Fatalf("got %q and %q", stale.Muted, other.Muted)
	}
}

func TestAnnouncementIDIgnoresSpacingAndCase(t *testing.T) {
	a := Announcement{Title: "Binance Will  Delist ICX"}
	b := Announcement{Title: "binance will delist icx "}

	if a.ID() != b.ID() {
		t.Fatalf("%q != %q", a.ID(), b.ID())
	}
}
