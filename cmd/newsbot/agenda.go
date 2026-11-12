package main

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"newsbot/internal/agenda"
	"newsbot/internal/bea"
	"newsbot/internal/binance"
	"newsbot/internal/deribit"
	"newsbot/internal/fed"
	"newsbot/internal/filestore"
	"newsbot/internal/finnhub"
	"newsbot/internal/fred"
	"newsbot/internal/httpclient"
	"newsbot/internal/monitor"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
	"newsbot/internal/unlocks"
)

const (
	agendaEvery        = 6 * time.Hour
	marketReminderPoll = time.Minute
	unlocksEvery       = 23 * time.Hour
	unlockTableShare   = 0.01
)

func buildAgenda(
	cfg config,
	client *httpclient.Client,
	futures *binance.Futures,
	keeper *research.Keeper,
	reads *filestore.ReadsFile,
	status *monitor.Status,
	log *slog.Logger,
) *agenda.Service {
	sources := []agenda.Source{
		fed.NewCalendar(client),
		bea.NewSchedule(client),
		deribit.NewExpiries(client),
	}

	if cfg.FredKey != "" {
		sources = append(sources, fred.NewReleases(client, cfg.FredKey))
	}

	if cfg.FinnhubKey != "" {
		sources = append(sources, earnings{reports: finnhub.NewEarnings(client, cfg.FinnhubKey), futures: futures})
	}

	if cfg.UnlocksScript != "" {
		sources = append(sources, unlocks.NewScript(unlocks.ScriptDeps{
			Python: cfg.Python,
			Path:   cfg.UnlocksScript,
			Listed: futures,
			Every:  unlocksEvery,
			Floor:  unlockTableShare,
			Memory: reads,
		}))
	}

	return agenda.NewService(agenda.Deps{
		Sources: slices.DeleteFunc(sources, func(s agenda.Source) bool {
			return cfg.Disabled[s.Name()]
		}),
		Coins:  keeper,
		Health: status,
		Every:  agendaEvery,
		Log:    log,
	})
}

func withReminders(cfg config, sources []monitor.Source, calendar *agenda.Service) []monitor.Source {
	reminders := agenda.NewReminders(calendar, news.Kyiv, cfg.ReminderLead, marketReminderPoll)

	if cfg.Disabled[reminders.Name()] {
		return sources
	}

	return append(sources, reminders)
}

type earnings struct {
	reports finnhub.Earnings
	futures *binance.Futures
}

func (e earnings) Name() string {
	return "finnhub"
}

func (e earnings) Entries(ctx context.Context) ([]schedule.Entry, error) {
	tracked, err := e.futures.Equities(ctx)
	if err != nil {
		return nil, err
	}

	return e.reports.Entries(ctx, tracked, time.Now())
}
