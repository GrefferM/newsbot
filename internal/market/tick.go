package market

import "time"

type Tick struct {
	Contract string
	Asset    string
	Price    float64
	Time     time.Time
}

type Interest struct {
	Time      time.Time
	Contracts float64
}
