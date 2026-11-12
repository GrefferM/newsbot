package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/research"
	"newsbot/internal/trading"
)

type settled []autopilot.Watch

func (s settled) Settled() []autopilot.Watch {
	return s
}

type bookOf trading.Book

func (b bookOf) Sized(_ context.Context, m trading.Market) trading.Market {
	return m
}

func (b bookOf) Book() trading.Book {
	return trading.Book(b)
}

func TestTheDeskOutcomesTellWhatBecameOfEveryPlanAndItsTrade(t *testing.T) {
	at := time.Date(2026, 11, 13, 20, 44, 0, 0, time.UTC)
	short := research.Scenario{Condition: "ціна пробиває 0.048", Action: "шорт", Risk: research.RiskMedium, Direction: research.DirectionShort, WakeBelow: 0.048, SizePct: 1, Stop: 0.0565, Take: 0.0385}
	tool := OutcomesTool{
		Watched: settled{
			{Asset: "AGT", Plan: short, State: autopilot.Fired, Note: "пробій підтверджено", Woke: "настала дата", Checked: at},
			{Asset: "AGT", Plan: short, State: autopilot.Refuted, Note: "імпульс уже минув", Checked: at.Add(-time.Hour)},
			{Asset: "ZEC", Plan: short, State: autopilot.Expired, Checked: at.Add(-2 * time.Hour)},
		},
		Trader: bookOf{Positions: []trading.Position{
			{Symbol: "AGT", Opened: at.Add(-time.Hour), Entry: 9},
			{Symbol: "AGT", Opened: at.Add(20 * time.Second), Entry: 0.0479, Exit: 0.0565, ClosedAt: at.Add(3 * time.Hour), Reason: "стоп", Return: -0.18, PnL: -1.8},
		}},
	}

	all, err := use(t, tool, `{}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`"fired":1`, `"refuted":1`, `"expired":1`, `"trades_lost":1`, `"trades_won":0`, `"closed_by":"стоп"`, `"return_fraction":-0.18`, `"entry_price":0.0479`, "імпульс уже минув", `"shown":3`} {
		if !strings.Contains(all, want) {
			t.Errorf("no %s in %s", want, all)
		}
	}

	if strings.Index(all, "пробій підтверджено") > strings.Index(all, "імпульс уже минув") || strings.Contains(all, `"entry_price":9`) {
		t.Errorf("the newest first, each plan with its own trade: %s", all)
	}

	one, err := use(t, tool, `{"asset":"zec","limit":1}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(one, `"shown":1`) || strings.Contains(one, "AGT") {
		t.Errorf("one coin's plans: %s", one)
	}

	empty, _ := use(t, OutcomesTool{Watched: settled{}, Trader: stubAccount{}}, `{}`, &agents.Turn{})

	if !strings.Contains(empty, `"history":[]`) {
		t.Errorf("an empty journal: %s", empty)
	}
}
