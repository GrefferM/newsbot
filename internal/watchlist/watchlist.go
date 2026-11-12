package watchlist

import (
	"slices"
	"time"
)

const (
	ReasonTag       = "monitoring-tag"
	ReasonLowVolume = "low-volume"
)

type Coin struct {
	Asset        string
	Tags         []string
	Tagged       bool
	Price        float64
	Volume       float64
	VolumeDay    float64
	Change1d     float64
	Change7d     float64
	HasChanges   bool
	TrendVsBTC   float64
	HasTrend     bool
	Contract     string
	Funding      float64
	FundingHours int
	OpenInterest float64
	DayRange     float64
	SellDay      float64
	SellWeek     float64
	Depth        float64
	SpotVolume   float64
	HasSpot      bool
}

type Scan struct {
	Built time.Time
	Coins map[string]Coin
}

func (c Coin) Tradable() bool {
	return c.Contract != "" && c.HasChanges && c.Volume > 0
}

func (s Scan) Lookup(asset string) (Coin, bool) {
	c, ok := s.Coins[asset]

	return c, ok
}

type Rule struct {
	LowVolume    float64
	ExcludedTags []string
}

func (r Rule) Build(coins []Coin, built time.Time) Scan {
	all := make(map[string]Coin, len(coins))

	for _, c := range coins {
		if c.Tradable() {
			all[c.Asset] = c
		}
	}

	return Scan{Built: built, Coins: all}
}

func (r Rule) Reasons(c Coin) []string {
	if r.excluded(c) {
		return nil
	}

	var out []string

	if c.Tagged {
		out = append(out, ReasonTag)
	}

	if c.HasSpot && c.SpotVolume < r.LowVolume {
		out = append(out, ReasonLowVolume)
	}

	return out
}

func (r Rule) excluded(c Coin) bool {
	for _, tag := range c.Tags {
		if slices.Contains(r.ExcludedTags, tag) {
			return true
		}
	}

	return false
}
