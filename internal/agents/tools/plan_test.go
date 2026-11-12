package tools

import (
	"strings"
	"testing"

	"newsbot/internal/agents"
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
