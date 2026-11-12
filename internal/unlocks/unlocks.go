package unlocks

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/schedule"
)

type Listed interface {
	Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error)
}

type Memory interface {
	Recall(name string, kept any) (time.Time, error)
	Remember(name string, read time.Time, kept any) error
}

type Unlock struct {
	Asset string
	Price float64
	Value string
	Share string
	At    time.Time
}

func (u Unlock) Entry(url string) schedule.Entry {
	title := "Розблокування $" + u.Value

	if u.Share != "" {
		title = fmt.Sprintf("Розблокування %s%% обігу — $%s", u.Share, u.Value)
	}

	share, _ := strconv.ParseFloat(u.Share, 64)

	return schedule.Entry{
		ID:     "unlock:" + u.Asset + ":" + u.At.Format(time.RFC3339),
		Time:   u.At,
		Title:  title,
		Source: u.Asset,
		URL:    url,
		Kind:   schedule.Unlock,
		Assets: []string{u.Asset},
		Share:  share / 100,
	}
}

func recall(memory Memory, name string) (time.Time, []schedule.Entry) {
	var kept []schedule.Entry

	read, err := memory.Recall(name, &kept)
	if err != nil {
		return time.Time{}, nil
	}

	return read, kept
}
