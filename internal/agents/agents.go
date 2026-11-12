package agents

import (
	"context"
	"errors"
	"slices"
	"time"

	"newsbot/internal/research"
)

var (
	ErrUnknownAgent = errors.New("такого агента немає")
	ErrNoModel      = errors.New("модель не налаштована: MODEL_KEY")
)

type Agent struct {
	Name    string
	Prompt  string
	Tools   []Tool
	Reports []Tool
	Memory  int
	Rounds  int
}

func (a Agent) rounds() int {
	if a.Rounds > 0 {
		return a.Rounds
	}

	return agentSteps
}

func (a Agent) all() []Tool {
	return append(slices.Clone(a.Tools), a.Reports...)
}

type Turn struct {
	Findings  []research.Finding
	Scenarios []research.Scenario
	Calls     []string
	Steps     []Step
	Picks     []Pick
	Ideas     []Idea
	Verdict   Verdict
	Tokens    Tokens
}

type Verdict struct {
	Decision string
	Why      string
}

type Answer struct {
	Text      string
	Findings  []research.Finding
	Scenarios []research.Scenario
	Calls     []string
	Ideas     []Idea
	Verdict   Verdict
}

type Deps struct {
	Model  ChatModel
	Agents []Agent
	Log    Log
	Funnel Funnel
	Quick  Agent
}

type Team struct {
	deps   Deps
	byName map[string]Agent
}

func NewTeam(deps Deps) *Team {
	byName := make(map[string]Agent, len(deps.Agents))

	for _, a := range deps.Agents {
		byName[a.Name] = a
	}

	return &Team{deps: deps, byName: byName}
}

func (a *Team) Ask(ctx context.Context, name, topic, question string) (Answer, error) {
	agent, ok := a.byName[name]
	if !ok {
		return Answer{}, ErrUnknownAgent
	}

	if a.deps.Model == nil {
		return Answer{}, ErrNoModel
	}

	turn := &Turn{}
	started := time.Now()

	text, err := a.converse(ctx, agent, a.remembered(agent, topic), about(topic, question), turn)

	a.record(started, name, topic, question, text, turn, err)

	return a.answer(text, turn), err
}
