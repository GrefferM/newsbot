package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/schedule"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

const (
	checkBars   = 1500
	checkRows   = 500
	checkRecent = 12
	checkNews   = 5
	usualBars   = 3 * 288
	checkAhead  = 2 * time.Hour
	barWidth    = 5 * time.Minute
	benchmark   = "BTCUSDT"
)

const (
	sinceEvent = "момент події, якої чекав план"
	sinceArmed = "мить, коли план поставили на стеження"
)

type EntryTool struct {
	Market   Market
	Research Shelf
	Live     Quotes
	Watched  Watched
	Trader   Account
	Recent   Events
	Agenda   Agenda
	Rules    DeskRules
}

func (EntryTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "plan_check",
		Description: "Лист перевірки плану на стеженні одним викликом — усе, що ринок зробив від моменту, з якого план " +
			"чекав (момент його події або мить, коли його поставили на стеження): ціна тоді й зараз, найнижча й найвища " +
			"відтоді, чи перетнула ціна поріг плану і де вона зараз відносно порога, стопу й тейку; обіг відтоді за " +
			"годину проти звичайного й частка ринкових купівель; відкритий інтерес і частка акаунтів у лонг тоді й " +
			"зараз; рух BTC відтоді; фандинг за період і сам період; останні свічки по п'ять хвилин; перевірки входу " +
			"столу для боку й розміру саме цього плану — серед них, чи торгується контракт на майданчику столу й чи " +
			"розмір не менший за найменший ордер біржі; події бота про інструмент відтоді, останні новини про нього, " +
			"які знайшов дослідник, і події загального календаря найближчих годин. Частки — дробом: 0.05 — це 5%.",
		Parameters: object(map[string]any{
			"asset": text("тікер інструмента"),
		}, "asset"),
	}
}

type checkSheet struct {
	Plan         research.Scenario  `json:"plan"`
	Contract     string             `json:"perp_contract"`
	Since        time.Time          `json:"since"`
	SinceIs      string             `json:"since_is"`
	Minutes      int                `json:"minutes_since"`
	PriceThen    float64            `json:"price_then"`
	PriceNow     float64            `json:"price_now"`
	Move         float64            `json:"move_since_fraction"`
	Low          float64            `json:"low_since"`
	High         float64            `json:"high_since"`
	Crossed      bool               `json:"threshold_crossed_since"`
	Past         float64            `json:"now_past_threshold_fraction"`
	ToStop       float64            `json:"now_to_stop_fraction"`
	ToTake       float64            `json:"now_to_take_fraction"`
	VolumeHour   float64            `json:"volume_per_hour_since_usd"`
	VolumeUsual  float64            `json:"volume_per_hour_usual_usd"`
	BuyShare     float64            `json:"taker_buy_share_since_fraction"`
	InterestThen float64            `json:"open_interest_then_usd,omitempty"`
	InterestNow  float64            `json:"open_interest_now_usd,omitempty"`
	LongsThen    float64            `json:"long_accounts_then_fraction,omitempty"`
	LongsNow     float64            `json:"long_accounts_now_fraction,omitempty"`
	Benchmark    float64            `json:"btc_move_since_fraction"`
	Funding      float64            `json:"funding_per_period_fraction"`
	FundingHours int                `json:"funding_period_hours"`
	Bars         []checkBar         `json:"last_5m_bars"`
	Checks       []cardCheck        `json:"entry_checks_for_this_plan"`
	Halted       bool               `json:"new_entries_stopped_by_daily_loss"`
	Events       []cardEvent        `json:"bot_events_since_armed"`
	News         []research.Finding `json:"researcher_news_latest,omitempty"`
	Calendar     []cardEvent        `json:"market_calendar_next_hours"`
	Unread       []string           `json:"not_read,omitempty"`
}

type checkBar struct {
	Open     time.Time `json:"open"`
	Low      float64   `json:"low"`
	High     float64   `json:"high"`
	Close    float64   `json:"close"`
	Volume   float64   `json:"volume_usd"`
	BuyShare float64   `json:"taker_buy_share"`
}

func (t EntryTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Asset string `json:"asset"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	asset := watchlist.Normalize(args.Asset)

	var sheets []checkSheet

	for _, w := range t.Watched.Of(asset) {
		if w.State == autopilot.Armed {
			sheets = append(sheets, t.sheet(ctx, asset, w))
		}
	}

	if len(sheets) == 0 {
		return "", fmt.Errorf("плану на стеженні для %s немає", asset)
	}

	data, err := json.Marshal(sheets)

	return string(data), err
}

func (t EntryTool) sheet(ctx context.Context, asset string, w autopilot.Watch) checkSheet {
	now := time.Now().UTC()
	since, is := w.After, sinceEvent

	if since.IsZero() {
		since, is = w.Armed, sinceArmed
	}

	sheet := checkSheet{Plan: w.Plan, Since: since, SinceIs: is, Minutes: int(now.Sub(since).Minutes())}

	ticker, listed, err := t.Live.Ticker(ctx, news.Symbol(asset))
	if err != nil || !listed {
		sheet.Unread = append(sheet.Unread, "перп не знайдено")

		return sheet
	}

	sheet.Contract, sheet.PriceNow = ticker.Contract, ticker.Price
	sheet.Funding, sheet.FundingHours = ticker.Funding, ticker.FundingHours

	var (
		wg               sync.WaitGroup
		own, bench       []candle
		interest, crowd  []map[string]any
		ownErr, benchErr error
		interestErr      error
		crowdErr         error
	)

	wg.Go(func() {
		own, ownErr = klines(ctx, t.Market, ticker.Contract, "5m", checkBars)
	})

	wg.Go(func() {
		bench, benchErr = klines(ctx, t.Market, benchmark, "5m", checkBars)
	})

	wg.Go(func() {
		interest, interestErr = history(ctx, t.Market, openInterestPath, ticker.Contract, "5m", checkRows)
	})

	wg.Go(func() {
		crowd, crowdErr = history(ctx, t.Market, longAccountsPath, ticker.Contract, "5m", checkRows)
	})

	wg.Wait()

	if ownErr != nil {
		sheet.Unread = append(sheet.Unread, "свічки: "+ownErr.Error())
	} else {
		sheet.priced(own, since)
	}

	if benchErr == nil {
		sheet.Benchmark = moved(bench, since)
	}

	if interestErr == nil {
		sheet.InterestThen, sheet.InterestNow = around(interest, since, "sumOpenInterestValue")
	}

	if crowdErr == nil {
		sheet.LongsThen, sheet.LongsNow = around(crowd, since, "longAccount")
	}

	sheet.desked(t.Trader.Book(), t.Rules.Limits, t.Trader.Sized(ctx, trading.Market{
		Listed:      true,
		Contract:    ticker.Contract,
		Price:       ticker.Price,
		Funding:     ticker.Funding,
		QuoteVolume: ticker.QuoteVolume,
	}), now)
	sheet.Events = t.events(asset, w.Armed)
	sheet.News = t.Research.Of(asset)

	if len(sheet.News) > checkNews {
		sheet.News = sheet.News[:checkNews]
	}

	sheet.Calendar = t.calendar(asset, now)

	return sheet
}

func after(bars []candle, since time.Time) int {
	for i, b := range bars {
		if b.open.Add(barWidth).After(since) {
			return i
		}
	}

	return len(bars)
}

func moved(bars []candle, since time.Time) float64 {
	i := after(bars, since)
	if i >= len(bars) || bars[i].o <= 0 {
		return 0
	}

	return bars[len(bars)-1].c/bars[i].o - 1
}

func (s *checkSheet) priced(bars []candle, since time.Time) {
	i := after(bars, since)
	if i >= len(bars) {
		i = len(bars) - 1
	}

	from := bars[i:]
	s.PriceThen, s.Low, s.High = from[0].o, from[0].l, from[0].h

	if s.PriceNow <= 0 {
		s.PriceNow = bars[len(bars)-1].c
	}

	quote, bought := 0.0, 0.0

	for _, b := range from {
		s.Low, s.High = math.Min(s.Low, b.l), math.Max(s.High, b.h)
		quote += b.quote
		bought += b.boughtUSD
	}

	if s.PriceThen > 0 {
		s.Move = s.PriceNow/s.PriceThen - 1
	}

	if quote > 0 {
		s.BuyShare = bought / quote
	}

	hours := math.Max(float64(len(from))*barWidth.Hours(), barWidth.Hours())
	s.VolumeHour = quote / hours

	before := bars[max(0, i-usualBars):i]

	if len(before) > 0 {
		usual := 0.0

		for _, b := range before {
			usual += b.quote
		}

		s.VolumeUsual = usual / (float64(len(before)) * barWidth.Hours())
	}

	for _, b := range bars[max(0, len(bars)-checkRecent):] {
		share := 0.0

		if b.quote > 0 {
			share = b.boughtUSD / b.quote
		}

		s.Bars = append(s.Bars, checkBar{Open: b.open, Low: b.l, High: b.h, Close: b.c, Volume: b.quote, BuyShare: share})
	}

	s.reached()
}

func (s *checkSheet) reached() {
	p := s.Plan

	switch {
	case p.WakeBelow > 0:
		s.Crossed = s.Low <= p.WakeBelow
		s.Past = p.WakeBelow/s.PriceNow - 1
	case p.WakeAbove > 0:
		s.Crossed = s.High >= p.WakeAbove
		s.Past = s.PriceNow/p.WakeAbove - 1
	}

	if s.PriceNow > 0 {
		s.ToStop = math.Abs(p.Stop/s.PriceNow - 1)
		s.ToTake = math.Abs(p.Take/s.PriceNow - 1)
	}
}

func around(rows []map[string]any, since time.Time, field string) (float64, float64) {
	if len(rows) == 0 {
		return 0, 0
	}

	then := numeric(rows[0][field])

	for _, r := range rows {
		if time.UnixMilli(int64(numeric(r["timestamp"]))).After(since) {
			break
		}

		then = numeric(r[field])
	}

	return then, numeric(rows[len(rows)-1][field])
}

func (s *checkSheet) desked(book trading.Book, limits trading.Limits, m trading.Market, now time.Time) {
	side := trading.Short

	if s.Plan.Long() {
		side = trading.Long
	}

	s.Checks = checked(limits.Checks(m, side, s.Plan.SizePct/100, book.Exposure()))
	s.Halted = book.Halted(limits, now)
}

func (t EntryTool) events(asset string, since time.Time) []cardEvent {
	var out []cardEvent

	for _, e := range t.Recent.Latest(eventsForAgents) {
		if slices.Contains(e.Symbols, news.Symbol(asset)) && e.Time.After(since) {
			out = append(out, cardEvent{Time: e.Time, Kind: e.Kind, Title: e.Title})
		}
	}

	return out
}

func (t EntryTool) calendar(asset string, now time.Time) []cardEvent {
	var out []cardEvent

	for _, e := range t.Agenda.Upcoming(now, checkAhead) {
		if e.Kind == schedule.Expiry || len(e.Assets) == 0 || slices.Contains(e.Assets, asset) {
			out = append(out, cardEvent{Time: e.Time, Kind: e.Kind, Title: e.Title})
		}
	}

	return out
}
