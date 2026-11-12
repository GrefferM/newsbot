package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"newsbot/internal/agents"
	"newsbot/internal/watchlist"
)

const (
	directionShort = "шорт"
	directionLong  = "лонг"
)

var directions = []string{directionShort, directionLong}

type PickTool struct {
	Watchlist Coins
	Limit     int
}

func (t PickTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "pick_finalist",
		Description: fmt.Sprintf(
			"Називає одного фіналіста відбору: монету, яку далі буде досліджено до дна окремо, з платними "+
				"джерелами. Фіналіст має мати перп на Binance. Фіналістів на одне питання — щонайбільше %d, "+
				"тож називай лише найсильніших.",
			t.Limit,
		),
		Parameters: object(map[string]any{
			"symbol":    text("тікер монети"),
			"direction": enum(directions...),
			"why":       text("чому саме вона: числа й факти одним-двома реченнями"),
		}, "symbol", "direction", "why"),
	}
}

func (t PickTool) Call(_ context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args struct {
		Symbol    string `json:"symbol"`
		Direction string `json:"direction"`
		Why       string `json:"why"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	asset := watchlist.Normalize(args.Symbol)

	if !slices.Contains(directions, args.Direction) || strings.TrimSpace(args.Why) == "" {
		return "", errors.New("потрібні direction (шорт або лонг) і why")
	}

	listed := slices.ContainsFunc(t.Watchlist.Coins(), func(k watchlist.Kept) bool {
		return k.Coin.Asset == asset
	})

	if !listed {
		return "", fmt.Errorf("монети %s немає серед монет з перпом у зрізі", asset)
	}

	named := slices.ContainsFunc(turn.Picks, func(p agents.Pick) bool {
		return p.Asset == asset
	})

	if named {
		return "", errors.New("цю монету вже названо")
	}

	if len(turn.Picks) >= t.Limit {
		return "", fmt.Errorf("фіналістів уже %d, більше не можна", t.Limit)
	}

	turn.Picks = append(turn.Picks, agents.Pick{Asset: asset, Direction: args.Direction, Why: strings.TrimSpace(args.Why)})

	return "фіналіста прийнято", nil
}
