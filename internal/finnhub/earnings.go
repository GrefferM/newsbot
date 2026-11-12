package finnhub

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/schedule"
)

const (
	earningsURL = "https://finnhub.io/api/v1/calendar/earnings"
	pageURL     = "https://www.nasdaq.com/market-activity/stocks/%s/earnings"
	sourceName  = "Звіти"
	zoneName    = "America/New_York"
	hidden      = "***"
	ahead       = 45 * 24 * time.Hour
)

var hours = map[string]struct {
	hour, minute int
	said         string
}{
	"bmo": {7, 0, "до відкриття біржі"},
	"amc": {16, 0, "після закриття біржі"},
	"dmh": {12, 0, "під час торгів"},
}

type Earnings struct {
	http *httpclient.Client
	url  string
	key  string
}

func NewEarnings(http *httpclient.Client, key string) Earnings {
	return Earnings{http: http, url: earningsURL, key: key}
}

type calendar struct {
	Reports []struct {
		Symbol  string `json:"symbol"`
		Date    string `json:"date"`
		Hour    string `json:"hour"`
		Quarter int    `json:"quarter"`
	} `json:"earningsCalendar"`
}

func (e Earnings) Entries(ctx context.Context, symbols map[string]bool, now time.Time) ([]schedule.Entry, error) {
	zone, err := time.LoadLocation(zoneName)
	if err != nil {
		return nil, err
	}

	query := url.Values{
		"from":  {now.In(zone).Format(time.DateOnly)},
		"to":    {now.Add(ahead).In(zone).Format(time.DateOnly)},
		"token": {e.key},
	}

	var found calendar

	if err := e.http.JSON(ctx, e.url+"?"+query.Encode(), &found); err != nil {
		return nil, errors.New(strings.ReplaceAll(err.Error(), e.key, hidden))
	}

	out := []schedule.Entry{}

	for _, r := range found.Reports {
		if !symbols[r.Symbol] {
			continue
		}

		day, err := time.ParseInLocation(time.DateOnly, r.Date, zone)
		if err != nil {
			continue
		}

		title := r.Symbol + ": квартальний звіт"

		if h, known := hours[r.Hour]; known {
			day = day.Add(time.Duration(h.hour)*time.Hour + time.Duration(h.minute)*time.Minute)
			title += ", " + h.said
		}

		entry := schedule.NewEntry(sourceName, day, title, fmt.Sprintf(pageURL, strings.ToLower(r.Symbol)))

		out = append(out, entry.Of(schedule.Earnings, r.Symbol))
	}

	return out, nil
}
