package market

import "time"

type Bar struct {
	Open        time.Time
	Close       float64
	QuoteVolume float64
}
