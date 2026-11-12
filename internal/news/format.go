package news

import (
	"fmt"
	"math"
)

func FormatUSD(v float64) string {
	a := math.Abs(v)

	switch {
	case a >= 1e9:
		return fmt.Sprintf("$%.2fB", v/1e9)
	case a >= 1e6:
		return fmt.Sprintf("$%.1fM", v/1e6)
	case a >= 1e3:
		return fmt.Sprintf("$%.0fK", v/1e3)
	default:
		return fmt.Sprintf("$%.0f", v)
	}
}

func FormatPrice(v float64) string {
	a := math.Abs(v)

	switch {
	case a >= 1000:
		return fmt.Sprintf("$%.0f", v)
	case a >= 1:
		return fmt.Sprintf("$%.3f", v)
	case a == 0:
		return "$0"
	default:
		digits := int(math.Ceil(-math.Log10(a))) + 3

		return fmt.Sprintf("$%.*f", digits, v)
	}
}

func FormatPercent(v float64) string {
	return fmt.Sprintf("%+.2f%%", v)
}
