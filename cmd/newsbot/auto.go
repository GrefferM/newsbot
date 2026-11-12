package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"newsbot/internal/agenda"
	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/autotrade"
	"newsbot/internal/filestore"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
)

const (
	autoEvery     = 30 * time.Second
	autoPlanLife  = 48 * time.Hour
	eventQuestion = "Склади плани входу під подію календаря: «%s», %s. Побажання власника: автотрейдинг — на стеження " +
		"стане лише найобережніший план, і суддя перевірить його один раз. Якщо входу з перевагою не бачиш — не " +
		"складай жодного плану"
)

type autoPlanner struct {
	team *agents.Team
}

func (p autoPlanner) Event(ctx context.Context, asset string, entry schedule.Entry) (research.Plan, error) {
	question := fmt.Sprintf(eventQuestion, entry.Title, entry.Time.UTC().Format("02.01 15:04 UTC"))

	answer, err := p.team.Ask(ctx, riskAgent, asset, question)
	if err != nil {
		return research.Plan{}, err
	}

	return research.Plan{
		Wish:      "автотрейдинг, подія: " + entry.Title,
		Summary:   answer.Text,
		Scenarios: answer.Scenarios,
		Made:      time.Now().UTC(),
	}, nil
}

type autoWatcher struct {
	pilot *autopilot.Service
}

func (w autoWatcher) Arm(asset string, plan research.Scenario) error {
	_, err := w.pilot.ArmFor(asset, plan, autoPlanLife)

	return err
}

func buildAuto(
	cfg config, keeper *research.Keeper, pilot *autopilot.Service, calendar *agenda.Service, team *agents.Team,
	log *slog.Logger,
) (*autotrade.Service, error) {
	file := filestore.NewJSONFile[autotrade.State](cfg.AutoPath)

	saved, err := file.Load(autotrade.State{})
	if err != nil {
		return nil, fmt.Errorf("load auto trading state: %w", err)
	}

	return autotrade.NewService(autotrade.Deps{
		Planner: autoPlanner{team: team},
		Shelf:   keeper,
		Watcher: autoWatcher{pilot: pilot},
		Agenda:  calendar,
		Store:   file,
		Lead:    cfg.ReminderLead,
		Every:   autoEvery,
		AtOnce:  agentsAtOnce,
		Log:     log,
	}, saved), nil
}
