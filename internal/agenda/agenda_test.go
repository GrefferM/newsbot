package agenda

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"newsbot/internal/schedule"
)

type fixed struct {
	name    string
	entries []schedule.Entry
	err     error
}

func (f fixed) Name() string {
	return f.name
}

func (f fixed) Entries(context.Context) ([]schedule.Entry, error) {
	return f.entries, f.err
}

type health struct {
	ok     int
	failed error
}

func (h *health) OK(_ string, seen, _ int) {
	h.ok = seen
}

func (h *health) Fail(_ string, err error) {
	h.failed = err
}

func TestUpcomingIsSortedAndAFailedSourceKeepsWhatItHad(t *testing.T) {
	now := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	fomc := schedule.NewEntry("ФРС", now.Add(48*time.Hour), "FOMC Meeting", "")
	gdp := schedule.NewEntry("BEA", now.Add(24*time.Hour), "GDP", "")
	far := schedule.NewEntry("BEA", now.Add(900*time.Hour), "GDP", "")
	past := schedule.NewEntry("BEA", now.Add(-time.Hour), "GDP", "")

	state := &health{}
	fed := &fixed{name: "fed", entries: []schedule.Entry{fomc}}
	service := NewService(Deps{
		Sources: []Source{sourceOf(fed), fixed{name: "bea", entries: []schedule.Entry{far, gdp, past}}},
		Health:  state,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	service.Refresh(context.Background())

	got := service.Upcoming(now, 72*time.Hour)
	if len(got) != 2 || got[0].Title != "GDP" || got[1].Title != "FOMC Meeting" || state.ok != 4 {
		t.Fatalf("upcoming %+v, health %+v", got, state)
	}

	fed.entries, fed.err = nil, errors.New("HTTP 500")

	service.Refresh(context.Background())

	if len(service.Upcoming(now, 72*time.Hour)) != 2 || state.failed == nil {
		t.Fatalf("after a failure: %+v, health %+v", service.Upcoming(now, 72*time.Hour), state)
	}
}

type pointer struct{ *fixed }

func sourceOf(f *fixed) Source {
	return pointer{f}
}

func TestATimedEventIsToldAheadAndWhenItStartsInTheZone(t *testing.T) {
	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 12, 9, 19, 0, 0, 0, time.UTC)
	service := NewService(Deps{
		Sources: []Source{fixed{name: "fed", entries: []schedule.Entry{schedule.NewEntry("ФРС", at, "FOMC Meeting", "https://x")}}},
		Health:  &health{},
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	service.Refresh(context.Background())

	soon := Soon(service.Between(at.Add(-10*time.Minute), at), kyiv, at.Add(-10*time.Minute), 10*time.Minute)
	if len(soon) != 1 || soon[0].Kind != "за 10 хв, 09-12-2026 21:00" || soon[0].Title != "ФРС: FOMC Meeting" {
		t.Fatalf("soon %+v", soon)
	}

	started := Started(service.Between(at.Add(-time.Hour), at.Add(time.Minute)), kyiv, at.Add(time.Minute))
	if len(started) != 1 || started[0].Kind != "зараз, 09-12-2026 21:00" || started[0].ID != "started:"+strings.TrimPrefix(soon[0].ID, "soon:") {
		t.Fatalf("started %+v", started)
	}

	if today := Today(service.On("2026-12-09", kyiv), kyiv, at); len(today) != 0 {
		t.Fatalf("a timed event is not told as an all-day one: %+v", today)
	}

	if len(service.On("2026-12-10", kyiv)) != 0 {
		t.Fatal("the next day must be empty")
	}
}

type kept []schedule.Entry

func (k kept) Dates(*time.Location) []schedule.Entry {
	return k
}

func TestACoinsDateIsRemindedOnItsDayAndAnUnlockIsNotListedTwice(t *testing.T) {
	kyiv, _ := time.LoadLocation("Europe/Kyiv")
	vote := schedule.Entry{ID: "vote:2026-11-16", Time: time.Date(2026, 11, 15, 22, 0, 0, 0, time.UTC), Title: "Голосування", Source: "ZEC",
		Kind: schedule.Coin, Assets: []string{"ZEC"}, AllDay: true, Outlet: "Forum"}
	rough := schedule.Entry{ID: "nu8:2026-11-16", Time: vote.Time, Title: "NU8", Source: "ZEC", Kind: schedule.Coin, Assets: []string{"ZEC"}, AllDay: true, Rough: true}
	unlock := schedule.Entry{ID: "unlock:AGT", Time: time.Date(2026, 11, 16, 20, 43, 0, 0, time.UTC), Title: "Розблокування", Source: "AGT",
		Kind: schedule.Coin, Assets: []string{"AGT"}}
	told := unlock
	told.ID, told.Title = "f1:2026-11-16", "Розблокування з новини"

	service := NewService(Deps{
		Sources: []Source{fixed{name: "unlocks", entries: []schedule.Entry{unlock}}},
		Coins:   kept{vote, rough, told},
		Health:  &health{},
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	service.Refresh(context.Background())

	if got := service.On("2026-11-16", kyiv); len(got) != 3 || got[2].ID != "unlock:AGT" {
		t.Fatalf("the digest's unlock stands for the same one told by the news: %+v", got)
	}

	today := Today(service.On("2026-11-16", kyiv), kyiv, vote.Time.Add(6*time.Hour))
	if len(today) != 1 || today[0].ID != "today:vote:2026-11-16" || today[0].Kind != "сьогодні, 16-11-2026" || today[0].Title != "ZEC: Голосування" {
		t.Fatalf("an all-day date is told on its day, a timed one when it starts: %+v", today)
	}

	if today[0].Symbols[0] != "ZEC" || today[0].Details[0] != "Forum" {
		t.Errorf("a coin's reminder names the coin and the outlet: %+v", today[0])
	}

	halt := vote
	halt.ID, halt.AllDay, halt.Time = "halt:2026-11-16", false, vote.Time.Add(11*time.Hour)
	clocked := NewService(Deps{Coins: kept{vote, halt}, Health: &health{}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})

	if got := clocked.On("2026-11-16", kyiv); len(got) != 1 || got[0].ID != halt.ID {
		t.Errorf("a coin's date without an hour gives way to the same day's date with one: %+v", got)
	}

	if started := Started(service.Between(vote.Time, unlock.Time), kyiv, unlock.Time); len(started) != 1 || started[0].ID != "started:unlock:AGT" {
		t.Errorf("started %+v", started)
	}
}
