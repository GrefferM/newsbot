package main

import (
	"context"
	"fmt"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
)

type planJudge struct {
	team *agents.Team
}

func (j *planJudge) Judge(ctx context.Context, w autopilot.Watch, price float64, cause autopilot.Cause) (autopilot.Verdict, error) {
	plan := w.Plan
	question := fmt.Sprintf(
		"План на стеженні з %s: умова — %s; дія — %s; напрям — %s; розмір %.1f%% рахунку; стоп %v; тейк %v; "+
			"чому такий ризик — %s. Ціна перпа зараз %v. Тебе питають, бо %s. Чи виконана умова і чи чинні причини?",
		w.Armed.Format("2006-01-02 15:04 UTC"), plan.Condition, plan.Action, plan.Direction,
		plan.SizePct, plan.Stop, plan.Take, plan.Why, price, cause,
	)

	answer, err := j.team.Ask(ctx, guardAgent, w.Asset, question)
	if err != nil {
		return autopilot.Verdict{}, err
	}

	if answer.Verdict.Decision == "" {
		return autopilot.Verdict{Decision: autopilot.Denied, Why: "агент не дав вердикту: " + answer.Text}, nil
	}

	return autopilot.Verdict{Decision: answer.Verdict.Decision, Why: answer.Verdict.Why}, nil
}
