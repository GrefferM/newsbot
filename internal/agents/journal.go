package agents

import (
	"fmt"
	"time"

	"newsbot/internal/research"
)

type Step struct {
	Tool      string `json:"tool"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
}

type Record struct {
	At        time.Time           `json:"at"`
	Agent     string              `json:"agent"`
	Topic     string              `json:"topic,omitempty"`
	Cleared   bool                `json:"cleared,omitempty"`
	Question  string              `json:"question"`
	Seconds   float64             `json:"seconds"`
	Tokens    Tokens              `json:"tokens"`
	Steps     []Step              `json:"steps"`
	Findings  []research.Finding  `json:"findings,omitempty"`
	Scenarios []research.Scenario `json:"scenarios,omitempty"`
	Ideas     []Idea              `json:"ideas,omitempty"`
	Answer    string              `json:"answer"`
	Error     string              `json:"error,omitempty"`
}

type Log interface {
	Keep(run Record)
	Last(agent, topic string, count int) ([]Record, error)
}

func (a *Team) History(name, topic string, count int) ([]Record, error) {
	if _, ok := a.byName[name]; !ok {
		return nil, ErrUnknownAgent
	}

	if a.deps.Log == nil {
		return []Record{}, nil
	}

	return a.deps.Log.Last(name, topic, count)
}

func (a *Team) Forget(name, topic string) error {
	if _, ok := a.byName[name]; !ok {
		return ErrUnknownAgent
	}

	if a.deps.Log != nil {
		a.deps.Log.Keep(Record{At: time.Now().UTC(), Agent: name, Topic: topic, Cleared: true})
	}

	return nil
}

func (a *Team) remembered(agent Agent, topic string) []ChatMessage {
	if agent.Memory == 0 || a.deps.Log == nil {
		return nil
	}

	past, err := a.deps.Log.Last(agent.Name, topic, agent.Memory)
	if err != nil {
		return nil
	}

	var out []ChatMessage

	for _, r := range past {
		if r.Error == "" && r.Answer != "" {
			out = append(out,
				ChatMessage{Role: "user", Content: about(topic, r.Question)},
				ChatMessage{Role: "assistant", Content: r.Answer},
			)
		}
	}

	return out
}

func about(topic, question string) string {
	if topic == "" {
		return question
	}

	return fmt.Sprintf("Інструмент %s. %s", topic, question)
}

func (a *Team) record(started time.Time, name, topic, question, text string, turn *Turn, err error) {
	if a.deps.Log == nil {
		return
	}

	run := Record{
		At:        started.UTC(),
		Agent:     name,
		Topic:     topic,
		Question:  question,
		Seconds:   time.Since(started).Seconds(),
		Tokens:    turn.Tokens,
		Steps:     turn.Steps,
		Findings:  turn.Findings,
		Scenarios: turn.Scenarios,
		Ideas:     turn.Ideas,
		Answer:    text,
	}

	if err != nil {
		run.Error = err.Error()
	}

	a.deps.Log.Keep(run)
}
