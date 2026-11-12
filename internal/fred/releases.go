package fred

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
	datesURL   = "https://api.stlouisfed.org/fred/release/dates"
	pageURL    = "https://fred.stlouisfed.org/releases/calendar"
	sourceName = "США"
	zoneName   = "America/New_York"
	hour       = 8
	minute     = 30
	hidden     = "***"
)

type release struct {
	id    int
	title string
	kind  string
}

var watched = []release{
	{10, "CPI — інфляція США", schedule.Inflation},
	{50, "Employment Situation — ринок праці США", schedule.Jobs},
	{46, "PPI — ціни виробників США", ""},
}

type Releases struct {
	http *httpclient.Client
	url  string
	key  string
}

func NewReleases(http *httpclient.Client, key string) Releases {
	return Releases{http: http, url: datesURL, key: key}
}

func (r Releases) Name() string {
	return "fred"
}

type dates struct {
	Dates []struct {
		Date string `json:"date"`
	} `json:"release_dates"`
}

func (r Releases) Entries(ctx context.Context) ([]schedule.Entry, error) {
	zone, err := time.LoadLocation(zoneName)
	if err != nil {
		return nil, err
	}

	today := time.Now().In(zone).Format(time.DateOnly)
	out := []schedule.Entry{}

	for _, w := range watched {
		query := url.Values{
			"release_id":                         {fmt.Sprint(w.id)},
			"file_type":                          {"json"},
			"include_release_dates_with_no_data": {"true"},
			"realtime_start":                     {today},
			"realtime_end":                       {"9999-12-31"},
			"sort_order":                         {"asc"},
			"api_key":                            {r.key},
		}

		var list dates

		if err := r.http.JSON(ctx, r.url+"?"+query.Encode(), &list); err != nil {
			return nil, errors.New(strings.ReplaceAll(err.Error(), r.key, hidden))
		}

		for _, d := range list.Dates {
			if at, dated := moment(d.Date, zone); dated {
				out = append(out, schedule.NewEntry(sourceName, at, w.title, pageURL).Of(w.kind))
			}
		}
	}

	return out, nil
}

func moment(day string, zone *time.Location) (time.Time, bool) {
	at, err := time.ParseInLocation(time.DateOnly, day, zone)
	if err != nil {
		return time.Time{}, false
	}

	return at.Add(hour*time.Hour + minute*time.Minute), true
}
