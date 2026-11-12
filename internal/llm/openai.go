package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"newsbot/internal/agents"
)

const (
	steady         = 0
	callMarkup     = "<｜DSML｜"
	requestTimeout = 5 * time.Minute
	replyLimit     = 4 << 20
)

type Client struct {
	http  *http.Client
	base  string
	key   string
	model string
}

func New(base, key, model string) *Client {
	return &Client{http: &http.Client{Timeout: requestTimeout}, base: base, key: key, model: model}
}

type wireFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type wireCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function wireFunction `json:"function"`
}

type wireMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []wireCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type wireTool struct {
	Type     string         `json:"type"`
	Function map[string]any `json:"function"`
}

func (c *Client) Complete(
	ctx context.Context,
	session string,
	messages []agents.ChatMessage,
	tools []agents.ToolSpec,
) (agents.ChatMessage, error) {
	request := map[string]any{"model": c.model, "messages": toWire(messages), "temperature": steady}

	if len(tools) > 0 {
		request["tools"] = toolsOf(tools)
	}

	body, err := json.Marshal(request)
	if err != nil {
		return agents.ChatMessage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return agents.ChatMessage{}, err
	}

	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-opencode-session", session)

	resp, err := c.http.Do(req)
	if err != nil {
		return agents.ChatMessage{}, err
	}

	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, replyLimit))
	if err != nil {
		return agents.ChatMessage{}, err
	}

	return parse(resp.StatusCode, raw)
}

func parse(status int, raw []byte) (agents.ChatMessage, error) {
	var reply struct {
		Choices []struct {
			Message wireMessage `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Usage struct {
			Input  int `json:"prompt_tokens"`
			Output int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(raw, &reply); err != nil {
		return agents.ChatMessage{}, fmt.Errorf("model HTTP %d: %.200s", status, raw)
	}

	if reply.Error != nil {
		return agents.ChatMessage{}, fmt.Errorf("model HTTP %d: %s", status, reply.Error.Message)
	}

	if len(reply.Choices) == 0 {
		return agents.ChatMessage{}, fmt.Errorf("model HTTP %d: no choices", status)
	}

	message := fromWire(reply.Choices[0].Message)
	message.Tokens = agents.Tokens{Input: reply.Usage.Input, Output: reply.Usage.Output}

	return message, nil
}

func toWire(messages []agents.ChatMessage) []wireMessage {
	out := make([]wireMessage, 0, len(messages))

	for _, m := range messages {
		w := wireMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}

		for _, call := range m.ToolCalls {
			w.ToolCalls = append(w.ToolCalls, wireCall{
				ID:       call.ID,
				Type:     "function",
				Function: wireFunction{Name: call.Name, Arguments: call.Arguments},
			})
		}

		out = append(out, w)
	}

	return out
}

func fromWire(w wireMessage) agents.ChatMessage {
	m := agents.ChatMessage{Role: "assistant", Content: spoken(w.Content)}

	for _, call := range w.ToolCalls {
		m.ToolCalls = append(m.ToolCalls, agents.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		})
	}

	return m
}

func spoken(content string) string {
	text, _, _ := strings.Cut(content, callMarkup)

	return strings.TrimSpace(text)
}

func toolsOf(specs []agents.ToolSpec) []wireTool {
	out := make([]wireTool, 0, len(specs))

	for _, s := range specs {
		out = append(out, wireTool{
			Type: "function",
			Function: map[string]any{
				"name":        s.Name,
				"description": s.Description,
				"parameters":  s.Parameters,
			},
		})
	}

	return out
}
