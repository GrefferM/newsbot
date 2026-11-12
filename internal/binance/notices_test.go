package binance

import (
	"testing"
	"time"

	"newsbot/internal/exchange"
)

func TestAnAnnouncementNamesACoinByTickerOrPair(t *testing.T) {
	cases := []struct {
		title string
		asset string
		want  bool
	}{
		{"Binance Will Delist ANT, MULTI, VAI and XMR on 2024-02-20", "VAI", true},
		{"Binance Futures Will Delist USDⓈ-M BANANAUSDT Perpetual Contract", "BANANA", true},
		{"Notice of Removal of Spot Trading Pairs - 2026-10-30", "ANT", false},
		{"Binance Will Delist BANANAS31 on 2026-01-01", "BANANA", false},
		{"Binance Will Add the Monitoring Tag to blur (BLUR)", "BLUR", true},
	}

	for _, c := range cases {
		if got := written(c.title, c.asset); got != c.want {
			t.Errorf("%q names %s: %v, want %v", c.title, c.asset, got, c.want)
		}
	}
}

func TestAnnouncementsAreNarrowedToTheCoin(t *testing.T) {
	published := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	announcements := []exchange.Announcement{
		{Title: "Binance Will Delist MOVE on 2026-11-17", URL: "u1", Published: published},
		{Title: "Binance Will Delist HOT on 2026-11-17", URL: "u2", Published: published},
	}

	got := pick("delisting", announcements, "MOVE")

	if len(got) != 1 || got[0].URL != "u1" || got[0].Published != "2026-11-03T09:00:00Z" || got[0].Section != "delisting" {
		t.Fatalf("picked %+v", got)
	}

	if all := pick("delisting", announcements, ""); len(all) != 2 {
		t.Errorf("without a coin: %+v", all)
	}
}

func TestACoinNamedOnlyInTheTextOfAnAnnouncementIsFound(t *testing.T) {
	body := `{"node":"root","child":[{"node":"element","tag":"p","child":[{"node":"text","text":"Fellow Binancians,"}]},
{"node":"element","tag":"ul","child":[{"node":"element","tag":"li","child":[{"node":"text","text":"Cross margin: BANANA/USDC, HOT/USDC"}]}]},
{"node":"element","tag":"tag","child":[{"node":"text","text":"node text"}]}]}`

	text := plain(body)

	if found, ok := mention(text, "BANANA"); !ok || found != "Cross margin: BANANA/USDC, HOT/USDC" {
		t.Errorf("found %q, %v in %q", found, ok, text)
	}

	for _, asset := range []string{"TAG", "NODE", "BANANAS31", "USD"} {
		if found, ok := mention(text, asset); ok {
			t.Errorf("%s found in %q", asset, found)
		}
	}
}

func TestATickerThatIsAnEnglishWordIsNotFoundInOrdinaryWords(t *testing.T) {
	if written("Binance Futures Will End Last Price Protected Period on USDⓈ-Margined BULLAUSDT Perpetual Contract", "ON") {
		t.Error("the word «on» was taken for the ticker ON")
	}

	if !written("Binance Will Delist ON, XYZ on 2026-11-25", "ON") || !written("Binance Futures Will Launch ONUSDT Perpetual Contract", "ON") {
		t.Error("the ticker ON was not found in its own announcements")
	}
}
