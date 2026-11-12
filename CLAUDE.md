# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
It holds only instructions and conventions for working in this project; what the project is and how
to run it is in `README.md`.

## Project

A trading desk for one owner, in two parts that share a server.

The monitor watches Binance announcements and DefiLlama incidents and sends a Telegram message the
moment one of them lands. It sends only the kinds of event after which the price has measurably
moved, each with its importance and the facts around it, and it **compares nothing and forecasts
nothing**: it says what happened. Whether a kind of event is worth sending is decided offline, on
history, against a control moment of the same coin — never by adding scoring to the bot.

The desk — the scan, the agents and the macOS app — exists for one thing: **to understand where a
coin is moving and why**. A move may go either way, so the analysis looks for shorts and longs alike.
A coin is worth showing only when its direction is clear and has a named cause, statistics and an
event pointing the same way; a coin that only moves with the market is the market, not a candidate.
Saying «there is no clear move now» is a full answer, and showing some coin to fill a list is a
defect. Every change to the scan, a tool or a prompt is judged by that. Orders go either way too: a plan
names a short or a long, the checks before an entry and the stop and target mirror each other, and an
order is placed on the Binance Futures testnet only from a plan set to be watched — by the owner, or, while he
keeps auto trading switched on, by the desk before a timed event of a coin — and only after an agent
confirms the plan's condition.

## Project structure

A plain Go layout: `cmd` for the program, `internal` for everything else, one package per feature
or per outside service, named for what it does. No layer folders.

- `cmd/newsbot` — the program: configuration (`config.go`), the sources (`sources.go`), the agents'
  wiring (`agents.go`) and the plan judge (`judge.go`), the testnet check (`check.go`), the builders of
  each feature (`build.go`) and `main.go`, which starts everything. It is the only package that knows every other one.
- Features of the bot:
  - `monitor` — collecting events from the sources, dropping what was seen, enriching, publishing,
    the health of the sources, the latest events, and `Service`, which runs it all;
  - `watchlist` — the scan of every crypto perpetual (spot adds only the Monitoring Tag and the spot
    volume behind the thin mark) and the owner's list;
  - `trading` — the strategy, the account, the desk's checks before an entry, the `Trader` and the
    `Broker` port; how the account is counted is under «Behaviour that must not break»;
  - `research` — the news found about each coin and the risk manager's entry plans for it; a finding
    with an exact date ahead is also a calendar entry of the kind `coin`, with its hour when the
    researcher reported one or the text names it in UTC, all-day otherwise;
  - `autopilot` — the plans the owner set to be watched: it wakes at a plan's price — a touch between two looks counts, read from the minute candles — after its date, if it has one, or on a new event about the coin, asks an agent
    whether the plan's condition holds, and only on a yes places the order through the trader — the
    plans of different contracts side by side, a few at a time, those of one contract one after
    another; once a
    plan enters, the contract's other watched plans are taken off — they were alternatives, not more trades.
    What became of every watched plan — the judge's verdict with its reason, or why it lapsed, and the
    trade it opened with its result — is the desk's shared journal: every agent reads it through
    `desk_outcomes` to weigh a new plan or verdict against what already happened. It holds facts only; no
    agent writes notes for another, and none sees another's answer.
    Setting a plan to be watched or changing its numbers tells the feed and Telegram nothing — the
    watch is seen among the triggers; only what became of it is told.
    What became of a plan is stamped with the moment of the judge's verdict, so it stands before the
    entry it opened.
    A plan made for a calendar event carries the event's moment as its date and sleeps until then.
    `GET /results` counts what became of every watched plan and the desk's closed trades;
  - `autotrade` — the owner's auto-trading switch, off until he turns it on and kept in
    `data/auto.json`. While it is on, `REMINDER_LEAD_MINUTES` before a timed calendar event of a coin — an
    unlock or a date from its news — the risk manager is asked for plans once, the plans are kept
    for the coin page, and the most cautious one that can be armed, never a high-risk one, is set to be
    watched with the event's moment as its date, for two days after it rather than the owner's `PLAN_WATCH_DAYS`: a plan made for an event has no meaning long after it; the judge and the desk's checks do the rest. Events of the same
    minute are planned side by side, a few at a time, so the last is not planned after its moment. No plan is
    a valid answer, and the feed is told quietly what was done; a question the risk manager failed to
    answer is asked once more at the next look, and a second failure is told in the feed too. Nothing else is planned on its own;
  - `radar` — the market's live pulse kept for the app: every minute it reads, from five-minute bars,
    every crypto perpetual — the 150 most liquid and the owner's watchlist closely, with their open
    interest and long share, the rest on candles alone, shown only when marked and never failing the
    pulse — and from a week of 15-minute
    bars, read once an hour per coin and no more than a set number in one pass, each coin's beta to BTC, its usual move and the usual volume of
    each hour; once per five-minute row it also reads each contract's open interest and share of long
    accounts. A contract **woke up** when within the last hour its open interest changed by
    `PULSE_INTEREST_SHIFT` in five minutes while the share of long accounts shifted by
    `PULSE_LONGS_SHIFT` over the quarter ending at the same row; the mark carries how long ago and by
    how much. A contract **launched from quiet** when, within the last four hours, one hourly candle — the
    open one included — rose by `PULSE_LAUNCH_MOVE` and `PULSE_LAUNCH_TIMES` usual hours of the two
    days before it, on five times their usual volume, closing in the upper third of its range, while
    those two days fit in a narrow range; the hours come from the week's bars with the fresh
    five-minute bars over them. A **steady trend** is three usual moves of a day that went fairly straight, or
    two of three days, with the price still near its edge. Beside the crypto board it keeps a TradFi
    board — every commodity perpetual and the 20 most liquid US equity ones — marked by the same rule.
    None of the marks tells a direction;
    what history measured about each is in `agents/tools/measured.go` and reaches the agents through
    the coin card. A mark is only shown: the radar gathers no news, asks no agent and starts no plan; a
    marked contract shows the fresh news already kept about it;
  - `agenda` — the calendar: every few hours it reads the market's events — the Fed's committee
    meetings, press conferences and minutes, GDP and income from BEA, with `FRED_KEY` the days of CPI,
    PPI and the jobs report, the monthly and quarterly expiries of BTC and ETH options, with
    `FINNHUB_KEY` the earnings dates of the 20 US equities of the TradFi board, and with
    `NEWSBOT_UNLOCKS_SCRIPT` the coming unlocks of the owner's table, each with its share of the circulating supply — and keeps what each source last answered, so one
    failed source does not empty the calendar. The coins' dates from `research` it reads live, at every
    look, drops one that repeats an unlock of the same coin at the same moment, and one without an hour when
    the same coin has a date with an hour that day. Through the feed
    (source `market-reminders`) it tells of an all-day event on its day and of a timed one
    `REMINDER_LEAD_MINUTES` ahead and when it starts; an approximate date is never told. The coin card (the coin's own dates only) and
    `server_get` (`calendar`) hand it to the agents;
  - `agents` — the loop that asks a model and runs its tools; `agents/tools` — every tool an agent
    may call, each a thin wrapper over a feature or a client; `agents/prompts` — the embedded role
    prompts.
- Shared vocabulary: `news` (the event, the ticker, the category, an observation, a failure and the Kyiv
  time zone), `exchange` (which Binance titles matter), `market` (the ticker, the pulse, and how a contract's pulse and week are measured from its candles — pure functions the client and the tools share), `incident`
  (hacks and depegs), `schedule` (a dated entry of the market's calendar).
- Clients of outside services, one package each: `binance`, `defillama`, `coingecko`, `googlenews`,
  `fed`, `bea`, `deribit`, `fred`, `finnhub` (the sources of the general calendar; `fred` needs `FRED_KEY`,
  `finnhub` `FINNHUB_KEY`), `unlocks` (the coming unlocks of a table that a script of the owner's prints, kept outside the repository: with `NEWSBOT_UNLOCKS_SCRIPT` set it is run once a day with `NEWSBOT_PYTHON` and prints a CSV — `symbol`, `price_usd`, `next_unlock_time` with its hour, `next_unlock_usd`, `market_cap_usd` —, and a row becomes a timed unlock whose share of the circulating supply is its value against the coin's market value; a row is kept only when that share is a hundredth or more and the table's price agrees with the perpetual's, so another coin under the same ticker is left out),
  `tavily`, `xapi`, `web` (a page as text), `llm` (the model API), `telegram`, `console` (dry run), `chart`,
  `websocket`, `httpclient`.
- `filestore` (the seen-state, the journals, what the unlock table last gave, and one generic JSON file for
  the account, the lists, the scan, the research, the watched plans, the feed and the auto-trading switch), `dotenv`, and `httpapi` — the HTTP API, one
  file per area: `monitor.go` (`/healthz`, `/events`), `watchlist.go` (`/coins`, `/watchlist` — the scan's figures with the perpetual's live price over its six-hour-old one),
  `trading.go` (`/positions`, `/positions/{id}/close`, `/auto` — the auto-trading switch —, `/results`), `research.go` (`/coins/{asset}/news`, `/news/{id}`, `/coins/{asset}/plan` — the coin's plans),
  `research.go` also serves `/watches`, `/coins/{asset}/watches` and `/watches/{id}` (`PUT` changes a watched plan's
  numbers — never its condition, which the agent checks — and `DELETE` takes it off);
  `radar.go` (`/pulse`, `/tradfi`), `agenda.go` (`/calendar`, the past week and the next 90 days of the calendar),
  `agents.go` (`/ideas`, `/scout`, `/agents/{name}`, `/agents/{name}/history` — read it, or `DELETE` to make the agent
  forget, which writes a mark to the journal and removes nothing).
- `deploy` — the systemd unit. `.github/workflows` — CI. `dist` — the app's disk image for a release,
  made from `mac/scripts/build.sh`'s build (debug symbols stripped, so no local path stays in it) with
  an Applications link beside it; before it is committed, the mounted image is searched for local
  paths, names and server details.
- `mac` — the macOS app. `Newsbot/` holds the views: `App` (the app, its menu-bar icon and the settings form),
  `Features/<screen>` (`Main`, `Watchlist`, `Calendar`, `Events`, `Agents`, `Coin` with its `Chart`,
  `Market`, `News`, `Plan`, `Trader` and `Orders`, `Position`, `MenuBar`) and `Components` shared by them, one
  view per file. `Core` is the Swift package `NewsbotCore` with everything testable without a
  screen: `Models`, `Server` (the API client), `Market` (Binance candles, book and derivatives, the
  chart's frame and its event marks, the chart model that keeps the candles and lets the window be
  pulled back into history, the models that keep a coin's book and its derivatives — read at the chart's
  interval, for the window it shows — and an open
  position's price fresh — no view asks Binance itself —, and the `Brake` that stops asking a host
  for at least a minute after it says «too many requests»),
  `Stores` (`AppStore`, `Conversation` — one talk with one agent about one topic, the last ten
  exchanges —, `CoinPage` with the coin's `CoinNews`, `CoinPlans`, trader talk, chart and market, kept per coin in `CoinPages` so
  a window shown again finds a question still waiting for its answer, `Navigator` — which coin and which news a window shows) and `Support`, with its tests. `Newsbot.xcodeproj` builds the app
  from the `Newsbot/` folder, kept in sync by Xcode, with `Core` as a local package; its shared
  scheme is `Newsbot`; its icon, a robot with a chart for a mouth, is in `Newsbot/Assets.xcassets`.
  `scripts/build.sh` and `scripts/test.sh` build and test from the terminal;
  `design` holds the design, frame by frame.

## The macOS app

- The app reads the server only: `/healthz`, `/events` (and removes one), `/coins`, `/watchlist`,
  `/positions`, `/pulse`, `/tradfi`, `/calendar`, `/coins/{asset}/news`, `/coins/{asset}/plan`,
  `/coins/{asset}/watches`, `/watches`, `/results`, `/auto` (which it also turns), the agents' answers and history, and Binance's public futures data (candles, the order book, funding, open
  interest). It decides nothing the server has not decided; what it shows is what the server says.
- `NewsbotCore` holds everything that can be tested without a screen and imports no SwiftUI; the
  `Newsbot` target holds views only. The store asks the server every 15 seconds for the events, the
  account, the triggers and the results, all at once, for the pulses every 30 seconds and for the
  coins and the calendar every five minutes, and keeps a value only when it changed, so a view is
  drawn again only when its data did. A view keeps no logic beyond layout: ordering, filtering and
  what a window should show — the feed newest first by the event's own time among them — live in a store, and windows talk through the `Navigator`, not through
  stored strings.
- The same code style as Go: no comments, a blank line between steps, small types, names that explain.
- The main window is the overview — tokens (the owner's list, whose search reaches every scanned coin, and
  «Монети» and «TradFi»: the most liquid crypto perpetuals with every other marked one, and commodities and US equities, live, the contracts
  that woke up first, the newest on top, each with its shifts and the fresh news kept about it),
  events, the analyst's chat, the sources; everything
  about one coin, trading included, is on its page: news and a chat with the researcher, the entry plans the
  risk manager makes on request, a chat with the trader, the plans that wait to become orders, its positions.
- The app acts only through the server: arming a plan and closing a position are server calls; it
  holds no exchange key. It reaches a remote server only through an SSH tunnel it opens itself while it
  runs (`Tunnel`, with the server and the key in the app's Settings, never in the code) and closes on quit.
- The watchlist is the owner's: a coin enters and leaves it only by the owner's hand
  (`data/watchlist.json`), found by the search in the app's «Вішліст» tab or added from the coin's page. The scan supplies market data
  and marks thin or tagged coins; it never edits the list.
- The agents — Binance-аналітик, ОСІНТер, Ризик-менеджер, Трейдер — and the plan judge, who is never
  talked to, run on the server, which holds the model key: the app sends the owner's words and the coin they are about to `POST /agents/{name}`; the server
  names the coin in what the model reads, keeps the owner's words as they were, calls the model
  (an OpenAI-compatible API, `MODEL_KEY`, `MODEL_URL` and `MODEL_NAME` in `.env`) and runs its tools until it answers. No
  agent sees another's answer. The analyst and the researcher are shown their own last exchanges
  (`AGENT_MEMORY_EXCHANGES`, read back from the journal; the researcher's only about the same coin);
  the trader likewise sees its last exchanges about the coin, since the owner talks to it; the risk
  manager starts every question anew. The app shows the last ten exchanges with the analyst, the researcher and
  the trader, read from the same journal. A prompt
  (`internal/agents/prompts/*.md`) gives a role, a way of working and the shape of the answer, and holds
  no reference data, numbers or limits; a part several prompts share lives once in a file starting with
  `_` and stands in a prompt as a line `{{name}}`. What history measured — only the exchange
  announcements paid over the twelve months to October 2026; a large unlock was followed by a week's
  decline about seven times in ten on a few dozen cases, and a stop closer than half the coin's usual
  daily range was knocked out by noise; a release and a mark of the pulse gave no direction — is kept
  in `agents/tools/measured.go` and reaches the agents through the coin card. What an agent knows it
  gets through its tools:
  - the analyst — the quick state of the market of coins and TradFi, and no research of its own:
    `desk_brief` (one call that lists only the marked contracts of both boards — woke up or
    in a steady trend — each with its key factors: the run so far, volume against the usual, the takers,
    funding and a crowded side, the hour's change in the share of long accounts, the researcher's fresh
    news about it, the bot's events, the calendar ahead and what history measured about its case),
    `market_pulse` (the same live pulse the quick ideas read, for a wider look), `market_screen` (the
    bot's scan of every crypto perpetual, from its daily candles, filtered, sorted and cut in one call:
    tag, volume, changes, trend against BTC, mean day range, the takers' sell share, the perpetual's
    live price and day, funding brought to eight hours with the period it is really paid at, open
    interest, depth of the perpetual's book near the price) and `server_get` (the coin card and the
    market's calendar). It is the one agent of the main window's chat;
  - ОСІНТер — `news_feed` (free headlines of news outlets, from Google News' public feed, for a list
    of queries answered apart in one call), `project_links` (the project's official site, X account and Telegram from CoinGecko,
    remembered per coin while the server runs), `web_search` (a web search through the Tavily API, present when
    `TAVILY_KEY` is set; one question may spend the searches the setting allows, and the tool refuses
    the rest), `x_posts` (public posts of the last seven days through the X API, by account or by
    query, present when `X_BEARER_TOKEN` is set and limited by `X_READS_PER_QUESTION`), `http_get` (any page, HTML turned into text with each link's address kept, limited by
    `PAGES_PER_QUESTION`), `binance_announcements` and `report_news` (one dated fact with
    its sources, kept with the coin and shown as a card the owner may remove) and `server_get`, to start
    from what is already kept about the coin. The paid tools refuse to run in a question that has not
    called `news_feed`, and a question may spend `OSINTER_SEARCHES_PER_QUESTION` web searches (3 by
    default). It is asked from the coin card, and by the search for candidates first about the world and
    then about each finalist;
  - Ризик-менеджер — `plan_sheet` (one call for the coin card and the perpetual's market to set levels
    from: the price's range over four hours, a day, three days and a week, the hour's and the day's
    move, volume and the takers against the usual, open interest and the long accounts with their change,
    BTC's move), `server_get` (the coin card — the perpetual's live price over the scan's figures, the
    news found about it, the bot's events, its pulse with how far it usually moves in an hour, the
    analyst's latest ideas about it read from the journal, its plans and the watched ones, positions,
    and the desk's rules: whether an entry would pass the desk's checks now, apart for a short and a
    long, the smallest order the exchange takes as a share of the account, the hours after which a
    position is closed and the days a plan is watched —, events, calendar, account, watchlist, coins),
    `binance_get`, `binance_announcements`, `desk_outcomes` and `plan_scenario` (one scenario of the
    entry plan: a condition, an action, a risk level and why). The coin card asks it for the plans, with
    the owner's wish for them if there is one: a request and a result, not a talk;
  - the plan judge — `plan_check` (one call for what the market did since the moment the plan waited
    from: the price then and now, the low and high since, whether the threshold was crossed, the volume
    and the takers against the usual, open interest and the long accounts then and now, BTC's move,
    funding, the last five-minute bars, the desk's checks for this plan's side and size, the bot's
    events, the researcher's latest news and the calendar ahead), the risk manager's reading tools and
    `entry_verdict`;
  - Трейдер — `server_get`, `binance_get`, `binance_announcements` and `desk_outcomes`, and no reporting
    one. It answers the owner's questions about the coin
    in free form, in a chat on the coin card. Making plans stays with the risk manager.
- Every question to an agent is kept in `data/agents.jsonl`: the tool calls with their arguments and
  the head of each result, the findings, the answer, the time it took and the model's tokens in and out. An agent's work is judged
  from this file, not from memory.
- Quick ideas are the analyst's short mode, `POST /ideas`: `desk_brief` — the marked contracts of the
  coin and TradFi boards with their key factors, the researcher's news and what the bot saw about each —, a
  coin card for a contract about to be handed over, and
  up to five `trade_idea`s: a coin, a direction, why now, what cancels it and the event beside it with the
  side it works for (an idea with an event against it is refused); the pulse gives every coin the desk's
  own verdict, `woke_up`, by the rule the radar marks contracts with; the tool adds the price and
  the funding itself and refuses an idea into the side funding shows as crowded, by the limit the desk
  applies to an entry. The pulse answers several views in one call, so a run is a few rounds. No paid source, no memory,
  no prices of entry: the risk manager makes the plan on the coin's page. An idea to fill the list is a defect.
- The search for candidates is a funnel, `POST /scout`. The researcher goes first: it surveys what is
  happening outside Binance — hacks, rulings, launches, unlocks, buybacks — and keeps each event with the
  instrument it concerns, so a coin with an event and no move yet is not missed. Then come three stages
  by the analyst: selection reads
  the whole market from the bot's own data and Binance, looks at the coins with a fresh finding apart
  (`market_screen`, `fresh_news_only`) and names a few finalists (`pick_finalist`,
  `FUNNEL_FINALISTS`); each finalist is then given to the researcher, who is asked about it with the direction and
  the reason it was picked and keeps what it finds in the coin card, and only then studied to the bottom by
  the analyst in its own conversation, from the card and the market, with its own memory by coin — the
  finalists side by side; a last call
  ranks them from the dossiers alone; the analyst reads what the researcher kept through the card, never
  its answer.
- An agent first gathers with all its tools, then is told to report and keeps only the reporting
  tools, so what it found is never lost to the step limit.
- A model extracts facts and dates; it never decides what the bot sends or whether to enter — on
  history it called 93% of events significant. A finding carries the day its source
  published it and, for what is still ahead, the day of the event and its hour when the source names
  it; both calendars of the app — the main window's and the coin page's — are the server's. The main
  one follows the open tab and has no approximate dates: coins' dates, unlocks and option expiries under
  the watchlist and the coins, US releases, the Fed and earnings under TradFi; the coin page's has only
  that coin's, and for a TradFi contract the general releases beside its own. Calendar reminders are
  sent to Telegram and kept out of the app's feed. An unlock's title starts with its share of the
  circulating supply, and the calendar shows one of a twentieth or more in bold orange. The coin chart
  marks what already happened to the contract — its calendar entries of the past week and the bot's
  events — with a faint yellow line and a dot above it, telling the event when the candle is pointed
  at, and draws an entry that is ten minutes away or nearer as a dashed yellow line ahead of the last
  candle, with the minutes left.

## Design

`mac/design/canvas/` holds the agreed design, one `*.dc.html` per frame, and it is the **only** source
for anything visual. `mac/design/Карта кадрів.md` says what each frame decides; find the frame there
first, then read the values off the frame by opening it in a browser — never from memory or a
screenshot. Where the frames say nothing, ask. The canvas is edited in claude.ai and copied here
whole, every frame and the map together, never one of them alone.

## Architecture

- A feature (`monitor`, `watchlist`, `trading`, `research`, `autopilot`, `radar`, `agenda`, `agents`) imports the shared vocabulary
  and never a client of an outside service: it declares the small interface it needs, where it is
  used, and `cmd/newsbot` hands in the client. `agents/tools` is the bridge between the model and the
  rest, so it may import both.
- A client turns someone else's API into plain Go values and decides nothing: a change in Binance's
  or DefiLlama's format stops there. Descriptions for a model, budgets and limits of a question live
  in `agents/tools`, not in a client.
- Interfaces are small (one to three methods). A feature that needs another one sees only the calls
  it makes: the monitor sees the trader and the scanner as things that run, the tools see the
  account, the watchlist and the research as the few reads they make.
- Time enters a rule as a parameter (`now time.Time`), so a rule is tested without waiting.
- Title classification is a pure function, `ClassifyBinance(title) (Classification, bool)`.
- Value objects (`Symbol`, `Category`, `Severity`, `Direction`) are passed by value and never mutated.
  Tickers are read out of a title only through the `news.SymbolsFrom…` functions.
- One term has one name everywhere: `Event`, `ShortTrigger`, `Incident`, `Announcement`, `Symbol`,
  `Seed` (the silent first poll), `Seen` (already handled), `Finding`, `Archive`.
- A source is anything behind `monitor.Source` — polled or pushed over a WebSocket — and differs from
  another only behind that interface, never through a `switch` on its name.
- Tests sit beside their code and bring their own small fakes; nothing is shared between test files
  of different packages.

## Code style

- **Clear, without filler, in its place** — the first rule for code, prompts, tool descriptions and
  documents alike. Length is never the problem: a long file or a full description is fine. What is
  wrong is a sentence that says nothing, the same thing said twice, a description that retells how it
  came to be instead of what it is, and a thing kept where a reader would not look for it.
- **No comments** — no `//`, no `/* */`, no doc comments, no separators, no `TODO`, tests included.
  The name of the type, the function and the variable explains it; a wish to comment is a sign to
  rename or to extract a function named after the comment. Compiler directives (`//go:build`,
  `//go:embed`) are not comments.
- **A function is not one block.** Every distinct step is separated by a blank line: a call, its
  handling, the next call, its handling, the result. A blank line follows every block (`if`, `for`,
  `switch`, `select`, a function literal), and precedes a `return` that has statements before it.
  Only lines that make one step stay together — a declaration and the check of that same call's error.
- **Nothing is written on one line that is a block**: a declaration with a body (`func`, a method, a computed
  `var`), an `if`, `guard`, `for` or `switch` opens and closes its braces on lines of their own, even for
  one statement; in Go a function literal's body always starts on a new line (`wg.Go(func() {` … `})`),
  and a one-line predicate is a named function. Only a short closure inside a Swift chain
  (`.filter { $0.active }`) may stay on one line.
- A struct literal with several fields, a long call, a signature or a compound condition is split one
  field, argument or `||`-part per line once it stops reading at a glance.
- A constructor with many dependencies takes a `…Deps` struct, not a row of arguments.
- **KISS.** The simplest thing that works; no abstraction for a future that has not come. Only the Go
  standard library — a third-party module only when a task cannot be done without it, with the reason
  written here. A function does one thing. No global mutable state and no `init()` with side effects;
  dependencies are passed explicitly.
- **DRY.** One piece of knowledge, one place: ticker parsing, money formatting, the HTTP request with
  `Retry-After`, title classification. Twice is tolerable; the third time it moves into a shared
  function or type. DRY does not justify coupling two things that change for different reasons.
- **Dead code is deleted, not kept.** A removed feature takes with it everything only it used: types,
  functions, constants, configuration, environment variables, documentation lines.
- Errors are wrapped with context (`fmt.Errorf("poll %s: %w", name, err)`); `panic` only for an
  impossible state at start-up.
- Every network operation takes a `context.Context` and has a timeout. Goroutines start only in
  a feature package (and inside a client that owns a connection, or that fans one read out over many
  contracts with a bounded number of workers), stop through `ctx`, and are awaited.
- Tests are table-driven where there are cases, check behaviour through the package's API, and use
  fakes of the ports instead of live requests.

## Behaviour that must not break

- **Seed:** a source's first poll remembers what is already there and sends none of it.
- **No repeats:** an event's ID is stable whatever the language, region or server address. Any
  normalisation of an ID happens in the adapter. An announcement's ID is its normalised title, so one
  announcement is never sent twice. Changing an ID format needs a plan, so
  that old events do not come out again.
- **Only the first minutes:** an announcement older than its maximum age is muted, not sent late.
- **State survives a restart:** `data/state.json` is saved periodically and on shutdown, the
  desk's positions `data/account.json` and the feed's last 200 events `data/events.json` on every change, and
  what the owner's unlock table last gave `data/reads.json` after each run, so a restart does not run
  its script again before its time.
- **Trading is on the testnet only:** the code knows the Binance Futures testnet address and no
  address of the real exchange, so no order with real money can leave it. Real trading is a separate
  decision of the owner, with its own key, never on the way of another change.
- **The account is the exchange's:** the equity is the balance the broker reads every minute, a plan's
  size is a share of it, and a closed trade's result is what the exchange booked — the realised result
  with both fees and the funding; the desk's own estimate stands only until then and is replaced at the
  next look. With `ACCOUNT_CAPITAL` the desk counts from that working capital and the exchange's balance
  only moves it, so the test account is sized like the real one; a balance that jumps by more than the
  whole account in one look is a reset or a transfer and starts the count anew. The desk keeps only what
  the exchange does not: which plan a position came from and why. Without a testnet key it is idle.
- **An order the exchange would refuse is not sent:** the broker says the smallest order of a contract —
  the minimum rounded up to a whole step of quantity — and whether its venue trades it; the desk checks
  an entry against both before sending, and the risk manager and the judge see both in their checks.
- **A watched plan enters only through the same door:** it is armed by the owner or by auto trading,
  one plan one order, only with a price threshold, and lapses after `PLAN_WATCH_DAYS`; a date, if it
  has one, only makes it sleep until then. An agent must confirm its condition when its price comes or
  the bot sees an announcement about the coin, and the order then passes every check of
  `Trader.Place`. A doubt is a no: a plan is asked about once, and one whose condition the agent did
  not confirm is dropped at once — there is no waiting and no second check; a reminder of the calendar
  never wakes a plan.
- **A plan's condition is numbers seen at the touch:** the judge reads the market at the moment the
  price reaches the threshold and searches nothing, so a condition names a price, volume, open interest,
  funding or the share of buyers or long accounts — never a time to wait or a price that must «hold».
- **Only the researcher reaches the world:** the news feed, pages, a project's links, the paid web
  search and the reads of X are its tools alone; every other agent works from Binance, the bot's own
  data and what the researcher kept in the coin card.
- **No agent takes an instrument for a coin by default:** the card names what it is — a crypto coin or a
  TradFi contract —, the prompts speak of an instrument, and what moves it follows from its kind.
- **A host that says «too many requests» is left alone:** on HTTP 429 or 418 `httpclient` asks that
  host nothing more for the time it named, a minute at the least; the callers fail for that while and
  try again on their next round. Nothing retries around it.
- **A position is never left without a stop:** if the stop and the target cannot be placed after an
  entry, the position is closed at once and the failure reported.
- **Everything seen is journaled:** every event seen for the first time, sent or muted, and every
  error of a source or of Telegram goes to `data/journal.jsonl`.
- **Respect for the sources:** pauses between requests, exponential back-off, `Retry-After`. No
  bypassing of bot protection, no browser User-Agent, no proxies around a geo-block: a source that
  blocks the server is switched off with `NEWSBOT_DISABLE`.
- Thresholds worth tuning (`HACK_MIN_USD`, `DEPEG_THRESHOLD`, `ANNOUNCEMENT_MAX_AGE_MINUTES`) and the
  list of disabled sources live in `.env`; the remaining constants live in `cmd/newsbot/config.go`,
  not scattered across packages.

## Checks

Run all of them once, after every edit a task needs and before committing:

- `test -z "$(gofmt -l .)"`, `go vet ./...` and `go test -race ./...`;
- `deadcode ./...` (`golang.org/x/tools/cmd/deadcode`) prints nothing;
- `grep -rn --include='*.go' '//' . | grep -v '://' | grep -v '//go:'` prints nothing;
- `grep -rnE --include='*.go' 'func\([^)]*\)[^{]*\{ .+ \}|^func .*\{ .+ \}$' .` prints nothing, and in the app
  `grep -rnE --include='*.swift' '^\s*((public|private|static) )*(var|func)\b.*\{ .+ \}$' mac/Newsbot mac/Core` shows only
  assignments of closures;
- imports are grouped standard library first, then `newsbot/…` (`goimports -local newsbot`);
- no feature package imports a client: `go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./internal/...`
  shows `binance`, `defillama`, `coingecko`, `googlenews`, `fed`, `bea`, `fred`, `deribit`, `tavily`, `xapi`,
  `web`, `llm`, `telegram` only under
  `agents/tools`, the clients themselves and `cmd/newsbot`;
- for the app, `swift format lint -r mac/Newsbot mac/Core/Sources mac/Core/Tests` prints nothing
  (`mac/.swift-format` sets four spaces), `mac/scripts/test.sh` (the core's tests, `swift test`) and
  `mac/scripts/build.sh` (the app through `xcodebuild`, copied to `mac/.build/Newsbot.app`).

CI runs all of them but the import order, the feature-import check and the app's build: the runner's Xcode may
be older than the project's.

Before a new binary replaces the one on the server, run it with `DRY_RUN=1` against a copy of the
server's `data/` with every `NEWSBOT_*` path pointing into it: nothing already sent may come out again.
Run it with `NEWSBOT_DISABLE=pulse,watchlist`, so it does not read the whole market a second time beside
the live server — that drew a «too many requests» from Binance twice —, and with the testnet keys and
`MODEL_KEY` set empty, so the copy can neither ask the judge nor send an order beside the live server;
the copy of `data/reads.json` keeps it from running the unlock table's script. The previous binary stays beside it as `newsbot.prev`
for a rollback.

## Documentation

- `README.md` is for a stranger: what the bot sends and why, what it does, what was tried and left
  out, how the code is laid out, how to run it, the licence. It names no infrastructure — no cloud
  project, address, user name or key path.
- `CLAUDE.md` holds how work is done here.
- `.env.example` lists every variable the code reads, and nothing it does not.
- Each describes the system as it stands, factual and without filler. **None may contradict another or the
  code.** Syncing them is the last step of a change, in the same commit: re-read every section the
  change touched and correct what stopped being true.

## Secrets in the repository

- `.env` holds every key: the Telegram token and chat, the Binance keys, and those of the model, the search,
  X, FRED and Finnhub. It stays in `.gitignore` and never
  enters a commit: a token in a commit is a token to rotate, since history keeps it.
- Nothing else in the repository holds a secret literal, and nothing names the infrastructure it runs
  on. Before every push, scan what is pushed for both.
- The Binance key has read permission only.

## Commits

- The only author is `Mykhailo Biloshkurskyi <zmishabelijj@gmail.com>`, set in this repository's
  local git config. No co-authors, no trailers, no mention of tools or assistants.
- One commit per finished piece, written the moment the piece is checked and green.
- Messages are English. A subject is one of two states, a colon, and in a few words what the change
  did: `feat` for what the bot gained, `fix` for what it stopped doing wrong — a refactoring is a
  `fix`. There is no third state. The subject is **one action**: a subject that needs «and» is two
  commits.
- **The subject names the piece; the body tells it whole** — what was done (not a file list), why,
  what was decided on the way and what was left undone. A reader of the log should not need the diff.
- Read `git status` first and stage by name; `git add -A` only when the status shows nothing but the
  piece.
- Write the message through a heredoc — `git commit -q -F - <<'MSG' … MSG` — not `-m`.
