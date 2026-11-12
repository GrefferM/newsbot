package monitor

import (
	"context"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

type Source interface {
	Name() string
	Interval() time.Duration
	Poll(ctx context.Context) ([]news.Event, error)
}

type Post struct {
	Event news.Event
	Chart []byte
}

type Publisher interface {
	Publish(ctx context.Context, p Post) error
}

type Illustrator interface {
	Illustrate(ctx context.Context, e news.Event) ([]byte, bool)
}

type Journal interface {
	Record(o news.Observation)
	Fail(f news.Failure)
}

type SeenRepository interface {
	MarkNew(id string) bool
	IsSeeded(source string) bool
	MarkSeeded(source string)
	Prune(maxAge time.Duration)
	Save() error
}

type MarketData interface {
	Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error)
	OpenInterestChange(ctx context.Context, contract string, window time.Duration) (float64, bool)
}

type RetryHint interface {
	RetryAfter() time.Duration
}

type Trader interface {
	Run(ctx context.Context, out chan<- news.Event)
}

type Runner interface {
	Run(ctx context.Context)
}

type Reporter interface {
	Run(ctx context.Context, out chan<- news.Event)
}
