package tools

import (
	"context"
	"strings"
	"testing"

	"newsbot/internal/agents"
	"newsbot/internal/research"
)

func TestAScenarioIsCheckedBeforeItCounts(t *testing.T) {
	turn := &agents.Turn{}

	if _, err := use(t, ScenarioTool{}, `{"condition":"?","action":"?","risk":"невідомий","why":"?"}`, turn); err == nil {
		t.Error("an unknown risk passed")
	}

	if _, err := use(t, ScenarioTool{}, `{"condition":"Делістинг","action":"шорт","risk":"мінімальний","why":"рух відомий"}`, turn); err != nil || len(turn.Scenarios) != 1 {
		t.Errorf("scenario: %v, %+v", err, turn.Scenarios)
	}
}

func TestAPlanWithNumbersMustAgreeAndAVerdictNeedsAReason(t *testing.T) {
	turn := &agents.Turn{}
	plan := `{"condition":"закриття вище 1120","action":"лонг","risk":"середній","why":"розворот",`

	if _, err := use(t, ScenarioTool{}, plan+`"direction":"лонг","wake_price_above":1120,"size_pct":7,"stop_price":1130,"take_price":1250}`, turn); err == nil {
		t.Error("a long with its stop above the entry passed")
	}

	if out, err := use(t, ScenarioTool{}, plan+`"direction":"лонг","wake_price_above":1120,"size_pct":7,"stop_price":1055,"take_price":1250}`, turn); err != nil || !strings.Contains(out, "можна поставити") {
		t.Errorf("a sound plan: %q, %v", out, err)
	}

	if _, err := use(t, ScenarioTool{}, `{"condition":"немає підстав","action":"не входити","risk":"мінімальний","why":"рух ринковий","direction":"не входити"}`, turn); err != nil || len(turn.Scenarios) != 2 {
		t.Errorf("a plan to stay out: %v, %+v", err, turn.Scenarios)
	}

	if _, err := use(t, VerdictTool{}, `{"decision":"можливо","why":"?"}`, turn); err == nil {
		t.Error("an unknown verdict passed")
	}

	if _, err := use(t, VerdictTool{}, `{"decision":"підтверджено","why":"година закрилась вище 1120"}`, turn); err != nil || turn.Verdict.Decision != "підтверджено" {
		t.Errorf("verdict %+v, %v", turn.Verdict, err)
	}
}

type footing struct {
	price, day float64
	harm       string
	idle       bool
}

func (f footing) Footing(context.Context, string) (Ground, bool) {
	return Ground{Price: f.price, DayMove: f.day, Smallest: 0.05, Harm: f.harm, Caused: !f.idle}, true
}

func TestAPlanWithNumbersIsAnsweredWithItsTicket(t *testing.T) {
	tool := ScenarioTool{Market: footing{price: 1, day: 0.04}}
	turn := &agents.Turn{Topic: "ABC"}
	plan := `{"condition":"нижче порога","action":"шорт","why":"продажі","direction":"шорт","size_pct":6,`

	near, err := use(t, tool, plan+`"risk":"високий","wake_price_below":0.99,"stop_price":1.02,"take_price":0.95}`, turn)
	if err != nil || !strings.Contains(near, `"threshold_in_day_moves":0.25`) || strings.Contains(near, "what_is_wrong") {
		t.Errorf("a sound plan: %q, %v", near, err)
	}

	far, err := use(t, tool, plan+`"risk":"мінімальний","wake_price_below":0.94,"stop_price":0.95,"take_price":0.935}`, turn)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"ціна до нього звичайно не доходить", "виб'є шум", "тейк ближчий за стоп", "не далекий поріг"} {
		if !strings.Contains(far, want) {
			t.Errorf("no %q in %s", want, far)
		}
	}

	tight, err := use(t, tool, `{"condition":"нижче порога","action":"шорт","why":"продажі","direction":"шорт","size_pct":5.1,"risk":"середній","wake_price_below":0.99,"stop_price":1.02,"take_price":0.95}`, turn)
	if err != nil || !strings.Contains(tight, "найменшого ордера") {
		t.Errorf("a size at the smallest order: %q, %v", tight, err)
	}

	fixed, err := use(t, tool, plan+`"risk":"мінімальний","wake_price_below":0.985,"stop_price":1.01,"take_price":0.95}`, turn)
	if err != nil || strings.Contains(fixed, "what_is_wrong") || len(turn.Scenarios) != 3 {
		t.Errorf("the plan sent again replaces the first: %q, %v, %d plans", fixed, err, len(turn.Scenarios))
	}
}

func TestAPlanWithoutCauseOrAgainstAHarmfulEventIsToldSo(t *testing.T) {
	plan := `{"condition":"вище порога","action":"лонг","why":"відскок","direction":"лонг","size_pct":6,"risk":"мінімальний","wake_price_above":1.01,"stop_price":0.97,"take_price":1.06}`

	against, err := use(t, ScenarioTool{Market: footing{price: 1, day: 0.04, harm: "EXPL Bridge: злам на $40M"}}, plan, &agents.Turn{Topic: "EXPL"})
	if err != nil || !strings.Contains(against, "закриває лонг") {
		t.Errorf("a long against a hack: %q, %v", against, err)
	}

	idle, err := use(t, ScenarioTool{Market: footing{price: 1, day: 0.04, idle: true}}, plan, &agents.Turn{Topic: "WOKE"})
	if err != nil || !strings.Contains(idle, "немає причини") {
		t.Errorf("a plan without a cause: %q, %v", idle, err)
	}
}

func TestAPlanWithoutNumbersWithdrawsTheNumberedOneOfItsRisk(t *testing.T) {
	turn := &agents.Turn{}
	plan := `{"condition":"вище порога","action":"лонг","why":"відскок","direction":"лонг","size_pct":6,"risk":"мінімальний","wake_price_above":1.01,"stop_price":0.97,"take_price":1.06}`

	if _, err := use(t, ScenarioTool{}, plan, turn); err != nil {
		t.Fatal(err)
	}

	if _, err := use(t, ScenarioTool{}, `{"condition":"причини немає","action":"не входити","risk":"мінімальний","why":"позначка без події","direction":"не входити"}`, turn); err != nil {
		t.Fatal(err)
	}

	if len(turn.Scenarios) != 1 || turn.Scenarios[0].Direction != research.DirectionNone {
		t.Errorf("the numbered plan stayed: %+v", turn.Scenarios)
	}
}
