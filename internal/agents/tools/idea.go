package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"newsbot/internal/agents"
	"newsbot/internal/market"
	"newsbot/internal/radar"
	"newsbot/internal/watchlist"
)

type IdeaTool struct {
	Market  Pulses
	Rule    radar.Rule
	TradFi  Pulses
	Limit   int
	Crowded float64
}

func (t IdeaTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "trade_idea",
		Description: fmt.Sprintf(
			"Передає власнику одну ідею короткої угоди: інструмент — монету чи контракт TradFi, — напрям, чому саме зараз, що її скасовує і яка "+
				"подія про монету стоїть поруч. Погана новина про монету діє за шорт і проти лонга, добра — за лонг "+
				"і проти шорта; ідею, проти якої є подія, інструмент не прийме. "+
				"Контракт має бути серед тих, що їх показує пульс монет або TradFi. Ціну й фандинг до ідеї додає сам "+
				"інструмент. Ідею в бік, який уже переповнений за фандингом, він не прийме: стіл такого входу "+
				"не пропускає. Ідей на одне питання — щонайбільше %d, одна на монету; слабку ідею не передавай зовсім.",
			t.Limit,
		),
		Parameters: object(map[string]any{
			"symbol":       text("тікер монети"),
			"direction":    enum(directions...),
			"why":          text("чому саме зараз: числа пульсу одним-двома реченнями"),
			"cancel":       text("що скасовує ідею: рух ціни чи подія, після яких вона вже не діє"),
			"event":        text("свіжа подія про монету одним реченням із виданням; порожньо, якщо події немає"),
			"event_effect": enum(eventEffects...),
		}, "symbol", "direction", "why", "cancel", "event_effect"),
	}
}

type ideaArgs struct {
	Symbol    string `json:"symbol"`
	Direction string `json:"direction"`
	Why       string `json:"why"`
	Cancel    string `json:"cancel"`
	Event     string `json:"event"`
	Effect    string `json:"event_effect"`
}

func (a ideaArgs) checked() error {
	if !slices.Contains(directions, a.Direction) || strings.TrimSpace(a.Why) == "" || strings.TrimSpace(a.Cancel) == "" {
		return errors.New("потрібні direction (шорт або лонг), why і cancel")
	}

	switch {
	case !slices.Contains(eventEffects, a.Effect):
		return fmt.Errorf("event_effect має бути одним із: %s", strings.Join(eventEffects, ", "))
	case a.Effect == eventAgainst:
		return errors.New("проти ідеї є подія: таку ідею не передають")
	case (a.Effect == eventNone) != (strings.TrimSpace(a.Event) == ""):
		return errors.New("event і event_effect мають збігатися: є подія — назви її, немає — лиши event порожнім")
	}

	return nil
}

func (t IdeaTool) Call(ctx context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args ideaArgs

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if err := args.checked(); err != nil {
		return "", err
	}

	asset := watchlist.Normalize(args.Symbol)

	coin, woke, err := t.pulsed(ctx, asset)
	if err != nil {
		return "", err
	}

	if err := t.admitted(turn, args.Direction, asset, coin); err != nil {
		return "", err
	}

	turn.Ideas = append(turn.Ideas, agents.Idea{
		Asset:     asset,
		Direction: args.Direction,
		Why:       strings.TrimSpace(args.Why),
		Cancel:    strings.TrimSpace(args.Cancel),
		Price:     coin.Price,
		Funding:   coin.Funding,
		Event:     strings.TrimSpace(args.Event),
		Woke:      woke,
	})

	return "ідею передано", nil
}

func (t IdeaTool) pulsed(ctx context.Context, asset string) (market.Pulse, bool, error) {
	for _, source := range []Pulses{t.Market, t.TradFi} {
		if source == nil {
			continue
		}

		pulses, err := source.Pulse(ctx)
		if err != nil {
			return market.Pulse{}, false, err
		}

		for _, p := range pulses {
			if p.Asset == asset {
				_, woke := t.Rule.Woke(p)

				return p, woke, nil
			}
		}
	}

	return market.Pulse{}, false, fmt.Errorf("контракту %s немає в пульсі монет і TradFi", asset)
}

func (t IdeaTool) admitted(turn *agents.Turn, direction, asset string, coin market.Pulse) error {
	if t.crowded(direction, coin.Funding) {
		return fmt.Errorf(
			"%s по %s переповнений: фандинг %+.3f%% за 8 год, стіл такого входу не пропустить",
			direction, asset, coin.Funding*100,
		)
	}

	told := slices.ContainsFunc(turn.Ideas, func(i agents.Idea) bool {
		return i.Asset == asset
	})

	if told {
		return errors.New("про цю монету ідею вже передано")
	}

	if len(turn.Ideas) >= t.Limit {
		return fmt.Errorf("ідей уже %d, більше не можна", t.Limit)
	}

	return nil
}

const (
	eventNone    = "події немає"
	eventFor     = "за ідею"
	eventAgainst = "проти ідеї"
)

var eventEffects = []string{eventNone, eventFor, eventAgainst}

func (t IdeaTool) crowded(direction string, funding float64) bool {
	if direction == directionLong {
		return funding > -t.Crowded
	}

	return funding < t.Crowded
}
