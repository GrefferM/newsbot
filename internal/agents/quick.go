package agents

import (
	"context"
	"errors"
	"time"
)

var ErrNoQuick = errors.New("швидкі ідеї не налаштовані")

type Idea struct {
	Asset     string  `json:"asset"`
	Direction string  `json:"direction"`
	Why       string  `json:"why"`
	Cancel    string  `json:"cancel"`
	Price     float64 `json:"price"`
	Funding   float64 `json:"funding_8h"`
	Event     string  `json:"event,omitempty"`
	Woke      bool    `json:"woke_up"`
}

func (a *Team) Quick(ctx context.Context, question string) (Answer, error) {
	agent := a.deps.Quick

	if agent.Name == "" {
		return Answer{}, ErrNoQuick
	}

	if a.deps.Model == nil {
		return Answer{}, ErrNoModel
	}

	turn := &Turn{}
	started := time.Now()

	text, err := a.converse(ctx, agent, nil, question, turn)

	a.record(started, agent.Name, "", question, text, turn, err)

	return a.answer(text, turn), err
}
