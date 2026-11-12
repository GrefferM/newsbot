package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"newsbot/internal/agents"
	"newsbot/internal/tavily"
)

type Searcher interface {
	Search(ctx context.Context, q tavily.Query) ([]tavily.Result, error)
}

type SearchTool struct {
	Search Searcher
	Budget int
}

func (t SearchTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: searchName,
		Description: fmt.Sprintf(
			"Шукає в інтернеті й повертає знайдені сторінки: заголовок, адресу, уривок і дату сторінки. "+
				"Саму сторінку читає http_get; якщо вона не відкривається, уривок лишається відомим фактом із цією "+
				"адресою. fresh звужує пошук до сторінок за останній день, тиждень, місяць чи рік; news шукає "+
				"лише серед новинних видань. Запит пиши мовою джерел, які шукаєш; в одному запиті можна "+
				"спитати про кілька монет. Пошук платний: "+
				"на одне питання доступно викликів — %d.",
			t.Budget,
		),
		Parameters: object(map[string]any{
			"query": text("пошуковий запит"),
			"fresh": enum(tavily.Periods...),
			"news":  map[string]any{"type": "boolean", "description": "true — лише новинні видання"},
		}, "query"),
	}
}

func (t SearchTool) Call(ctx context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args struct {
		Query string `json:"query"`
		Fresh string `json:"fresh"`
		News  bool   `json:"news"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if strings.TrimSpace(args.Query) == "" {
		return "", errors.New("порожній запит")
	}

	if err := paid(turn, searchName, t.Budget); err != nil {
		return "", err
	}

	found, err := t.Search.Search(ctx, tavily.Query{Text: args.Query, Period: args.Fresh, News: args.News})
	if err != nil {
		return "", err
	}

	return encode(found)
}
