package autotrade

import (
	"context"
	"errors"

	"newsbot/internal/market"
	"newsbot/internal/trading"
)

var ErrNoStrategy = errors.New("такої стратегії немає")

type Term struct {
	Name  string
	Value string
}

type Strategy struct {
	ID     string
	Name   string
	About  string
	Terms  []Term
	On     bool
	Record trading.Record
	Rule   any
}

type strategy interface {
	describe() Strategy
	retune(raw []byte) ([]byte, func(), error)
	observe(ctx context.Context, ticks []market.Tick, on bool)
}
