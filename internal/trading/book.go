package trading

import (
	"cmp"
	"math"
	"time"

	"newsbot/internal/news"
)

const (
	ExitStop        = "стоп"
	ExitTarget      = "тейк"
	ExitTime        = "строк вийшов"
	ExitUnprotected = "без стопу"
	ExitManual      = "вручну"
)

type Side string

const (
	Short Side = "short"
	Long  Side = "long"
)

func (s Side) Name() string {
	if s == Long {
		return "лонг"
	}

	return "шорт"
}

type Position struct {
	ID       string
	Symbol   news.Symbol
	Contract string
	Side     Side
	Kind     string
	Size     float64
	Stake    float64
	Quantity float64
	Entry    float64
	Stop     float64
	Target   float64
	Opened   time.Time
	Deadline time.Time
	Exit     float64
	ClosedAt time.Time
	Reason   string
	Return   float64
	PnL      float64
	Booked   bool
}

type Book struct {
	Start     float64
	Equity    float64
	Base      float64
	Balance   float64
	Positions []Position
}

type Record struct {
	Open   int     `json:"open"`
	Closed int     `json:"closed"`
	Won    int     `json:"won"`
	Lost   int     `json:"lost"`
	PnL    float64 `json:"pnl_usd"`
}

func (b Book) Record() Record {
	var r Record

	for _, p := range b.Positions {
		if p.Open() {
			r.Open++

			continue
		}

		r.Closed++
		r.PnL += p.PnL

		if p.PnL > 0 {
			r.Won++
		} else {
			r.Lost++
		}
	}

	return r
}

func (b Book) Funded(balance, capital float64) Book {
	start := capital

	if start <= 0 {
		start = b.Base
	}

	if b.Base <= 0 || b.Start != start {
		b.Base = balance
		b.Start = cmp.Or(capital, balance)
	}

	if jump := balance - b.Balance; b.Balance > 0 && math.Abs(jump) > b.Start {
		b.Base += jump
	}

	b.Balance = balance
	b.Equity = b.Start + balance - b.Base

	return b
}

func (p Position) Open() bool {
	return p.ClosedAt.IsZero()
}

func (s Strategy) Open(d Decision, book Book, now time.Time) Position {
	return s.Filled(Position{
		ID:       d.Market.Contract + "-" + now.UTC().Format("20060102T150405"),
		Symbol:   d.Symbol,
		Contract: d.Market.Contract,
		Side:     d.Side,
		Kind:     d.Kind,
		Size:     d.Size,
		Stake:    book.Equity * d.Size,
		Opened:   now,
		Deadline: now.Add(s.Hold),
	}, d.Market.Price, 0)
}

func (s Strategy) Filled(p Position, price, quantity float64) Position {
	p.Entry = price
	p.Quantity = quantity
	p.Stop = price * (1 + s.StopLoss)
	p.Target = price * (1 - s.TakeProfit)

	if p.Side == Long {
		p.Stop = price * (1 - s.StopLoss)
		p.Target = price * (1 + s.TakeProfit)
	}

	return p
}

func (p Position) ExitAt(price float64, now time.Time) (string, bool) {
	if reason, hit := p.Triggered(price); hit {
		return reason, true
	}

	if p.Expired(now) {
		return ExitTime, true
	}

	return "", false
}

func (p Position) Triggered(price float64) (string, bool) {
	if p.Side == Long {
		return crossed(price <= p.Stop, price >= p.Target)
	}

	return crossed(price >= p.Stop, price <= p.Target)
}

func crossed(stop, target bool) (string, bool) {
	switch {
	case stop:
		return ExitStop, true
	case target:
		return ExitTarget, true
	default:
		return "", false
	}
}

func (p Position) Expired(now time.Time) bool {
	return !now.Before(p.Deadline)
}

func (p Position) Settled(earned float64) Position {
	p.PnL = earned
	p.Booked = true

	if p.Stake > 0 {
		p.Return = earned / p.Stake
	}

	return p
}

func (s Strategy) Close(p Position, price float64, reason string, now time.Time) Position {
	p.Exit = price
	p.ClosedAt = now
	p.Reason = reason
	p.Return = p.Entry/price - 1 - s.EstimatedFee

	if p.Side == Long {
		p.Return = price/p.Entry - 1 - s.EstimatedFee
	}

	p.PnL = p.Stake * p.Return

	return p
}

func (b Book) Exposure() float64 {
	var sum float64

	for _, p := range b.Positions {
		if p.Open() {
			sum += p.Size
		}
	}

	return sum
}

func (b Book) Halted(l Limits, now time.Time) bool {
	day := now.UTC().Truncate(24 * time.Hour)

	var lost float64

	for _, p := range b.Positions {
		if !p.Open() && !p.ClosedAt.Before(day) {
			lost += p.PnL
		}
	}

	return lost < 0 && -lost >= l.DailyLoss*b.Equity
}

func (b Book) With(p Position) Book {
	positions := make([]Position, 0, len(b.Positions)+1)

	replaced := false

	for _, q := range b.Positions {
		if q.ID == p.ID {
			q, replaced = p, true
		}

		positions = append(positions, q)
	}

	if !replaced {
		positions = append(positions, p)
	}

	b.Positions = positions

	return b
}

func (b Book) Unbooked(since time.Time) []Position {
	var out []Position

	for _, p := range b.Positions {
		if !p.Open() && !p.Booked && p.ClosedAt.After(since) {
			out = append(out, p)
		}
	}

	return out
}

func (b Book) OpenPositions() []Position {
	var out []Position

	for _, p := range b.Positions {
		if p.Open() {
			out = append(out, p)
		}
	}

	return out
}
