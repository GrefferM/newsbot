package watchlist

import (
	"context"

	"newsbot/internal/news"
)

type Universe interface {
	Coins(ctx context.Context) ([]Coin, error)
}

type OpenInterest interface {
	OpenInterestValue(ctx context.Context, contract string) (float64, bool)
}

type Depth interface {
	DepthValue(ctx context.Context, contract string) (float64, bool)
}

type ListStore interface {
	Save(list List) error
}

type ScanStore interface {
	Save(scan Scan) error
}

type Journal interface {
	Fail(f news.Failure)
}

type Health interface {
	OK(name string, items, fresh int)
	Fail(name string, err error)
}
