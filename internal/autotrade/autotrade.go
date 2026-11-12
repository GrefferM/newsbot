package autotrade

import (
	"slices"
	"time"

	"newsbot/internal/research"
	"newsbot/internal/schedule"
)

var kinds = []string{schedule.Unlock, schedule.Coin}

type State struct {
	Enabled bool                 `json:"enabled"`
	Planned map[string]time.Time `json:"planned,omitempty"`
}

func Planned(e schedule.Entry) bool {
	return slices.Contains(kinds, e.Kind) && !e.AllDay && !e.Rough && len(e.Assets) > 0
}

func Safest(plan research.Plan) (research.Scenario, bool) {
	for _, risk := range []string{research.RiskMinimal, research.RiskMedium} {
		for _, scenario := range plan.Scenarios {
			if scenario.Risk == risk && scenario.Armable() == nil {
				return scenario, true
			}
		}
	}

	return research.Scenario{}, false
}

func Timed(scenario research.Scenario, e schedule.Entry) research.Scenario {
	if scenario.WakeAfter == "" {
		scenario.WakeAfter = e.Time.UTC().Format(time.RFC3339)
	}

	return scenario
}
