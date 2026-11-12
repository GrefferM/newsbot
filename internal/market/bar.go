package market

import "time"

type Bar struct {
	Open        time.Time
	High        float64
	Low         float64
	Close       float64
	QuoteVolume float64
}
