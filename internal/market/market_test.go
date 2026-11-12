package market

import (
	"strings"
	"testing"
	"time"
)

func TestOpenInterestLineNamesTheShift(t *testing.T) {
	cases := map[float64]string{
		0.25:  "+25.00% — відкриваються нові позиції",
		-0.15: "-15.00% — позиції закриваються",
		0.02:  "+2.00% — без різких змін",
	}

	for change, want := range cases {
		if got := OpenInterestLine(change, time.Hour); !strings.Contains(got, "за 1 год "+want) {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
