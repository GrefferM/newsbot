package agents

import (
	"context"
	"encoding/json"
)

type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ChatMessage struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
	Tokens     Tokens
}

type Tokens struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

func (t Tokens) plus(other Tokens) Tokens {
	return Tokens{Input: t.Input + other.Input, Output: t.Output + other.Output}
}

type ChatModel interface {
	Complete(ctx context.Context, session string, messages []ChatMessage, tools []ToolSpec) (ChatMessage, error)
}

type Tool interface {
	Spec() ToolSpec
	Call(ctx context.Context, arguments json.RawMessage, turn *Turn) (string, error)
}
