package tools

import (
	"context"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/market"
	"newsbot/internal/trading"
)

const ideaRecords = 5

type Journal interface {
	Last(agent, topic string, count int) ([]agents.Record, error)
}

type DeskRules struct {
	Limits    trading.Limits
	Hold      time.Duration
	WatchDays int
	Analyst   string
}

type cardDesk struct {
	Short     []cardCheck `json:"entry_checks_short"`
	Long      []cardCheck `json:"entry_checks_long"`
	Smallest  float64     `json:"smallest_order_pct_of_account,omitempty"`
	Halted    bool        `json:"new_entries_stopped_by_daily_loss"`
	HoldHours float64     `json:"position_closed_by_time_after_hours"`
	WatchDays int         `json:"plan_watched_for_days"`
}

type cardCheck struct {
	Name   string `json:"check"`
	Passes bool   `json:"passes"`
	Value  string `json:"value"`
}

type cardIdea struct {
	At        time.Time `json:"at"`
	Direction string    `json:"direction"`
	Why       string    `json:"why"`
	Cancel    string    `json:"cancel"`
	Event     string    `json:"event,omitempty"`
	Price     float64   `json:"price_then"`
}

func (t ServerTool) desk(ctx context.Context, ticker market.Ticker, now time.Time) *cardDesk {
	book := t.Trader.Book()
	contract := t.Trader.Sized(ctx, trading.Market{
		Listed:      true,
		Contract:    ticker.Contract,
		Price:       ticker.Price,
		Funding:     ticker.Funding,
		QuoteVolume: ticker.QuoteVolume,
	})

	return &cardDesk{
		Smallest:  contract.MinSize * 100,
		Short:     checked(t.Rules.Limits.Checks(contract, trading.Short, 0, book.Exposure())),
		Long:      checked(t.Rules.Limits.Checks(contract, trading.Long, 0, book.Exposure())),
		Halted:    book.Halted(t.Rules.Limits, now),
		HoldHours: t.Rules.Hold.Hours(),
		WatchDays: t.Rules.WatchDays,
	}
}

func checked(checks []trading.Check) []cardCheck {
	out := make([]cardCheck, 0, len(checks))

	for _, c := range checks {
		out = append(out, cardCheck{Name: c.Name, Passes: c.OK, Value: c.Value})
	}

	return out
}

func (t ServerTool) ideas(asset string) []cardIdea {
	records, err := t.Journal.Last(t.Rules.Analyst, "", ideaRecords)
	if err != nil {
		return nil
	}

	var out []cardIdea

	for _, r := range records {
		for _, idea := range r.Ideas {
			if idea.Asset == asset {
				out = append(out, cardIdea{At: r.At, Direction: idea.Direction, Why: idea.Why, Cancel: idea.Cancel, Event: idea.Event, Price: idea.Price})
			}
		}
	}

	return out
}
