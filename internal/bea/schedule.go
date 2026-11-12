package bea

import (
	"context"
	"strings"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/schedule"
)

const (
	scheduleURL = "https://www.bea.gov/news/schedule/ics/online-calendar-subscription.ics"
	pageURL     = "https://www.bea.gov/news/schedule"
	sourceName  = "BEA"
	stamp       = "20060102T150405Z"
)

var watched = []string{"Gross Domestic Product,", "GDP (", "Personal Income and Outlays"}

type Schedule struct {
	http *httpclient.Client
	url  string
}

func NewSchedule(http *httpclient.Client) Schedule {
	return Schedule{http: http, url: scheduleURL}
}

func (s Schedule) Name() string {
	return "bea"
}

func (s Schedule) Entries(ctx context.Context) ([]schedule.Entry, error) {
	body, err := s.http.Bytes(ctx, s.url)
	if err != nil {
		return nil, err
	}

	return parse(string(body)), nil
}

func parse(body string) []schedule.Entry {
	unfolded := strings.NewReplacer("\r\n ", "", "\n ", "", "\r\n", "\n").Replace(body)
	out := []schedule.Entry{}

	var title, start string

	for line := range strings.SplitSeq(unfolded, "\n") {
		name, value, _ := strings.Cut(line, ":")

		switch {
		case line == "BEGIN:VEVENT":
			title, start = "", ""
		case name == "SUMMARY":
			title = strings.NewReplacer(`\,`, ",", `\;`, ";").Replace(value)
		case strings.HasPrefix(name, "DTSTART"):
			start = value
		case line == "END:VEVENT":
			if at, err := time.Parse(stamp, start); err == nil && major(title) {
				out = append(out, schedule.NewEntry(sourceName, at, title, pageURL))
			}
		}
	}

	return out
}

func major(title string) bool {
	for _, prefix := range watched {
		if strings.HasPrefix(title, prefix) {
			return true
		}
	}

	return false
}
