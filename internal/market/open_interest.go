package market

import (
	"fmt"
	"time"

	"newsbot/internal/news"
)

const (
	openInterestShift     = 0.10
	MinOpenInterestWindow = time.Hour
)

func OpenInterestLine(change float64, window time.Duration) string {
	meaning := "без різких змін"

	switch {
	case change >= openInterestShift:
		meaning = "відкриваються нові позиції"
	case change <= -openInterestShift:
		meaning = "позиції закриваються"
	}

	return fmt.Sprintf("Open Interest за %.0f год %s — %s", window.Hours(), news.FormatPercent(change*100), meaning)
}
