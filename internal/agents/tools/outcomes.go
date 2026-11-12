package tools

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/news"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

const (
	outcomesShown = 20
	outcomesMost  = 40
	entryWithin   = 2 * time.Minute
)

type Settled interface {
	Settled() []autopilot.Watch
}

type OutcomesTool struct {
	Watched Settled
	Trader  Account
}

func (OutcomesTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "desk_outcomes",
		Description: "Спільний журнал столу: чим закінчились плани, які власник ставив на стеження, найновіші перші. " +
			"По кожному — монета, напрям, умова, дія, ризик і числа плану; що його розбудило; вердикт судді та його " +
			"пояснення або чому план знято; а якщо за планом увійшли — угода: ціна входу й виходу, чим закрилась " +
			"(стоп, тейк, час чи вручну) і результат. Окремо — підсумок: скільки планів увійшло, відхилено, минуло і " +
			"знято, скільки угод у плюс і в мінус. Це факти про минулі рішення столу, а не думки агентів: за ними " +
			"звіряють новий план чи вердикт із тим, що вже було в подібних випадках. Порожній журнал означає, що " +
			"завершених планів ще не було. asset звужує журнал до однієї монети.",
		Parameters: object(map[string]any{
			"asset": text("тікер: лише плани цієї монети"),
			"limit": map[string]any{"type": "integer", "description": "скільки останніх планів показати"},
		}),
	}
}

type outcome struct {
	Asset     string        `json:"asset"`
	Direction string        `json:"direction"`
	Condition string        `json:"condition"`
	Action    string        `json:"action"`
	Risk      string        `json:"risk"`
	Numbers   string        `json:"numbers"`
	Armed     time.Time     `json:"set_to_watch"`
	Woke      string        `json:"woken_by,omitempty"`
	Settled   time.Time     `json:"settled,omitzero"`
	State     string        `json:"outcome"`
	Why       string        `json:"why,omitempty"`
	Trade     *outcomeTrade `json:"trade,omitempty"`
}

type outcomeTrade struct {
	Entry    float64   `json:"entry_price"`
	Exit     float64   `json:"exit_price,omitempty"`
	Closed   time.Time `json:"closed,omitzero"`
	Reason   string    `json:"closed_by,omitempty"`
	Return   float64   `json:"return_fraction"`
	PnL      float64   `json:"pnl_usd"`
	StillOn  bool      `json:"still_open,omitempty"`
	Deadline time.Time `json:"closes_by_time_at,omitzero"`
}

type outcomes struct {
	Plans   map[string]int `json:"plans_by_outcome"`
	Won     int            `json:"trades_won"`
	Lost    int            `json:"trades_lost"`
	Open    int            `json:"trades_open"`
	Shown   int            `json:"shown"`
	History []outcome      `json:"history"`
}

func (t OutcomesTool) Call(_ context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Asset string `json:"asset"`
		Limit int    `json:"limit"`
	}

	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return "", err
		}
	}

	asset := watchlist.Normalize(args.Asset)
	limit := outcomesShown

	if args.Limit > 0 {
		limit = min(args.Limit, outcomesMost)
	}

	positions := t.Trader.Book().Positions
	out := outcomes{Plans: map[string]int{}, History: []outcome{}}

	for _, w := range t.Watched.Settled() {
		if asset != "" && w.Asset != asset {
			continue
		}

		row := told(w, positions)
		out.Plans[row.State]++

		switch {
		case row.Trade == nil:
		case row.Trade.StillOn:
			out.Open++
		case row.Trade.Return > 0:
			out.Won++
		default:
			out.Lost++
		}

		if len(out.History) < limit {
			out.History = append(out.History, row)
		}
	}

	out.Shown = len(out.History)

	return encode(out)
}

func told(w autopilot.Watch, positions []trading.Position) outcome {
	row := outcome{
		Asset:     w.Asset,
		Direction: w.Plan.Direction,
		Condition: w.Plan.Condition,
		Action:    w.Plan.Action,
		Risk:      w.Plan.Risk,
		Numbers:   w.Plan.Terms(),
		Armed:     w.Armed,
		Woke:      w.Woke,
		Settled:   w.Checked,
		State:     string(w.State),
		Why:       w.Note,
	}

	if w.State != autopilot.Fired {
		return row
	}

	entered := slices.IndexFunc(positions, func(p trading.Position) bool {
		return p.Symbol == news.Symbol(w.Asset) && p.Opened.Sub(w.Checked).Abs() <= entryWithin
	})

	if entered < 0 {
		return row
	}

	p := positions[entered]
	row.Trade = &outcomeTrade{Entry: p.Entry, Exit: p.Exit, Closed: p.ClosedAt, Reason: p.Reason, Return: p.Return, PnL: p.PnL}

	if p.ClosedAt.IsZero() {
		row.Trade.StillOn, row.Trade.Deadline = true, p.Deadline
	}

	return row
}
