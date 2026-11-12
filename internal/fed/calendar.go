package fed

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/schedule"
)

const (
	calendarURL = "https://www.federalreserve.gov/json/calendar.json"
	pageURL     = "https://www.federalreserve.gov/monetarypolicy/fomccalendars.htm"
	sourceName  = "ФРС"
	decision    = "FOMC Meeting"
	committee   = "FOMC"
	zoneName    = "America/New_York"
)

var byteOrderMark = []byte("\xef\xbb\xbf")

type Calendar struct {
	http *httpclient.Client
	url  string
}

func NewCalendar(http *httpclient.Client) Calendar {
	return Calendar{http: http, url: calendarURL}
}

func (c Calendar) Name() string {
	return "fed"
}

type listing struct {
	Events []struct {
		Title string `json:"title"`
		Time  string `json:"time"`
		Month string `json:"month"`
		Days  string `json:"days"`
		Type  string `json:"type"`
	} `json:"events"`
}

func (c Calendar) Entries(ctx context.Context) ([]schedule.Entry, error) {
	body, err := c.http.Bytes(ctx, c.url)
	if err != nil {
		return nil, err
	}

	return parse(body)
}

func parse(body []byte) ([]schedule.Entry, error) {
	zone, err := time.LoadLocation(zoneName)
	if err != nil {
		return nil, err
	}

	var list listing

	if err := json.Unmarshal(bytes.TrimPrefix(body, byteOrderMark), &list); err != nil {
		return nil, err
	}

	out := []schedule.Entry{}

	for _, e := range list.Events {
		if e.Type != committee {
			continue
		}

		at, dated := moment(e.Month, e.Days, e.Time, zone)
		if !dated {
			continue
		}

		title := strings.TrimSpace(e.Title)
		entry := schedule.NewEntry(sourceName, at, title, pageURL)

		if title == decision {
			entry = entry.Of(schedule.Rates)
		}

		out = append(out, entry)
	}

	return out, nil
}

func moment(month, days, clock string, zone *time.Location) (time.Time, bool) {
	last := days[strings.LastIndex(days, "-")+1:]

	day, err := strconv.Atoi(strings.TrimSpace(last))
	if err != nil {
		return time.Time{}, false
	}

	start, err := time.ParseInLocation("2006-01", month, zone)
	if err != nil {
		return time.Time{}, false
	}

	hour := strings.NewReplacer("a.m.", "AM", "p.m.", "PM").Replace(strings.TrimSpace(clock))

	at, err := time.Parse("3:04 PM", hour)
	if err != nil {
		return time.Date(start.Year(), start.Month(), day, 0, 0, 0, 0, zone), true
	}

	return time.Date(start.Year(), start.Month(), day, at.Hour(), at.Minute(), 0, 0, zone), true
}
