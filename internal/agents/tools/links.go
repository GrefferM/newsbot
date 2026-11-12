package tools

import (
	"context"
	"encoding/json"
	"errors"

	"newsbot/internal/agents"
	"newsbot/internal/coingecko"
	"newsbot/internal/watchlist"
)

type Directory interface {
	Links(ctx context.Context, asset string) (coingecko.Links, error)
}

type LinksTool struct {
	Directory Directory
}

func (LinksTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "project_links",
		Description: "Офіційні адреси проєкту монети за даними CoinGecko: сайт, акаунт X, Telegram, документація. " +
			"Якщо той самий тікер мають кілька монет, повертає найбільшу за капіталізацією і називає решту.",
		Parameters: object(map[string]any{"symbol": text("тікер монети")}, "symbol"),
	}
}

func (t LinksTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Symbol string `json:"symbol"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	asset := watchlist.Normalize(args.Symbol)
	if asset == "" {
		return "", errors.New("потрібен тікер")
	}

	links, err := t.Directory.Links(ctx, asset)
	if err != nil {
		return "", err
	}

	return encode(links)
}
