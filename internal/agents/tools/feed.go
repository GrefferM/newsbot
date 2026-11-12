package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"newsbot/internal/agents"
	"newsbot/internal/googlenews"
)

type Headlines interface {
	Headlines(ctx context.Context, query string, days int) ([]googlenews.Headline, error)
}

type FeedTool struct {
	News Headlines
}

func (FeedTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: feedName,
		Description: "Безкоштовна стрічка заголовків новинних видань: дата, заголовок, видання. queries — список " +
			"окремих запитів, як у пошуковику, англійською; про кожну монету — свій запит із назвою проєкту й " +
			"тікером, без слів про ціну чи курс: вони приносять перекази руху замість подій. Усі запити — в одному виклику. Заголовки повертаються окремо за кожним запитом. days — за " +
			"скільки останніх днів. З неї починається будь-яка перевірка новин; платні інструменти, якщо вони в " +
			"тебе є, — лише для того, чого стрічка не закрила.",
		Parameters: object(map[string]any{
			"queries": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "окремі пошукові запити",
			},
			"days": map[string]any{"type": "integer", "description": "за скільки останніх днів"},
		}, "queries"),
	}
}

type asked struct {
	Query     string                `json:"query"`
	Headlines []googlenews.Headline `json:"headlines"`
	Error     string                `json:"error,omitempty"`
}

func (t FeedTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Queries []string `json:"queries"`
		Days    int      `json:"days"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	queries := slices.DeleteFunc(args.Queries, blankQuery)

	if len(queries) == 0 {
		return "", errors.New("потрібен хоча б один запит у queries")
	}

	if len(queries) > feedQueries {
		return "", fmt.Errorf("забагато запитів в одному виклику: %d, можна до %d", len(queries), feedQueries)
	}

	days := feedDays

	if args.Days > 0 {
		days = min(args.Days, feedLimit)
	}

	out := make([]asked, 0, len(queries))

	for _, query := range queries {
		found, err := t.News.Headlines(ctx, query, days)

		answer := asked{Query: query, Headlines: found}

		if err != nil {
			answer.Error = err.Error()
		}

		out = append(out, answer)
	}

	return encode(out)
}

func blankQuery(query string) bool {
	return strings.TrimSpace(query) == ""
}
