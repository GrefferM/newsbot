package deribit

import (
	"testing"
	"time"
)

func TestOnlyTheLastFridayOfAMonthIsKept(t *testing.T) {
	at := func(month time.Month, day int) time.Time {
		return time.Date(2026, month, day, 8, 0, 0, 0, time.UTC)
	}

	got := monthly(map[time.Time][]string{
		at(11, 12): {"BTC"},
		at(11, 20): {"BTC", "ETH"},
		at(11, 27): {"ETH", "BTC"},
		at(12, 25): {"BTC"},
	})

	titles := map[string]bool{}

	for _, e := range got {
		titles[e.Title] = true
	}

	if len(got) != 2 || !titles["Місячна експірація опціонів BTC, ETH"] || !titles["Квартальна експірація опціонів BTC"] {
		t.Fatalf("got %+v", got)
	}
}
