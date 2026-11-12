package tools

import (
	"context"
	"math"
	"slices"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

const (
	sizeHeadroom   = 1.1
	reachDayMoves  = 0.75
	noiseDayMoves  = 0.5
	cautiousSpread = 0.3
)

type planTicket struct {
	Price         float64  `json:"price_now"`
	DayMove       float64  `json:"usual_day_move_fraction"`
	Threshold     float64  `json:"threshold_from_price_fraction"`
	ThresholdDays float64  `json:"threshold_in_day_moves"`
	Stop          float64  `json:"stop_from_entry_fraction"`
	StopDays      float64  `json:"stop_in_day_moves"`
	Take          float64  `json:"take_from_entry_fraction"`
	TakeDays      float64  `json:"take_in_day_moves"`
	Loss          float64  `json:"loss_at_stop_fraction_of_account"`
	Smallest      float64  `json:"smallest_order_fraction_of_account,omitempty"`
	Wrong         []string `json:"what_is_wrong,omitempty"`
}

func (t ScenarioTool) ticket(ctx context.Context, turn *agents.Turn, s research.Scenario) (planTicket, bool) {
	if t.Market == nil || turn.Topic == "" {
		return planTicket{}, false
	}

	ground, found := t.Market.Footing(ctx, watchlist.Normalize(turn.Topic))
	if !found {
		return planTicket{}, false
	}

	price, day := ground.Price, ground.DayMove

	entry := s.Wake()
	threshold := math.Abs(entry/price - 1)
	stop, take := math.Abs(s.Stop/entry-1), math.Abs(s.Take/entry-1)

	ticket := planTicket{
		Price:         price,
		DayMove:       day,
		Threshold:     threshold,
		ThresholdDays: threshold / day,
		Stop:          stop,
		StopDays:      stop / day,
		Take:          take,
		TakeDays:      take / day,
		Loss:          s.SizePct / 100 * stop,
		Smallest:      ground.Smallest,
	}

	if ground.Smallest > 0 && s.SizePct/100 < ground.Smallest*sizeHeadroom {
		ticket.Wrong = append(ticket.Wrong, "розмір упритул до найменшого ордера біржі або менший за нього: щойно ціна зміниться, біржа ордер не прийме — бери розмір із запасом над найменшим ордером")
	}

	if ticket.ThresholdDays > reachDayMoves {
		ticket.Wrong = append(ticket.Wrong, "поріг далі за три чверті звичайного денного ходу: ціна до нього звичайно не доходить, і план, найімовірніше, не ввійде")
	}

	if ticket.StopDays < noiseDayMoves {
		ticket.Wrong = append(ticket.Wrong, "стоп ближчий за половину звичайного денного ходу: його виб'є шум раніше, ніж план виявиться правим чи хибним")
	}

	if take < stop {
		ticket.Wrong = append(ticket.Wrong, "тейк ближчий за стоп: можлива втрата більша за можливий прибуток")
	}

	if ground.Harm != "" && s.Long() {
		ticket.Wrong = append(ticket.Wrong, "лонг проти свіжої події бота «"+ground.Harm+"»: подія, що шкодить проєкту, закриває лонг")
	}

	if !ground.Caused {
		ticket.Wrong = append(ticket.Wrong, "у картці немає ні події, ні новини, ні дати календаря про інструмент: у плану немає причини, а позначка столу причиною не є")
	}

	if farther(turn.Scenarios, s, price, day) {
		ticket.Wrong = append(ticket.Wrong, "поріг найобережнішого плану далі від ціни, ніж у ризикованіших планів цього боку: обережність — це розмір і ясна перевага в обігу, а не далекий поріг")
	}

	return ticket, true
}

func farther(scenarios []research.Scenario, s research.Scenario, price, day float64) bool {
	if s.Risk != research.RiskMinimal {
		return false
	}

	own := math.Abs(s.Wake()/price-1) / day

	for _, other := range scenarios {
		if other.Direction != s.Direction || other.Risk == s.Risk || !other.Numbered() {
			continue
		}

		if own-math.Abs(other.Wake()/price-1)/day > cautiousSpread {
			return true
		}
	}

	return false
}

type Ground struct {
	Price    float64
	DayMove  float64
	Smallest float64
	Harm     string
	Caused   bool
}

func (t ServerTool) Footing(ctx context.Context, asset string) (Ground, bool) {
	var g Ground

	for _, k := range t.Watchlist.Coins() {
		if k.Coin.Asset == asset {
			g.Price, g.DayMove = k.Coin.Price, k.Coin.DayRange
		}
	}

	ticker, listed, err := t.Live.Ticker(ctx, news.Symbol(asset))
	if err == nil && listed && ticker.Price > 0 {
		g.Price = ticker.Price
		g.Smallest = t.Trader.Sized(ctx, trading.Market{Listed: true, Contract: ticker.Contract, Price: ticker.Price}).MinSize
	}

	g.Harm, g.Caused = t.grounds(asset, time.Now().UTC())

	return g, g.Price > 0 && g.DayMove > 0
}

func (t ServerTool) grounds(asset string, now time.Time) (string, bool) {
	harm := ""
	caused := len(t.Research.Of(asset)) > 0

	for _, e := range t.Recent.Latest(eventsForAgents) {
		if !slices.Contains(e.Symbols, news.Symbol(asset)) {
			continue
		}

		caused = true

		if harm == "" && (e.Category == news.ShortTrigger || e.Category == news.Incident) {
			harm = e.Title
		}
	}

	tradfi := t.instrument(asset) == instrumentTradFi

	for _, e := range t.Agenda.Upcoming(now, calendarInCard) {
		if slices.Contains(e.Assets, asset) || (tradfi && len(e.Assets) == 0 && e.Kind != schedule.Expiry) {
			caused = true
		}
	}

	return harm, caused
}
