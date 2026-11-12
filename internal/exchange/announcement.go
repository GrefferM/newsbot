package exchange

import (
	"strings"
	"time"

	"newsbot/internal/news"
)

const (
	MutedStale      = "застарілий"
	MutedIrrelevant = "не цікавий"
)

type Announcement struct {
	Title     string
	URL       string
	Published time.Time
}

func (a Announcement) ID() string {
	return "binance:" + strings.ToLower(strings.Join(strings.Fields(a.Title), " "))
}

type AnnouncementPolicy struct {
	MaxAge time.Duration
}

func (p AnnouncementPolicy) Event(a Announcement, now time.Time) news.Event {
	c, ok := ClassifyBinance(a.Title)

	e := c.Event(news.Event{
		ID:     a.ID(),
		Source: "Binance",
		Title:  a.Title,
		URL:    a.URL,
		Time:   a.Published.UTC(),
	})

	switch {
	case !ok:
		e.Muted = MutedIrrelevant
	case now.Sub(a.Published) > p.MaxAge:
		e.Muted = MutedStale
	}

	return e
}
