package tools

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
)

func encode(v any) (string, error) {
	data, err := json.Marshal(v)

	return string(data), err
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}

func object(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func text(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func number(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}

func enum(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}
