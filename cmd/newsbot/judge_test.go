package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/research"
)

type silentModel struct {
	asked string
}

func (m *silentModel) Complete(
	_ context.Context,
	_ string,
	messages []agents.ChatMessage,
	_ []agents.ToolSpec,
) (agents.ChatMessage, error) {
	m.asked = messages[len(messages)-1].Content

	return agents.ChatMessage{Role: "assistant", Content: "Не знаю."}, nil
}

func TestTheJudgeNamesTheCoinAndTheCauseAndWaitsWithoutAVerdict(t *testing.T) {
	model := &silentModel{}
	judge := &planJudge{team: agents.NewTeam(agents.Deps{
		Model:  model,
		Agents: []agents.Agent{{Name: guardAgent, Prompt: "Роль."}},
	})}
	watch := autopilot.Watch{
		Asset: "ZEC",
		Armed: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
		Plan:  research.Scenario{Condition: "голосування завершилось", Direction: research.DirectionLong, SizePct: 3},
	}

	verdict, err := judge.Judge(context.Background(), watch, 1052, autopilot.ByPrice)
	if err != nil {
		t.Fatal(err)
	}

	if verdict.Decision != autopilot.Denied || !strings.Contains(verdict.Why, "Не знаю.") {
		t.Errorf("an answer without a verdict became %+v", verdict)
	}

	for _, want := range []string{"Інструмент ZEC.", "голосування завершилось", "1052", string(autopilot.ByPrice)} {
		if !strings.Contains(model.asked, want) {
			t.Errorf("the question lacks %q: %s", want, model.asked)
		}
	}
}
