package trading

import (
	"context"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

type Fill struct {
	Price    float64
	Quantity float64
}

type Order struct {
	ID       string
	Tag      string
	Contract string
	Price    float64
	Executed float64
}

type Exit struct {
	Price  float64
	Reason string
}

type Broker interface {
	Venue() string
	Balance(ctx context.Context) (float64, error)
	Earned(ctx context.Context, p Position) (float64, error)
	Smallest(ctx context.Context, contract string, price float64) (float64, error)
	Enter(ctx context.Context, p Position) (Fill, error)
	Protect(ctx context.Context, p Position) error
	Reprotect(ctx context.Context, was, next Position) error
	Exited(ctx context.Context, p Position) (Exit, bool, error)
	Close(ctx context.Context, p Position) (float64, error)
	Rest(ctx context.Context, p Position, tag string) (string, error)
	Orders(ctx context.Context) ([]Order, error)
	Cancel(ctx context.Context, contract, id string) (Fill, error)
	Filled(ctx context.Context, contract, id string) (Fill, error)
	Fills(ctx context.Context, each func(Order)) error
}

type BookStore interface {
	Save(book Book) error
}

type MarketData interface {
	Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error)
	OpenInterestChange(ctx context.Context, contract string, window time.Duration) (float64, bool)
}

type Journal interface {
	Record(o news.Observation)
	Fail(f news.Failure)
}
