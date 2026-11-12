package market

type Pulse struct {
	Asset           string
	Contract        string
	Price           float64
	Change15m       float64
	Change1h        float64
	Change4h        float64
	Change24h       float64
	VolumeHour      float64
	VolumeSurge     float64
	BuyShare        float64
	QuarterSurge    float64
	QuarterBuyShare float64
	LegMinutes      int
	LegMove         float64
	Runs            []Run
	RangePosition   float64
	Funding         float64
	VolumeDay       float64
	Beta            float64
	Sigma           float64
	Known           bool
	Crowd           *Crowd
	Hours           []Hour
	Wide            bool
}

type Hour struct {
	Open   int64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

type Crowd struct {
	LongShare  float64
	HourChange float64
	Interest   []float64
	Longs      []float64
}

type Run struct {
	Hours    int
	Move     float64
	Sigmas   float64
	Straight float64
	Place    float64
}
