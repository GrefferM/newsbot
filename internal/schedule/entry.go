package schedule

import (
	"slices"
	"time"
)

const (
	Inflation = "inflation"
	Jobs      = "jobs"
	Rates     = "rates"
	Earnings  = "earnings"
	Coin      = "coin"
	Unlock    = "unlock"
	Expiry    = "expiry"
)

type Entry struct {
	ID     string    `json:"id"`
	Time   time.Time `json:"time"`
	Title  string    `json:"title"`
	Source string    `json:"source"`
	URL    string    `json:"url,omitempty"`
	Kind   string    `json:"kind,omitempty"`
	Assets []string  `json:"assets,omitempty"`
	AllDay bool      `json:"all_day,omitempty"`
	Rough  bool      `json:"approximate,omitempty"`
	Outlet string    `json:"outlet,omitempty"`
	Share  float64   `json:"share_of_supply,omitempty"`
}

func (e Entry) Coins() bool {
	return e.Kind == Coin || e.Kind == Unlock
}

func (e Entry) Of(kind string, assets ...string) Entry {
	e.Kind, e.Assets = kind, assets

	return e
}

func (e Entry) Repeats(other Entry) bool {
	return e.Coins() && other.Coins() && e.Time.Equal(other.Time) && slices.Equal(e.Assets, other.Assets)
}

func (e Entry) Clocked(by Entry) bool {
	sameCoin := e.Coins() && by.Coins() && slices.Equal(e.Assets, by.Assets)
	sameDay := !by.Time.Before(e.Time) && by.Time.Before(e.Time.AddDate(0, 0, 1))

	return e.AllDay && !by.AllDay && sameCoin && sameDay
}

func NewEntry(source string, at time.Time, title, url string) Entry {
	return Entry{
		ID:     source + ":" + at.UTC().Format(time.RFC3339) + ":" + title,
		Time:   at.UTC(),
		Title:  title,
		Source: source,
		URL:    url,
	}
}
