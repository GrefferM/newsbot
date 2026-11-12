package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/research"
)

type ScenarioTool struct{}

func (ScenarioTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "plan_scenario",
		Description: "Додає один план входу: за якої умови входити, що саме робити і який ризик цього входу. " +
			"Плани власник бачить від найризикованішого до найобережнішого. Власник може поставити план на " +
			"автоматичне стеження, якщо план має числа: direction, тригер, size_pct, stop_price і take_price. " +
			"Тригер — ціна-поріг (wake_price_above або wake_price_below): умову перевірять, коли ціна до нього дійде. " +
			"До порога можна додати wake_after — момент події: до нього план спить, а після прокидається на порозі; " +
			"самої дати без порога для стеження не досить. План будить і нова подія бота про інструмент. Умову " +
			"перевіряють один раз і входять, лише якщо вхід проходить перевірки столу; не підтверджений план " +
			"знімається. Позицію стіл закриває за часом, а план, до порога якого ціна так і не дійшла, знімає зі " +
			"стеження — обидва строки є в картці монети.",
		Parameters: object(map[string]any{
			"condition":        text("умова входу: що має статися, однією короткою фразою"),
			"action":           text("дія власника за цієї умови, однією короткою фразою"),
			"risk":             enum(research.Risks...),
			"why":              text("чому ризик саме такий, одним коротким реченням"),
			"direction":        enum(research.Directions...),
			"wake_price_above": number("поріг: перевіряти умову, коли ціна перпа стане не нижчою за цю"),
			"wake_price_below": number("поріг: перевіряти умову, коли ціна перпа стане не вищою за цю"),
			"size_pct":         number("розмір позиції у відсотках рахунку"),
			"stop_price":       number("ціна стопу"),
			"take_price":       number("ціна тейку"),
			"wake_after":       text("момент події, лише разом із ціною-порогом: раніше умову не перевірятимуть, а після — коли ціна дійде до порога; РРРР-ММ-ДД або з часом, РРРР-ММ-ДДTГГ:ХХ:ССZ"),
		}, "condition", "action", "risk", "why"),
	}
}

func (ScenarioTool) Call(_ context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var s research.Scenario

	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}

	if !slices.Contains(research.Risks, s.Risk) || s.Condition == "" || s.Action == "" {
		return "", fmt.Errorf("потрібні condition, action і risk з %v", research.Risks)
	}

	if s.Direction == research.DirectionNone || !s.Measured() {
		turn.Scenarios = append(turn.Scenarios, s)

		return "додано; план без чисел на стеження не ставиться", nil
	}

	if err := s.Armable(); err != nil {
		return "", err
	}

	turn.Scenarios = append(turn.Scenarios, s)

	return "додано; план можна поставити на стеження", nil
}

type VerdictTool struct{}

func (VerdictTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "entry_verdict",
		Description: "Вердикт щодо плану на стеженні: чи виконалась його умова і чи чинні причини. Від " +
			"«підтверджено» буде поставлено ордер; від «спростовано» план знімається зі стеження, і новий власник " +
			"попросить сам. Третього вердикту немає: стіл не перевіряє той самий план удруге.",
		Parameters: object(map[string]any{
			"decision": enum(autopilot.Decisions...),
			"why":      text("що саме перевірено і чому вердикт такий, одним-двома реченнями"),
		}, "decision", "why"),
	}
}

func (VerdictTool) Call(_ context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args struct {
		Decision string `json:"decision"`
		Why      string `json:"why"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if !slices.Contains(autopilot.Decisions, args.Decision) || strings.TrimSpace(args.Why) == "" {
		return "", fmt.Errorf("потрібні decision з %v і why", autopilot.Decisions)
	}

	turn.Verdict = agents.Verdict{Decision: args.Decision, Why: strings.TrimSpace(args.Why)}

	return "вердикт прийнято", nil
}
