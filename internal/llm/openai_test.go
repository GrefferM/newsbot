package llm

import (
	"testing"

	"newsbot/internal/agents"
)

func TestAReplyWithToolCallsIsRead(t *testing.T) {
	raw := `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[
		{"id":"c1","type":"function","function":{"name":"binance_get","arguments":"{\"path\":\"/api/v3/time\"}"}}]}}],
		"usage":{"prompt_tokens":1200,"completion_tokens":85}}`

	m, err := parse(200, []byte(raw))
	if err != nil || len(m.ToolCalls) != 1 || m.ToolCalls[0].Name != "binance_get" || m.ToolCalls[0].ID != "c1" {
		t.Fatalf("message %+v err %v", m, err)
	}

	if m.Tokens.Input != 1200 || m.Tokens.Output != 85 {
		t.Errorf("tokens %+v", m.Tokens)
	}
}

func TestAnErrorReplyBecomesAnError(t *testing.T) {
	if _, err := parse(400, []byte(`{"error":{"message":"bad model"}}`)); err == nil {
		t.Error("no error")
	}
}

func TestToolCallsGoBackOnTheWire(t *testing.T) {
	wire := toWire([]agents.ChatMessage{
		{Role: "assistant", ToolCalls: []agents.ToolCall{{ID: "c1", Name: "x", Arguments: "{}"}}},
		{Role: "tool", ToolCallID: "c1", Content: "ok"},
	})

	if wire[0].ToolCalls[0].Type != "function" || wire[1].ToolCallID != "c1" {
		t.Errorf("wire %+v", wire)
	}
}

func TestToolCallsWrittenAsTextAreNotAnAnswer(t *testing.T) {
	raw := `{"choices":[{"message":{"role":"assistant","content":"Готово.\n\n<｜DSML｜ calls>\n<｜DSML｜ invoke name=\"http_get\">x</｜DSML｜ invoke>"}}]}`

	reply, err := parse(200, []byte(raw))
	if err != nil || reply.Content != "Готово." {
		t.Errorf("content %q, %v", reply.Content, err)
	}
}
