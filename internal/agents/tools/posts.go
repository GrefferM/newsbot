package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/xapi"
)

type Feed interface {
	Recent(ctx context.Context, query, order string, since time.Time) ([]xapi.Post, error)
}

type PostsTool struct {
	Posts  Feed
	Budget int
}

func (t PostsTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: postsName,
		Description: fmt.Sprintf(
			"Читає публічні пости X (Twitter) за останні сім днів через офіційний API: текст, час, адресу, "+
				"вподобання й поширення. account — останні власні пости цього акаунта без відповідей і "+
				"поширень; query — найпомітніші пости за запитом у синтаксисі пошуку X, від кого завгодно; "+
				"через OR в одному запиті можна спитати про кілька монет. "+
				"Потрібне одне з двох. "+
				"Читання платне: на одне питання доступно викликів — %d.",
			t.Budget,
		),
		Parameters: object(map[string]any{
			"account": text("ім'я акаунта без @"),
			"query":   text("пошуковий запит X"),
		}),
	}
}

func (t PostsTool) Call(ctx context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args struct {
		Account string `json:"account"`
		Query   string `json:"query"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	query := strings.TrimSpace(args.Query)
	order := xapi.Relevant

	if account := strings.TrimPrefix(strings.TrimSpace(args.Account), "@"); account != "" {
		query = strings.TrimSpace("from:" + account + " " + query)
		order = ""
	}

	if query == "" {
		return "", errors.New("потрібен account або query")
	}

	if err := paid(turn, postsName, t.Budget); err != nil {
		return "", err
	}

	posts, err := t.Posts.Recent(ctx, query+ownPosts, order, time.Time{})
	if err != nil {
		return "", err
	}

	return encode(posts)
}
