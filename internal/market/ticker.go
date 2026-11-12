package market

import (
	"fmt"

	"newsbot/internal/news"
)

type Ticker struct {
	Contract     string
	Price        float64
	QuoteVolume  float64
	Change24h    float64
	Funding      float64
	FundingHours int
}

func (t Ticker) Summary() string {
	return fmt.Sprintf(
		"%s (%s за 24 год), обсяг %s, funding %s",
		news.FormatPrice(t.Price),
		news.FormatPercent(t.Change24h),
		news.FormatUSD(t.QuoteVolume),
		news.FormatPercent(t.Funding*100),
	)
}
