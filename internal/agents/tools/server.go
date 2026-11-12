package tools

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/news"
	"newsbot/internal/radar"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

const (
	eventsForAgents   = 100
	calendarForAgents = 30 * 24 * time.Hour
	calendarInCard    = 14 * 24 * time.Hour
)

type Events interface {
	Latest(n int) []news.Event
}

type Account interface {
	Book() trading.Book
	Sized(ctx context.Context, m trading.Market) trading.Market
}

type Coins interface {
	Coins() []watchlist.Kept
	Kept() []watchlist.Kept
}

type Shelf interface {
	Of(asset string) []research.Finding
}

type Plans interface {
	Plan(asset string) (research.Plan, bool)
}

type Radar interface {
	Latest() radar.Board
	LatestTradFi() radar.Board
}

type Agenda interface {
	Upcoming(now time.Time, within time.Duration) []schedule.Entry
}

type Watched interface {
	Of(asset string) []autopilot.Watch
}

type ServerTool struct {
	Recent    Events
	Trader    Account
	Watchlist Coins
	Research  Shelf
	Plans     Plans
	Watched   Watched
	Live      Quotes
	Radar     Radar
	Agenda    Agenda
	Journal   Journal
	Rules     DeskRules
}

func (t ServerTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "server_get",
		Description: "Читає дані бота Newsbot. what=coin — картка одного інструмента, потрібен asset; криптомонета це чи " +
			"контракт TradFi, каже поле instrument. У картці: ринкові дані з живою ціною перпа; знайдені про інструмент " +
			"новини й події бота про нього; свіжі ідеї аналітика; перевірки входу столу для шорта й лонга — серед них, чи " +
			"торгується контракт на майданчику столу, — найменший ордер, який приймає біржа, у відсотках рахунку " +
			"(smallest_order_pct_of_account), і строки — коли стіл закриває позицію за часом і скільки днів стежить за планом; найближчі події загального календаря; " +
			"плани ризик-менеджера, плани на стеженні й позиції. " +
			"Пульс останніх годин у картці: відколи триває рух і скільки пройдено; скільки інструмент звичайно " +
			"проходить за годину без причини (usual_1h_move_fraction); частка акаунтів у лонг і її зміна за годину; " +
			"вердикти столу — зліт із тиші (launched_from_quiet: після двох діб майже без руху годинна свічка в кілька " +
			"разів більша за звичайну на зрослому обсязі), контракт ожив (woke_up: різко змінились відкритий інтерес і " +
			"частка акаунтів у лонг), стала тенденція на добу чи три (steady_trend_hours) — і що про такі випадки " +
			"виміряно на історії (measured_on_history). " +
			"what=events — останні події, які бачив бот (анонси бірж, злами, угоди); calendar — загальний календар " +
			"ринку: рішення ФРС, статистика США, експірації опціонів, із часом за UTC; positions — рахунок і позиції; " +
			"watchlist — вішліст власника; coins — усі монети з ринковими даними.",
		Parameters: object(map[string]any{
			"what":  enum("coin", "events", "calendar", "positions", "watchlist", "coins"),
			"asset": text("тікер: обов'язковий для coin, звужує coins"),
		}, "what"),
	}
}

func (t ServerTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		What  string `json:"what"`
		Asset string `json:"asset"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	switch args.What {
	case "coin":
		return encode(t.card(ctx, watchlist.Normalize(args.Asset)))
	case "events":
		return encode(t.Recent.Latest(eventsForAgents))
	case "calendar":
		return encode(t.Agenda.Upcoming(time.Now().UTC(), calendarForAgents))
	case "positions":
		return encode(t.Trader.Book())
	case "watchlist":
		return encode(t.Watchlist.Kept())
	case "coins":
		return encode(filter(t.Watchlist.Coins(), args.Asset))
	default:
		return "", fmt.Errorf("невідоме what: %q", args.What)
	}
}

const (
	instrumentCoin   = "криптомонета"
	instrumentTradFi = "TradFi: сировина, акція чи фонд США"
)

type coinCard struct {
	Asset         string             `json:"asset"`
	Instrument    string             `json:"instrument"`
	Known         bool               `json:"known_to_bot"`
	MonitoringTag bool               `json:"monitoring_tag"`
	Marks         []string           `json:"marks"`
	Price         float64            `json:"price_usd"`
	PriceSource   string             `json:"price_source"`
	Change24h     float64            `json:"perp_change_24h_fraction"`
	VolumeDay     float64            `json:"perp_volume_last_day_usd"`
	VolumeAverage float64            `json:"perp_volume_30d_average_usd"`
	Change1d      float64            `json:"change_1d_fraction"`
	Change7d      float64            `json:"change_7d_fraction"`
	TrendVsBTC    float64            `json:"change_90d_vs_btc_fraction"`
	Contract      string             `json:"perp_contract"`
	Funding       float64            `json:"funding_8h_fraction"`
	FundingHours  int                `json:"funding_paid_every_hours"`
	OpenInterest  float64            `json:"open_interest_usd"`
	DayRange      float64            `json:"mean_day_range_30d_fraction"`
	SellDay       float64            `json:"perp_taker_sell_share_1d_fraction"`
	SellWeek      float64            `json:"perp_taker_sell_share_7d_fraction"`
	Depth         float64            `json:"perp_depth_within_1pct_usd"`
	SpotVolume    float64            `json:"spot_volume_30d_average_usd,omitempty"`
	News          []research.Finding `json:"news"`
	BotEvents     []cardEvent        `json:"bot_events"`
	Announced     string             `json:"bot_events_measured_on_history,omitempty"`
	Pulse         *cardPulse         `json:"pulse_now,omitempty"`
	Ideas         []cardIdea         `json:"analyst_ideas,omitempty"`
	Desk          *cardDesk          `json:"desk,omitempty"`
	Calendar      []schedule.Entry   `json:"market_calendar"`
	Releases      string             `json:"market_calendar_measured_on_history,omitempty"`
	Unlocks       string             `json:"unlocks_measured_on_history,omitempty"`
	Plan          *research.Plan     `json:"entry_plans,omitempty"`
	Watched       []cardWatch        `json:"watched_plans"`
	Positions     []trading.Position `json:"positions"`
}

type cardPulse struct {
	pulseMove
	pulseMarks
	Measured []string `json:"measured_on_history,omitempty"`
}

type cardWatch struct {
	Plan      research.Scenario `json:"plan"`
	State     autopilot.State   `json:"state"`
	Note      string            `json:"last_check,omitempty"`
	Expires   time.Time         `json:"expires"`
	NotBefore time.Time         `json:"not_before,omitzero"`
	Checks    int               `json:"checks"`
}

type cardEvent struct {
	Time  time.Time `json:"time"`
	Kind  string    `json:"kind"`
	Title string    `json:"title"`
}

func (t ServerTool) instrument(asset string) string {
	if _, traded := t.Radar.LatestTradFi().Of(asset); traded {
		return instrumentTradFi
	}

	return instrumentCoin
}

func (t ServerTool) card(ctx context.Context, asset string) coinCard {
	card := coinCard{Asset: asset, Instrument: t.instrument(asset), News: t.Research.Of(asset)}
	now := time.Now().UTC()

	t.scanned(&card)
	t.live(ctx, &card, now)

	card.Ideas = t.ideas(asset)
	card.Calendar = slices.DeleteFunc(t.Agenda.Upcoming(now, calendarInCard), func(e schedule.Entry) bool {
		return e.Kind != schedule.Expiry && len(e.Assets) > 0 && !slices.Contains(e.Assets, asset)
	})
	card.Releases = releaseMeasured(card.Calendar)
	card.Unlocks = unlocksMeasured(card.Calendar)
	card.Pulse = t.pulsed(asset)
	card.Watched = t.watched(asset)
	card.BotEvents, card.Announced = t.botEvents(asset)
	card.Positions = t.positions(asset)

	if plan, kept := t.Plans.Plan(asset); kept {
		card.Plan = &plan
	}

	return card
}

func (t ServerTool) scanned(card *coinCard) {
	for _, k := range t.Watchlist.Coins() {
		if k.Coin.Asset != card.Asset {
			continue
		}

		c := k.Coin

		card.Known = true
		card.MonitoringTag = c.Tagged
		card.Marks = k.Reasons
		card.Price = c.Price
		card.VolumeDay = c.VolumeDay
		card.VolumeAverage = c.Volume
		card.Change1d = c.Change1d
		card.Change7d = c.Change7d
		card.TrendVsBTC = c.TrendVsBTC
		card.Contract = c.Contract
		card.Funding = c.Funding
		card.FundingHours = c.FundingHours
		card.OpenInterest = c.OpenInterest
		card.DayRange = c.DayRange
		card.SellDay = c.SellDay
		card.SellWeek = c.SellWeek
		card.Depth = c.Depth
		card.SpotVolume = c.SpotVolume
		card.PriceSource = "scan"
	}
}

func (t ServerTool) live(ctx context.Context, card *coinCard, now time.Time) {
	ticker, listed, err := t.Live.Ticker(ctx, news.Symbol(card.Asset))
	if err != nil || !listed || ticker.Price <= 0 {
		return
	}

	card.Price = ticker.Price
	card.PriceSource = "live"
	card.Change24h = ticker.Change24h / 100
	card.Contract = ticker.Contract
	card.Funding = ticker.Funding
	card.FundingHours = ticker.FundingHours
	card.Desk = t.desk(ctx, ticker, now)
}

func (t ServerTool) pulsed(asset string) *cardPulse {
	coin, found := t.Radar.Latest().Of(asset)

	if !found {
		coin, found = t.Radar.LatestTradFi().Of(asset)
	}

	if !found {
		return nil
	}

	return &cardPulse{
		pulseMove:  moveOf(coin.Pulse),
		pulseMarks: marksOf(coin.Wake, coin.Launch, coin.Trend),
		Measured:   measured(coin),
	}
}

func (t ServerTool) watched(asset string) []cardWatch {
	var out []cardWatch

	for _, w := range t.Watched.Of(asset) {
		out = append(out, cardWatch{
			Plan:      w.Plan,
			State:     w.State,
			Note:      w.Note,
			Expires:   w.Expires,
			NotBefore: w.After,
			Checks:    w.Checks,
		})
	}

	return out
}

func (t ServerTool) botEvents(asset string) ([]cardEvent, string) {
	var (
		out  []cardEvent
		note string
	)

	for _, e := range t.Recent.Latest(eventsForAgents) {
		if !slices.Contains(e.Symbols, news.Symbol(asset)) {
			continue
		}

		out = append(out, cardEvent{Time: e.Time, Kind: e.Kind, Title: e.Title})

		switch e.Category {
		case news.ShortTrigger:
			note = announcedMeasured
		case news.Incident:
			note = cmp.Or(note, incidentMeasured)
		}
	}

	return out, note
}

func (t ServerTool) positions(asset string) []trading.Position {
	var out []trading.Position

	for _, p := range t.Trader.Book().Positions {
		if string(p.Symbol) == asset {
			out = append(out, p)
		}
	}

	return out
}

func filter(coins []watchlist.Kept, asset string) []watchlist.Kept {
	asset = watchlist.Normalize(asset)

	if asset == "" {
		return coins
	}

	var out []watchlist.Kept

	for _, c := range coins {
		if strings.Contains(c.Coin.Asset, asset) {
			out = append(out, c)
		}
	}

	return out
}
