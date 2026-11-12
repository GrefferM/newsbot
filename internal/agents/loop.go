package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	agentSteps      = 12
	reportSteps     = 2
	wrapUp          = "Збір даних завершено. Передай результат інструментами, що лишились, і дай відповідь."
	closing         = "Інструментів більше немає. Дай відповідь текстом."
	toolOutputLimit = 24_000
	stepResultLimit = 400
)

func (a *Team) converse(
	ctx context.Context,
	agent Agent,
	past []ChatMessage,
	question string,
	turn *Turn,
) (string, error) {
	session := fmt.Sprintf("newsbot-%s-%d", agent.Name, time.Now().UnixNano())
	messages := []ChatMessage{
		{Role: "system", Content: agent.Prompt + "\n\nЗараз " + time.Now().UTC().Format(time.RFC3339) + "."},
	}

	messages = append(messages, past...)
	messages = append(messages, ChatMessage{Role: "user", Content: question})

	phases := []phase{
		{steps: agent.rounds(), tools: agent.all()},
		{steps: reportSteps, tools: agent.Reports, opening: wrapUp},
		{steps: 1, opening: closing},
	}

	for _, p := range phases {
		if p.opening != "" {
			messages = append(messages, ChatMessage{Role: "user", Content: p.opening})
		}

		reply, err := a.work(ctx, session, &messages, p, turn)
		if err != nil || reply != "" {
			return reply, err
		}
	}

	return "", nil
}

type phase struct {
	steps   int
	tools   []Tool
	opening string
}

func (a *Team) work(
	ctx context.Context,
	session string,
	messages *[]ChatMessage,
	p phase,
	turn *Turn,
) (string, error) {
	for range p.steps {
		reply, done, err := a.step(ctx, session, messages, p.tools, turn)
		if err != nil {
			return "", err
		}

		if done {
			return strings.TrimSpace(reply), nil
		}
	}

	return "", nil
}

func (a *Team) step(
	ctx context.Context,
	session string,
	messages *[]ChatMessage,
	tools []Tool,
	turn *Turn,
) (string, bool, error) {
	reply, err := a.deps.Model.Complete(ctx, session, *messages, specs(tools))
	if err != nil {
		return "", false, fmt.Errorf("model: %w", err)
	}

	turn.Tokens = turn.Tokens.plus(reply.Tokens)
	*messages = append(*messages, reply)

	if len(reply.ToolCalls) == 0 {
		return reply.Content, true, nil
	}

	for _, call := range reply.ToolCalls {
		*messages = append(*messages, ChatMessage{
			Role:       "tool",
			ToolCallID: call.ID,
			Content:    run(ctx, tools, call, turn),
		})
	}

	return reply.Content, false, nil
}

func (a *Team) answer(text string, turn *Turn) Answer {
	return Answer{
		Text:      text,
		Findings:  turn.Findings,
		Scenarios: turn.Scenarios,
		Calls:     turn.Calls,
		Ideas:     turn.Ideas,
		Verdict:   turn.Verdict,
	}
}

func run(ctx context.Context, tools []Tool, call ToolCall, turn *Turn) string {
	turn.Calls = append(turn.Calls, call.Name)

	out := use(ctx, tools, call, turn)

	turn.Steps = append(turn.Steps, Step{Tool: call.Name, Arguments: call.Arguments, Result: head(out)})

	return out
}

func head(out string) string {
	if len(out) <= stepResultLimit {
		return out
	}

	return strings.ToValidUTF8(out[:stepResultLimit], "") + "…"
}

func use(ctx context.Context, tools []Tool, call ToolCall, turn *Turn) string {

	for _, tool := range tools {
		if tool.Spec().Name != call.Name {
			continue
		}

		out, err := tool.Call(ctx, json.RawMessage(call.Arguments), turn)
		if err != nil {
			return "помилка: " + err.Error()
		}

		if len(out) > toolOutputLimit {
			out = out[:toolOutputLimit] + "\n…обрізано"
		}

		return out
	}

	return "помилка: інструмент " + call.Name + " зараз недоступний; збір даних завершено — передай результат"
}

func specs(tools []Tool) []ToolSpec {
	out := make([]ToolSpec, 0, len(tools))

	for _, t := range tools {
		out = append(out, t.Spec())
	}

	return out
}
