package main

import (
	"fmt"
	"log/slog"
	"time"

	"newsbot/internal/agenda"
	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/autotrade"
	"newsbot/internal/binance"
	"newsbot/internal/chart"
	"newsbot/internal/coingecko"
	"newsbot/internal/filestore"
	"newsbot/internal/httpclient"
	"newsbot/internal/monitor"
	"newsbot/internal/radar"
	"newsbot/internal/research"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

type desk struct {
	research *research.Keeper
	pilot    *autopilot.Service
	auto     *autotrade.Service
	agents   *agents.Team
}

func warn(log *slog.Logger, message string) func(err error) {
	return func(err error) {
		log.Warn(message, "err", err)
	}
}

func buildDispatcher(
	cfg config,
	f flags,
	futures *binance.Futures,
	journal *filestore.JournalFile,
	recent *monitor.Recent,
	status *monitor.Status,
	log *slog.Logger,
) *monitor.Dispatcher {
	return monitor.NewDispatcher(monitor.DispatcherDeps{
		Publisher:   publisher(cfg, f, log),
		Journal:     journal,
		Illustrator: chart.NewIllustrator(futures),
		Enricher:    monitor.NewEnricher(futures, log),
		Recent:      recent,
		Status:      status,
		Spacing:     publishSpacing,
		Log:         log,
	})
}

func openWatchlist(
	cfg config,
	client *httpclient.Client,
	futures *binance.Futures,
	journal *filestore.JournalFile,
	status *monitor.Status,
	log *slog.Logger,
) (*watchlist.Service, error) {
	listFile := filestore.NewJSONFile[watchlist.List](cfg.WatchlistPath)

	kept, err := listFile.Load(watchlist.List{})
	if err != nil {
		return nil, fmt.Errorf("load watchlist: %w", err)
	}

	scanFile := filestore.NewJSONFile[watchlist.Scan](cfg.ScanPath)

	lastScan, err := scanFile.Load(watchlist.Scan{})
	if err != nil {
		return nil, fmt.Errorf("load scan: %w", err)
	}

	return watchlist.NewService(watchlist.Deps{
		Universe:     binance.NewUniverse(client, futures),
		OpenInterest: futures,
		Depth:        futures,
		Store:        listFile,
		Scans:        scanFile,
		Rule:         cfg.Watchlist,
		Every:        cfg.WatchlistEvery,
		Journal:      journal,
		Status:       status,
		Log:          log,
	}, kept, lastScan), nil
}

func openTrader(
	cfg config,
	futures *binance.Futures,
	journal *filestore.JournalFile,
	log *slog.Logger,
) (*trading.Trader, error) {
	bookFile := filestore.NewJSONFile[trading.Book](cfg.AccountPath)

	book, err := bookFile.Load(trading.Book{})
	if err != nil {
		return nil, fmt.Errorf("load the account: %w", err)
	}

	return trading.NewTrader(trading.TraderDeps{
		Market:   futures,
		Broker:   buildBroker(cfg),
		Store:    bookFile,
		Strategy: cfg.Strategy,
		Limits:   cfg.Limits,
		Capital:  cfg.Capital,
		Every:    cfg.AccountEvery,
		Journal:  journal,
		Log:      log,
	}, book), nil
}

func openResearch(cfg config) (*research.Keeper, error) {
	researchFile := filestore.NewJSONFile[research.Archive](cfg.ResearchPath)

	archive, err := researchFile.Load(research.Archive{})
	if err != nil {
		return nil, fmt.Errorf("load research: %w", err)
	}

	return research.NewKeeper(researchFile, archive), nil
}

func openDesk(
	cfg config,
	client *httpclient.Client,
	futures *binance.Futures,
	recent *monitor.Recent,
	trader *trading.Trader,
	list *watchlist.Service,
	keeper *research.Keeper,
	pulse *radar.Service,
	calendar *agenda.Service,
	log *slog.Logger,
) (desk, error) {
	watchesFile := filestore.NewJSONFile[autopilot.Watches](cfg.WatchesPath)

	watches, err := watchesFile.Load(autopilot.Watches{})
	if err != nil {
		return desk{}, fmt.Errorf("load plan watches: %w", err)
	}

	judge := &planJudge{}

	pilot := autopilot.NewService(autopilot.Deps{
		Prices:   futures,
		Events:   recent,
		Judge:    judge,
		Desk:     trader,
		Store:    watchesFile,
		Every:    planWatchEvery,
		Lifetime: time.Duration(cfg.WatchDays) * 24 * time.Hour,
		Rest:     planCheckRest,
		Hold:     cfg.Strategy.Hold,
		AtOnce:   agentsAtOnce,
		Log:      log,
	}, watches)

	team := buildAgents(agentDeps{
		cfg:       cfg,
		client:    client,
		futures:   futures,
		recent:    recent,
		trader:    trader,
		watchlist: list,
		research:  keeper,
		pilot:     pilot,
		radar:     pulse,
		agenda:    calendar,
		links:     coingecko.NewDirectory(client),
		log:       filestore.NewJournal(cfg.AgentLogPath, warn(log, "agent log write failed")),
	})

	judge.team = team

	auto, err := buildAuto(cfg, keeper, pilot, calendar, team, log)
	if err != nil {
		return desk{}, err
	}

	return desk{research: keeper, pilot: pilot, agents: team, auto: auto}, nil
}

func keptContracts(list *watchlist.Service) func() []string {
	return func() []string {
		var out []string

		for _, kept := range list.Kept() {
			if kept.Coin.Contract != "" {
				out = append(out, kept.Coin.Contract)
			}
		}

		return out
	}
}
