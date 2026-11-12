package research

import (
	"testing"
	"time"
)

func TestADatedFindingStartsAtItsHourWhenOneIsKnown(t *testing.T) {
	kyiv, _ := time.LoadLocation("Europe/Kyiv")
	cases := []struct {
		finding Finding
		at      string
		timed   bool
	}{
		{Finding{Day: "2026-11-13", Hour: "12:00", Text: "Старт торгів"}, "2026-11-13T12:00:00Z", true},
		{Finding{Day: "2026-11-13", Text: "13 листопада о 20:43 UTC розблокування"}, "2026-11-13T20:43:00Z", true},
		{Finding{Day: "2026-11-13", Text: "Спот з 04:00 за східним часом США"}, "2026-11-12T22:00:00Z", false},
	}

	for _, c := range cases {
		at, timed := c.finding.Starts(kyiv)

		if at.UTC().Format(time.RFC3339) != c.at || timed != c.timed {
			t.Errorf("%s: starts %s, timed %v", c.finding.Text, at.UTC().Format(time.RFC3339), timed)
		}
	}
}

func TestTheDatesAheadOfTheKeptNewsAreCalendarEntries(t *testing.T) {
	kyiv, _ := time.LoadLocation("Europe/Kyiv")
	archive := Archive{Items: []Finding{
		{ID: "vote", Asset: "ZEC", Day: "2026-11-16", Text: "Голосування", Source: "https://forum", Outlet: "Forum"},
		{ID: "nu8", Asset: "ZEC", Day: "2026-11-16", Text: "NU8", Approximate: true},
		{ID: "old", Asset: "ZEC", Published: "2026-10-01", Text: "Минуле"},
	}}

	dates := archive.Dates(kyiv)

	if len(dates) != 2 || dates[0].ID != "vote:2026-11-16" || !dates[0].AllDay || dates[0].Outlet != "Forum" || dates[0].Source != "ZEC" {
		t.Fatalf("dates %+v", dates)
	}

	if dates[0].Time.Format(time.RFC3339) != "2026-11-15T22:00:00Z" || !dates[1].Rough {
		t.Errorf("an all-day date starts at Kyiv's midnight and an approximate one says so: %+v", dates)
	}
}
