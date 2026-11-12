package autopilot

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/trading"
)

type State string

const (
	Armed     State = "armed"
	Fired     State = "fired"
	Refuted   State = "refuted"
	Expired   State = "expired"
	Cancelled State = "cancelled"
)

const (
	Confirmed = "підтверджено"
	Denied    = "спростовано"
)

var Decisions = []string{Confirmed, Denied}

type Cause string

const (
	ByPrice Cause = "ціна дійшла до порога плану"
	ByEvent Cause = "бот побачив нову подію про цю монету"
)

type Verdict struct {
	Decision string
	Why      string
}

type Watch struct {
	ID      string            `json:"id"`
	Asset   string            `json:"asset"`
	Plan    research.Scenario `json:"plan"`
	State   State             `json:"state"`
	Note    string            `json:"note,omitempty"`
	Woke    string            `json:"woke,omitempty"`
	Armed   time.Time         `json:"armed"`
	Expires time.Time         `json:"expires"`
	Checked time.Time         `json:"checked,omitzero"`
	Checks  int               `json:"checks"`
	After   time.Time         `json:"not_before,omitzero"`
}

type Tally struct {
	Armed     int `json:"armed"`
	Fired     int `json:"fired"`
	Refuted   int `json:"refuted"`
	Expired   int `json:"expired"`
	Cancelled int `json:"cancelled"`
}

func (ws Watches) Tally() Tally {
	var t Tally

	for _, w := range ws.Items {
		switch w.State {
		case Armed:
			t.Armed++
		case Fired:
			t.Fired++
		case Refuted:
			t.Refuted++
		case Expired:
			t.Expired++
		case Cancelled:
			t.Cancelled++
		}
	}

	return t
}

func (w Watch) Asleep(now time.Time) bool {
	return now.Before(w.After)
}

func (w Watch) Since(from time.Time) time.Time {
	for _, moment := range []time.Time{w.Armed, w.After, w.Checked} {
		if moment.After(from) {
			from = moment
		}
	}

	return from
}

func (w Watch) Called(price float64, events []news.Event, now time.Time, pause time.Duration) (Cause, bool) {
	switch {
	case w.Roused(events):
		return ByEvent, true
	case w.Woken(price) && w.Rested(now, pause):
		return ByPrice, true
	default:
		return "", false
	}
}

func (w Watch) Roused(events []news.Event) bool {
	since := w.Armed

	if w.Checked.After(since) {
		since = w.Checked
	}

	for _, e := range events {
		if (e.Category == news.ShortTrigger || e.Category == news.Incident) && e.Time.After(since) && slices.Contains(e.Symbols, news.Symbol(w.Asset)) {
			return true
		}
	}

	return false
}

func (w Watch) Postponed(until time.Time, life time.Duration) Watch {
	if until.After(w.After) {
		w.After = until
	}

	if kept := w.After.Add(life); kept.After(w.Expires) {
		w.Expires = kept
	}

	return w
}

func (w Watch) Woken(price float64) bool {
	if w.Plan.WakeAbove > 0 {
		return price >= w.Plan.WakeAbove
	}

	return price <= w.Plan.WakeBelow
}

func (w Watch) Rested(now time.Time, pause time.Duration) bool {
	return w.Checked.IsZero() || now.Sub(w.Checked) >= pause
}

func (w Watch) Settled(state State, note string) Watch {
	w.State, w.Note = state, note

	return w
}

func (w Watch) Stirred(cause Cause, price float64) Watch {
	w.Woke = fmt.Sprintf("%s · ціна %s", cause, news.FormatPrice(price))

	return w
}

func (w Watch) Asked(now time.Time, note string) Watch {
	w.Checked, w.Note = now, note
	w.Checks++

	return w
}

func (w Watch) Draft(id string, contract string, price, equity float64, hold time.Duration, now time.Time) (trading.Draft, error) {
	plan := w.Plan
	side, stop, take := trading.Short, plan.Stop/price-1, 1-plan.Take/price

	if plan.Long() {
		side, stop, take = trading.Long, 1-plan.Stop/price, plan.Take/price-1
	}

	if stop <= 0 || take <= 0 || math.IsInf(stop, 0) || math.IsNaN(stop) {
		return trading.Draft{}, errors.New("ціна вже за стопом або тейком плану")
	}

	return trading.Draft{
		ID:         id,
		Symbol:     news.Symbol(w.Asset),
		Side:       side,
		Contract:   contract,
		Stake:      equity * plan.SizePct / 100,
		StopLoss:   stop,
		TakeProfit: take,
		Hold:       hold,
		Note:       "план: " + plan.Condition,
		Created:    now,
	}, nil
}

type Watches struct {
	Items []Watch
}

func (ws Watches) Of(asset string) []Watch {
	out := []Watch{}

	for _, w := range ws.Items {
		if w.Asset == asset {
			out = append(out, w)
		}
	}

	return out
}

func (ws Watches) Active() []Watch {
	var out []Watch

	for _, w := range ws.Items {
		if w.State == Armed {
			out = append(out, w)
		}
	}

	return out
}

func (ws Watches) ByAsset() [][]Watch {
	var groups [][]Watch

	at := map[string]int{}

	for _, w := range ws.Active() {
		i, seen := at[w.Asset]
		if !seen {
			i = len(groups)
			at[w.Asset] = i
			groups = append(groups, nil)
		}

		groups[i] = append(groups[i], w)
	}

	return groups
}

func (ws Watches) With(w Watch) Watches {
	items := slices.Clone(ws.Items)

	for i, old := range items {
		if old.ID == w.ID {
			items[i] = w

			return Watches{Items: items}
		}
	}

	return Watches{Items: append(items, w)}
}

func (ws Watches) Find(id string) (Watch, bool) {
	for _, w := range ws.Items {
		if w.ID == id {
			return w, true
		}
	}

	return Watch{}, false
}

func (w Watch) Amended(numbers research.Scenario, life time.Duration) (Watch, error) {
	plan := w.Plan.Retimed(numbers)

	if err := plan.Armable(); err != nil {
		return w, err
	}

	w.Plan, w.Note, w.After = plan, "змінено власником", plan.NotBefore()

	if kept := w.After.Add(life); kept.After(w.Expires) {
		w.Expires = kept
	}

	return w, nil
}

func (ws Watches) Arm(id, asset string, plan research.Scenario, now time.Time, life time.Duration) (Watches, Watch, error) {
	if err := plan.Armable(); err != nil {
		return ws, Watch{}, err
	}

	for _, w := range ws.Active() {
		if w.Asset == asset && w.Plan.Condition == plan.Condition {
			return ws, Watch{}, fmt.Errorf("за цим планом уже стежу")
		}
	}

	w := Watch{ID: id, Asset: asset, Plan: plan, State: Armed, Armed: now, Expires: now.Add(life)}
	w = w.Postponed(plan.NotBefore(), life)

	return ws.With(w), w, nil
}
