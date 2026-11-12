package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"newsbot/internal/exchange"
	"newsbot/internal/incident"
	"newsbot/internal/radar"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

type config struct {
	TelegramToken  string
	TelegramChatID string
	BinanceKey     string
	BinanceSecret  string
	TestnetKey     string
	TestnetSecret  string
	ModelURL       string
	ModelKey       string
	Model          string
	DryRun         bool
	StatePath      string
	JournalPath    string
	AgentLogPath   string
	HTTPAddr       string
	ChartDir       string
	Disabled       map[string]bool
	Announcements  exchange.AnnouncementPolicy
	Hacks          incident.HackPolicy
	Depeg          incident.DepegRules
	Watchlist      watchlist.Rule
	WatchlistEvery time.Duration
	WatchlistPath  string
	ScanPath       string
	ResearchPath   string
	WatchesPath    string
	EventsPath     string
	ReminderLead   time.Duration
	AutoPath       string
	ReadsPath      string
	Python         string
	UnlocksScript  string
	Radar          radar.Rule
	WatchDays      int
	SearchKey      string
	FredKey        string
	FinnhubKey     string
	Searches       int
	PageBudget     int
	XToken         string
	XBudget        int
	AgentMemory    int
	Finalists      int
	AccountPath    string
	Capital        float64
	Strategy       trading.Strategy
	Limits         trading.Limits
	AccountEvery   time.Duration
}

func loadConfig() config {
	return config{
		TelegramToken:  os.Getenv("TG_BOT_TOKEN"),
		TelegramChatID: os.Getenv("TG_CHAT_ID"),
		BinanceKey:     os.Getenv("BINANCE_API_KEY"),
		BinanceSecret:  os.Getenv("BINANCE_API_SECRET"),
		TestnetKey:     os.Getenv("BINANCE_TESTNET_KEY"),
		TestnetSecret:  os.Getenv("BINANCE_TESTNET_SECRET"),
		ModelURL:       envString("MODEL_URL", "https://opencode.ai/zen/go/v1"),
		ModelKey:       os.Getenv("MODEL_KEY"),
		Model:          envString("MODEL_NAME", "deepseek-v4.1-flash"),
		SearchKey:      os.Getenv("TAVILY_KEY"),
		FredKey:        os.Getenv("FRED_KEY"),
		FinnhubKey:     os.Getenv("FINNHUB_KEY"),
		Searches:       int(envFloat("OSINTER_SEARCHES_PER_QUESTION", 3)),
		PageBudget:     int(envFloat("PAGES_PER_QUESTION", 25)),
		XToken:         os.Getenv("X_BEARER_TOKEN"),
		XBudget:        int(envFloat("X_READS_PER_QUESTION", 1)),
		AgentMemory:    int(envFloat("AGENT_MEMORY_EXCHANGES", 5)),
		Finalists:      int(envFloat("FUNNEL_FINALISTS", 3)),
		DryRun:         os.Getenv("DRY_RUN") == "1",
		StatePath:      envString("NEWSBOT_STATE", "data/state.json"),
		JournalPath:    envString("NEWSBOT_JOURNAL", "data/journal.jsonl"),
		AgentLogPath:   envString("NEWSBOT_AGENT_LOG", "data/agents.jsonl"),
		HTTPAddr:       envString("NEWSBOT_ADDR", "127.0.0.1:8080"),
		ChartDir:       os.Getenv("NEWSBOT_CHART_DIR"),
		Disabled:       envSet("NEWSBOT_DISABLE"),
		Announcements: exchange.AnnouncementPolicy{
			MaxAge: time.Duration(envFloat("ANNOUNCEMENT_MAX_AGE_MINUTES", 10) * float64(time.Minute)),
		},
		Hacks: incident.HackPolicy{
			MinAmount: envFloat("HACK_MIN_USD", 5e6),
			MaxAge:    72 * time.Hour,
		},
		Depeg: incident.DepegRules{
			MinSupply: 300e6,
			Threshold: envFloat("DEPEG_THRESHOLD", 0.01),
			Dead:      0.5,
			Cooldown:  6 * time.Hour,
		},
		Watchlist: watchlist.Rule{
			LowVolume:    300e3,
			ExcludedTags: []string{"stablecoin", "bStocks", "tCommodities"},
		},
		WatchlistEvery: 6 * time.Hour,
		WatchlistPath:  envString("NEWSBOT_WATCHLIST", "data/watchlist.json"),
		ScanPath:       envString("NEWSBOT_SCAN", "data/scan.json"),
		ResearchPath:   envString("NEWSBOT_RESEARCH", "data/research.json"),
		WatchesPath:    envString("NEWSBOT_WATCHES", "data/watches.json"),
		EventsPath:     envString("NEWSBOT_EVENTS", "data/events.json"),
		ReminderLead:   time.Duration(envFloat("REMINDER_LEAD_MINUTES", 10) * float64(time.Minute)),
		AutoPath:       envString("NEWSBOT_AUTO", "data/auto.json"),
		ReadsPath:      envString("NEWSBOT_READS", "data/reads.json"),
		Python:         envString("NEWSBOT_PYTHON", "python3"),
		UnlocksScript:  envString("NEWSBOT_UNLOCKS_SCRIPT", ""),
		Radar: radar.Rule{
			Interest:    envFloat("PULSE_INTEREST_SHIFT", 0.02),
			Longs:       envFloat("PULSE_LONGS_SHIFT", 0.02),
			LaunchTimes: envFloat("PULSE_LAUNCH_TIMES", 10),
			LaunchMove:  envFloat("PULSE_LAUNCH_MOVE", 0.05),
		},
		WatchDays:   int(envFloat("PLAN_WATCH_DAYS", 7)),
		AccountPath: envString("NEWSBOT_ACCOUNT", "data/account.json"),
		Capital:     max(envFloat("ACCOUNT_CAPITAL", 0), 0),
		Strategy:    exits(),
		Limits: trading.Limits{
			MaxExposure:    1,
			MinVolume:      1e6,
			CrowdedFunding: -0.0005,
			DailyLoss:      0.10,
		},
		AccountEvery: time.Minute,
	}
}

func exits() trading.Strategy {
	return trading.Strategy{
		StopLoss:     0.30,
		TakeProfit:   0.30,
		Hold:         24 * time.Hour,
		EstimatedFee: 0.002,
	}
}

func (c config) telegramEnabled() bool {
	return c.TelegramToken != "" && c.TelegramChatID != "" && !c.DryRun
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return v
	}

	return fallback
}

func envSet(key string) map[string]bool {
	set := map[string]bool{}

	for _, name := range strings.Split(os.Getenv(key), ",") {
		if name = strings.TrimSpace(name); name != "" {
			set[name] = true
		}
	}

	return set
}
