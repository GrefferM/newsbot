package trading

import (
	"context"
	"fmt"
	"sync"

	"newsbot/internal/news"
)

type fakeVenue struct {
	market  MarketData
	mu      sync.Mutex
	balance float64
	fees    float64
	late    int
}

func newFakeVenue(market MarketData, balance float64) *fakeVenue {
	return &fakeVenue{market: market, balance: balance}
}

func (v *fakeVenue) Venue() string {
	return "testnet"
}

func (v *fakeVenue) Balance(context.Context) (float64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.balance, nil
}

func (v *fakeVenue) Earned(_ context.Context, p Position) (float64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.late > 0 {
		v.late--

		return 0, fmt.Errorf("not booked yet")
	}

	earned := p.PnL - v.fees
	v.balance += earned

	return earned, nil
}

func (v *fakeVenue) Smallest(context.Context, string, float64) (float64, error) {
	return 0, nil
}

func (v *fakeVenue) Enter(ctx context.Context, p Position) (Fill, error) {
	price, err := v.price(ctx, p.Symbol)
	if err != nil {
		return Fill{}, err
	}

	return Fill{Price: price, Quantity: p.Stake / price}, nil
}

func (v *fakeVenue) Protect(context.Context, Position) error {
	return nil
}

func (v *fakeVenue) Exited(ctx context.Context, p Position) (Exit, bool, error) {
	price, err := v.price(ctx, p.Symbol)
	if err != nil {
		return Exit{}, false, err
	}

	reason, hit := p.Triggered(price)

	if reason == ExitTarget {
		price = p.Target
	}

	return Exit{Price: price, Reason: reason}, hit, nil
}

func (v *fakeVenue) Close(ctx context.Context, p Position) (float64, error) {
	return v.price(ctx, p.Symbol)
}

func (v *fakeVenue) price(ctx context.Context, symbol news.Symbol) (float64, error) {
	ticker, listed, err := v.market.Ticker(ctx, symbol)
	if err != nil {
		return 0, err
	}

	if !listed || ticker.Price <= 0 {
		return 0, fmt.Errorf("no futures price for %s", symbol)
	}

	return ticker.Price, nil
}
