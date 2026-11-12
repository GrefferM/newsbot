package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"
	_ "time/tzdata"

	"newsbot/internal/binance"
	"newsbot/internal/console"
	"newsbot/internal/dotenv"
	"newsbot/internal/filestore"
	"newsbot/internal/httpapi"
	"newsbot/internal/httpclient"
	"newsbot/internal/monitor"
	"newsbot/internal/news"
	"newsbot/internal/telegram"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

const (
	publishSpacing   = 3 * time.Second
	recentCapacity   = 200
	shutdownDeadline = 5 * time.Second
	readHeaderLimit  = 5 * time.Second
	planWatchEvery   = 30 * time.Second
	planCheckRest    = 15 * time.Minute
	agentsAtOnce     = 5
)

type flags struct {
	once         int
	send         bool
	envFile      string
	checkTestnet string
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	f := parseFlags()

	if err := loadEnv(log, f.envFile); err != nil {
		return err
	}

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if f.checkTestnet != "" {
		return checkTestnet(ctx, cfg, f.checkTestnet, log)
	}

	base, err := openBase(cfg, f, log)
	if err != nil {
		return err
	}

	if f.once > 0 {
		base.service.Once(ctx, base.sources, f.once)

		return nil
	}

	return base.serve(ctx)
}

type base struct {
	cfg     config
	log     *slog.Logger
	client  *httpclient.Client
	futures *binance.Futures
	status  *monitor.Status
	recent  *monitor.Recent
	list    *watchlist.Service
	trader  *trading.Trader
	service *monitor.Service
	sources []monitor.Source
}

func openBase(cfg config, f flags, log *slog.Logger) (base, error) {
	seen, err := filestore.Open(cfg.StatePath)
	if err != nil {
		return base{}, fmt.Errorf("load state: %w", err)
	}

	client := httpclient.New()
	futures := binance.NewFutures(client, tickerMaxAge)
	sources := buildSources(cfg, client)

	if len(cfg.Disabled) > 0 {
		log.Info("sources disabled", "names", slices.Sorted(maps.Keys(cfg.Disabled)))
	}

	status := monitor.NewStatus(time.Now().UTC())

	recent, err := openRecent(cfg)
	if err != nil {
		return base{}, err
	}

	journal := filestore.NewJournal(cfg.JournalPath, warn(log, "journal write failed"))

	list, err := openWatchlist(cfg, client, futures, journal, status, log)
	if err != nil {
		return base{}, err
	}

	futures.Follow(keptContracts(list))

	trader, err := openTrader(cfg, futures, journal, log)
	if err != nil {
		return base{}, err
	}

	service := monitor.NewService(monitor.ServiceDeps{
		Collector:  monitor.NewCollector(seen, journal, status, log),
		Dispatcher: buildDispatcher(cfg, f, futures, journal, recent, status, log),
		Watchlist:  scanning(cfg, list),
		Trader:     trader,
		Seen:       seen,
		Log:        log,
	})

	return base{
		cfg: cfg, log: log, client: client, futures: futures, status: status, recent: recent,
		list: list, trader: trader, service: service, sources: sources,
	}, nil
}

func openRecent(cfg config) (*monitor.Recent, error) {
	eventsFile := filestore.NewJSONFile[[]news.Event](cfg.EventsPath)

	saved, err := eventsFile.Load(nil)
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}

	return monitor.NewRecent(recentCapacity, eventsFile, saved), nil
}

func (b base) serve(ctx context.Context) error {
	keeper, err := openResearch(b.cfg)
	if err != nil {
		return err
	}

	pulse := buildRadar(b.cfg, b.futures, b.status, keeper, b.log)

	reads := filestore.NewReadsFile(b.cfg.ReadsPath)
	calendar := buildAgenda(b.cfg, b.client, b.futures, keeper, reads, b.status, b.log)

	desk, err := openDesk(b.cfg, b.client, b.futures, b.recent, b.trader, b.list, keeper, pulse, calendar, b.log)
	if err != nil {
		return err
	}

	sources := withReminders(b.cfg, b.sources, calendar)

	server := startHTTP(b.cfg.HTTPAddr, httpapi.NewHandler(httpapi.HandlerDeps{
		Status:    b.status,
		Recent:    b.recent,
		Watchlist: b.list,
		Live:      b.futures,
		Trader:    b.trader,
		Agents:    desk.agents,
		Research:  desk.research,
		Autopilot: desk.pilot,
		Auto:      desk.auto,
		Radar:     pulse,
		Agenda:    calendar,
	}), b.log)

	reporters := []monitor.Reporter{desk.pilot, calendar, desk.auto}

	if !b.cfg.Disabled[pulse.Name()] {
		reporters = append(reporters, pulse)
	}

	b.service.Run(ctx, sources, reporters...)

	return shutdown(server)
}

func scanning(cfg config, list *watchlist.Service) monitor.Runner {
	if cfg.Disabled[list.Name()] {
		return nil
	}

	return list
}

func buildBroker(cfg config) trading.Broker {
	if cfg.TestnetKey == "" || cfg.TestnetSecret == "" {
		return trading.Idle{}
	}

	return binance.NewTestnet(cfg.TestnetKey, cfg.TestnetSecret)
}

func parseFlags() flags {
	var f flags

	flag.IntVar(&f.once, "once", 0, "poll every source once, publish up to N items per source, keep state untouched")
	flag.BoolVar(&f.send, "send", false, "with -once: send to Telegram instead of printing")
	flag.StringVar(&f.envFile, "env", ".env", "file with KEY=VALUE settings; real environment wins")
	flag.StringVar(&f.checkTestnet, "check-testnet", "", "short a small amount of this testnet contract, protect it, then close it")
	flag.Parse()

	return f
}

func loadEnv(log *slog.Logger, path string) error {
	n, err := dotenv.Load(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	if n > 0 {
		log.Info("loaded settings", "file", path, "vars", n)
	}

	return nil
}

func publisher(cfg config, f flags, log *slog.Logger) monitor.Publisher {
	if cfg.telegramEnabled() && (f.once == 0 || f.send) {
		log.Info("telegram sink enabled", "chat", cfg.TelegramChatID)

		return telegram.NewPublisher(cfg.TelegramToken, cfg.TelegramChatID)
	}

	log.Warn("TG_BOT_TOKEN/TG_CHAT_ID not set or DRY_RUN=1: printing messages to stdout")

	return console.NewPublisher(os.Stdout, cfg.ChartDir)
}

func startHTTP(addr string, handler http.Handler, log *slog.Logger) *http.Server {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderLimit,
	}

	go func() {
		log.Info("http listening", "addr", addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
		}
	}()

	return server
}

func shutdown(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownDeadline)
	defer cancel()

	if err := server.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return server.Close()
}
