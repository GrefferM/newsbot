package main

import (
	"newsbot/internal/agenda"
	"newsbot/internal/agents"
	"newsbot/internal/agents/prompts"
	"newsbot/internal/agents/tools"
	"newsbot/internal/autopilot"
	"newsbot/internal/binance"
	"newsbot/internal/coingecko"
	"newsbot/internal/googlenews"
	"newsbot/internal/httpclient"
	"newsbot/internal/llm"
	"newsbot/internal/monitor"
	"newsbot/internal/radar"
	"newsbot/internal/research"
	"newsbot/internal/tavily"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
	"newsbot/internal/web"
	"newsbot/internal/xapi"
)

const (
	analystAgent    = "binance"
	researcherAgent = "osinter"
	riskAgent       = "risk"
	traderAgent     = "trader"
	guardAgent      = "guard"
	quickIdeas      = 5
	quickRounds     = 4
)

type agentDeps struct {
	cfg       config
	client    *httpclient.Client
	futures   *binance.Futures
	recent    *monitor.Recent
	agenda    *agenda.Service
	trader    *trading.Trader
	watchlist *watchlist.Service
	research  *research.Keeper
	pilot     *autopilot.Service
	radar     *radar.Service
	log       agents.Log
	links     *coingecko.Directory
}

type toolbox struct {
	server  tools.ServerTool
	read    tools.ReadTool
	notices tools.NoticesTool
	pulse   tools.PulseTool
	screen  tools.ScreenTool
	report  tools.NewsTool
	osint   []agents.Tool
	brief   agents.Tool
	past    tools.OutcomesTool
	entry   tools.EntryTool
	sheet   tools.SheetTool
}

func newToolbox(d agentDeps) toolbox {
	rules := tools.DeskRules{
		Limits:    d.cfg.Limits,
		Hold:      d.cfg.Strategy.Hold,
		WatchDays: d.cfg.WatchDays,
		Analyst:   analystAgent,
	}
	public := binance.NewPublic(d.client)

	box := toolbox{
		server: tools.ServerTool{
			Recent:    d.recent,
			Trader:    d.trader,
			Watchlist: d.watchlist,
			Research:  d.research,
			Plans:     d.research,
			Watched:   d.pilot,
			Live:      d.futures,
			Radar:     d.radar,
			Agenda:    d.agenda,
			Journal:   d.log,
			Rules:     rules,
		},
		sheet: tools.SheetTool{Market: public},
		entry: tools.EntryTool{
			Research: d.research,
			Market:   public,
			Live:     d.futures,
			Watched:  d.pilot,
			Trader:   d.trader,
			Recent:   d.recent,
			Agenda:   d.agenda,
			Rules:    rules,
		},
		past:    tools.OutcomesTool{Watched: d.pilot, Trader: d.trader},
		brief:   tools.BriefTool{Radar: d.radar, Recent: d.recent, Agenda: d.agenda, Crowded: d.cfg.Limits.CrowdedFunding},
		read:    tools.ReadTool{Market: public},
		notices: tools.NoticesTool{Notices: binance.NewNotices(d.client)},
		pulse:   tools.PulseTool{Market: d.futures, TradFi: tradfiMarket{futures: d.futures}, Rule: d.cfg.Radar},
		screen:  tools.ScreenTool{Watchlist: d.watchlist, Research: d.research, Live: d.futures},
		report:  tools.NewsTool{Keeper: d.research},
		osint:   researching(d),
	}

	box.sheet.Server = box.server

	return box
}

func chatModel(cfg config) agents.ChatModel {
	if cfg.ModelKey == "" {
		return nil
	}

	return llm.New(cfg.ModelURL, cfg.ModelKey, cfg.Model)
}

func buildAgents(d agentDeps) *agents.Team {
	box := newToolbox(d)

	return agents.NewTeam(agents.Deps{
		Model:  chatModel(d.cfg),
		Log:    d.log,
		Quick:  quickAgent(d, box),
		Funnel: funnelAgents(d, box),
		Agents: deskAgents(d, box),
	})
}

func quickAgent(d agentDeps, box toolbox) agents.Agent {
	return agents.Agent{
		Name:   analystAgent,
		Prompt: prompts.Of("quick"),
		Tools:  []agents.Tool{box.brief, box.server, box.past},
		Reports: []agents.Tool{
			tools.IdeaTool{
				Market:  d.futures,
				TradFi:  tradfiMarket{futures: d.futures},
				Rule:    d.cfg.Radar,
				Limit:   quickIdeas,
				Crowded: d.cfg.Limits.CrowdedFunding,
			},
		},
		Rounds: quickRounds,
	}
}

func funnelAgents(d agentDeps, box toolbox) agents.Funnel {
	free := []agents.Tool{box.pulse, box.screen, box.server, box.read, box.notices}

	return agents.Funnel{
		Survey: agents.Agent{
			Name:    researcherAgent,
			Prompt:  prompts.Of("funnel-survey"),
			Tools:   append([]agents.Tool{box.server, box.notices}, box.osint...),
			Reports: []agents.Tool{box.report},
		},
		Select: agents.Agent{
			Name:    analystAgent,
			Prompt:  prompts.Of("funnel-select"),
			Tools:   free,
			Reports: []agents.Tool{tools.PickTool{Watchlist: d.watchlist, Limit: d.cfg.Finalists}},
			Memory:  d.cfg.AgentMemory,
		},
		Research: researcher(d, box),
		Dig: agents.Agent{
			Name:   analystAgent,
			Prompt: prompts.Of("funnel-dig"),
			Tools:  []agents.Tool{box.screen, box.server, box.read, box.notices},
			Memory: d.cfg.AgentMemory,
		},
		Rank: agents.Agent{Name: analystAgent, Prompt: prompts.Of("funnel-rank")},
	}
}

func researcher(d agentDeps, box toolbox) agents.Agent {
	return agents.Agent{
		Name:    researcherAgent,
		Prompt:  prompts.Of("osinter"),
		Tools:   append([]agents.Tool{box.server, box.notices, box.past}, box.osint...),
		Reports: []agents.Tool{box.report},
		Memory:  d.cfg.AgentMemory,
	}
}

func deskAgents(d agentDeps, box toolbox) []agents.Agent {
	checking := []agents.Tool{box.server, box.read, box.notices, box.past}

	return []agents.Agent{
		{
			Name:   analystAgent,
			Prompt: prompts.Of("binance"),
			Tools:  []agents.Tool{box.brief, box.pulse, box.screen, box.server, box.past},
			Memory: d.cfg.AgentMemory,
		},
		researcher(d, box),
		{
			Name:    guardAgent,
			Prompt:  prompts.Of("guard"),
			Tools:   append([]agents.Tool{box.entry}, checking...),
			Reports: []agents.Tool{tools.VerdictTool{}},
		},
		{
			Name:    riskAgent,
			Prompt:  prompts.Of("risk"),
			Tools:   append([]agents.Tool{box.sheet}, checking...),
			Reports: []agents.Tool{tools.ScenarioTool{}},
		},
		{
			Name:   traderAgent,
			Prompt: prompts.Of("trader"),
			Tools:  checking,
			Memory: d.cfg.AgentMemory,
		},
	}
}

func researching(d agentDeps) []agents.Tool {
	out := []agents.Tool{
		tools.LinksTool{Directory: d.links},
		tools.FeedTool{News: googlenews.New(d.client)},
		tools.FetchTool{Pages: web.NewFetcher(), Budget: d.cfg.PageBudget},
	}

	if d.cfg.SearchKey != "" {
		out = append(out, tools.SearchTool{Search: tavily.New(d.cfg.SearchKey), Budget: d.cfg.Searches})
	}

	if d.cfg.XToken != "" {
		out = append(out, tools.PostsTool{Posts: xapi.New(d.cfg.XToken), Budget: d.cfg.XBudget})
	}

	return out
}
