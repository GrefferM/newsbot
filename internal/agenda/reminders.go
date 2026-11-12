package agenda

import (
	"context"
	"fmt"
	"slices"
	"time"

	"newsbot/internal/news"
	"newsbot/internal/schedule"
)

const (
	remindersName = "market-reminders"
	startedWithin = 2 * time.Hour
	clock         = "02-01-2006 15:04"
	dayOnly       = "02-01-2006"
)

type Dated interface {
	On(day string, zone *time.Location) []schedule.Entry
	Between(from, to time.Time) []schedule.Entry
}

type Reminders struct {
	dated Dated
	zone  *time.Location
	lead  time.Duration
	every time.Duration
}

func NewReminders(dated Dated, zone *time.Location, lead, every time.Duration) *Reminders {
	return &Reminders{dated: dated, zone: zone, lead: lead, every: every}
}

func (r *Reminders) Name() string {
	return remindersName
}

func (r *Reminders) Interval() time.Duration {
	return r.every
}

func (r *Reminders) Poll(context.Context) ([]news.Event, error) {
	now := time.Now()
	day := now.In(r.zone)

	events := Today(r.dated.On(day.Format(time.DateOnly), r.zone), r.zone, now)
	events = append(events, Soon(r.dated.Between(now, now.Add(r.lead)), r.zone, now, r.lead)...)

	return append(events, Started(r.dated.Between(now.Add(-startedWithin), now), r.zone, now)...), nil
}

func Today(entries []schedule.Entry, zone *time.Location, now time.Time) []news.Event {
	return told(slices.DeleteFunc(slices.Clone(entries), timed), zone, now, "today:", "сьогодні")
}

func Soon(entries []schedule.Entry, zone *time.Location, now time.Time, lead time.Duration) []news.Event {
	when := fmt.Sprintf("за %d хв", int(lead.Minutes()))
	events := told(slices.DeleteFunc(slices.Clone(entries), allDay), zone, now, "soon:", when)

	for i := range events {
		events[i].Severity = news.High
	}

	return events
}

func Started(entries []schedule.Entry, zone *time.Location, now time.Time) []news.Event {
	events := told(slices.DeleteFunc(slices.Clone(entries), allDay), zone, now, "started:", "зараз")

	for i := range events {
		events[i].Severity = news.High
	}

	return events
}

func told(entries []schedule.Entry, zone *time.Location, now time.Time, prefix, when string) []news.Event {
	events := make([]news.Event, 0, len(entries))

	for _, e := range entries {
		if e.Rough {
			continue
		}

		at := e.Time.In(zone).Format(clock)

		if e.AllDay {
			at = e.Time.In(zone).Format(dayOnly)
		}

		event := news.Event{
			ID:       prefix + e.ID,
			Source:   remindersName,
			Category: news.Reminder,
			Kind:     when + ", " + at,
			Title:    e.Source + ": " + e.Title,
			URL:      e.URL,
			Time:     now.UTC(),
			Severity: news.Normal,
		}

		for _, asset := range e.Assets {
			event.Symbols = append(event.Symbols, news.Symbol(asset))
		}

		if e.Outlet != "" {
			event.Details = []string{e.Outlet}
		}

		events = append(events, event)
	}

	return events
}

func timed(e schedule.Entry) bool {
	return !e.AllDay
}

func allDay(e schedule.Entry) bool {
	return e.AllDay
}
