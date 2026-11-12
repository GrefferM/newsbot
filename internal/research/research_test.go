package research

import (
	"testing"
	"time"
)

func TestNewsIsKeptPerCoinNewestFirstAndARetellingAddsItsSource(t *testing.T) {
	old := Finding{ID: "a", Asset: "BLUR", Day: "2026-06-18", Text: "Monitoring Tag", Source: "binance"}
	fresh := Finding{ID: "b", Asset: "BLUR", Day: "2026-10-12", Text: "Сайт закривається", Source: "blog"}
	other := Finding{ID: "c", Asset: "MOVE", Day: "2026-07-21", Text: "Банкрутство", Source: "court"}

	archive := Archive{}

	for _, f := range []Finding{old, fresh, other} {
		var kept bool

		if archive, kept = archive.Keep(f); !kept {
			t.Fatalf("not kept: %+v", f)
		}
	}

	again := Finding{ID: "d", Asset: "BLUR", Day: "2026-10-12", Text: "Blur закриває сайт", Source: "x", Others: []string{"blog"}}

	archive, _ = archive.Keep(again)

	if _, kept := archive.Keep(Finding{ID: "e", Asset: "BLUR"}); kept {
		t.Error("an empty finding kept")
	}

	got := archive.Of("BLUR")
	if len(got) != 2 || got[0].ID != "b" || len(got[0].Sources()) != 2 {
		t.Errorf("BLUR news %+v", got)
	}

	archive, removed := archive.Remove("b")
	if !removed || len(archive.Of("BLUR")) != 1 {
		t.Errorf("after remove %+v", archive)
	}

	if _, removed := archive.Remove("b"); removed {
		t.Error("removed what was not there")
	}
}

func TestACoinKeepsItsPlanAndTheLastAnswerApart(t *testing.T) {
	plan := Plan{
		Summary:   "слабка монета",
		Scenarios: []Scenario{{Condition: "делістинг", Action: "шорт", Risk: RiskMinimal, Why: "вихід є"}},
		Made:      time.Date(2026, 11, 10, 10, 0, 0, 0, time.UTC),
	}

	archive, kept := Archive{}.Keep(Finding{ID: "a", Asset: "BLUR", Day: "2026-06-18", Text: "Monitoring Tag"})
	if !kept {
		t.Fatal("finding not kept")
	}

	if _, fixed := archive.Fix("BLUR", Plan{}); fixed {
		t.Error("an empty plan kept")
	}

	archive, _ = archive.Fix("BLUR", plan)

	base, ok := archive.Plan("BLUR")

	if !ok || base.Summary != plan.Summary || len(archive.Of("BLUR")) != 1 {
		t.Fatalf("plan %+v, news %v", base, archive.Of("BLUR"))
	}

	if _, ok := archive.Plan("MOVE"); ok {
		t.Error("another coin got the plan")
	}
}

func TestTheSameFactReportedTwiceBecomesOneWithAllItsSources(t *testing.T) {
	first := Finding{ID: "a", Asset: "BANANA", Day: "2026-11-09", Text: "Запуск", Source: "blog", Others: []string{"x"}}
	again := Finding{ID: "b", Asset: "BANANA", Day: "2026-11-09", Text: "Мейннет запущено", Source: "telegram", Others: []string{"blog"}}
	later := Finding{ID: "c", Asset: "BANANA", Day: "2026-12-07", Text: "Оновлення", Source: "blog"}

	found, added := Report(nil, first)
	if !added {
		t.Fatal("first report refused")
	}

	found, added = Report(found, again)
	if added || len(found) != 1 || len(found[0].Sources()) != 3 || found[0].Text != "Запуск" {
		t.Fatalf("after a repeat: %+v", found)
	}

	if found, added = Report(found, later); !added || len(found) != 2 {
		t.Errorf("another day from the same source refused: %+v", found)
	}
}

func TestAPlanIsWatchedOnlyWhenItsNumbersAgree(t *testing.T) {
	long := Scenario{Direction: DirectionLong, WakeAbove: 1120, SizePct: 7, Stop: 1055, Take: 1250}
	short := Scenario{Direction: DirectionShort, WakeAbove: 1085, SizePct: 10, Stop: 1130, Take: 1010}

	dated := Scenario{Direction: DirectionLong, WakeAbove: 1120, WakeAfter: "2026-11-16", SizePct: 5, Stop: 1010, Take: 1180}

	for _, s := range []Scenario{long, short, dated} {
		if err := s.Armable(); err != nil {
			t.Errorf("%+v: %v", s, err)
		}
	}

	wrong := []Scenario{
		{Direction: DirectionNone, WakeAbove: 1, SizePct: 1, Stop: 1, Take: 1},
		{Direction: DirectionLong, SizePct: 7, Stop: 1055, Take: 1250},
		{Direction: DirectionLong, WakeAfter: "2026-11-16", SizePct: 5, Stop: 1010, Take: 1180},
		{Direction: DirectionLong, WakeAbove: 1120, WakeBelow: 1000, SizePct: 7, Stop: 1055, Take: 1250},
		{Direction: DirectionLong, WakeAbove: 1120, SizePct: 7, Stop: 1130, Take: 1250},
		{Direction: DirectionShort, WakeBelow: 990, SizePct: 10, Stop: 980, Take: 900},
		{Direction: DirectionShort, WakeBelow: 990, SizePct: 0, Stop: 1030, Take: 900},
	}

	for _, s := range wrong {
		if s.Armable() == nil {
			t.Errorf("armable: %+v", s)
		}
	}

	if (Scenario{Condition: "?"}).Measured() || !long.Measured() {
		t.Error("measured is told wrong")
	}
}
