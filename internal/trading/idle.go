package trading

import (
	"context"
	"errors"
)

var ErrNoVenue = errors.New("торговий рахунок не підключено")

type Idle struct{}

func (Idle) Venue() string {
	return ""
}

func (Idle) Balance(context.Context) (float64, error) {
	return 0, ErrNoVenue
}

func (Idle) Earned(context.Context, Position) (float64, error) {
	return 0, ErrNoVenue
}

func (Idle) Smallest(context.Context, string, float64) (float64, error) {
	return 0, ErrNoVenue
}

func (Idle) Enter(context.Context, Position) (Fill, error) {
	return Fill{}, ErrNoVenue
}

func (Idle) Protect(context.Context, Position) error {
	return ErrNoVenue
}

func (Idle) Exited(context.Context, Position) (Exit, bool, error) {
	return Exit{}, false, ErrNoVenue
}

func (Idle) Close(context.Context, Position) (float64, error) {
	return 0, ErrNoVenue
}
