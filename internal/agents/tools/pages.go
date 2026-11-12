package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"newsbot/internal/agents"
)

type Pages interface {
	Fetch(ctx context.Context, address string) (string, error)
}

type FetchTool struct {
	Pages  Pages
	Budget int
}

func (t FetchTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: fetchName,
		Description: fmt.Sprintf(
			"Завантажує сторінку або API за адресою, як curl. HTML повертається текстом без розмітки, після "+
				"кожного посилання в дужках стоїть його адреса; інше — як є. Адрес не вгадуй: бери їх зі "+
				"сторінок і з пошуку. Публічний канал Telegram читається без входу за адресою "+
				"https://t.me/s/назва_каналу. На одне питання доступно сторінок — %d.",
			t.Budget,
		),
		Parameters: object(map[string]any{"url": text("адреса http(s)")}, "url"),
	}
}

func (t FetchTool) Call(ctx context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if spent(turn, fetchName) > t.Budget {
		return "", errors.New("сторінки на це питання вичерпано: працюй з уже прочитаним")
	}

	return t.Pages.Fetch(ctx, args.URL)
}
