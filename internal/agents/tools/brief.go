package tools

import (
	"cmp"
	"context"
	"encoding/json"
	"math"
	"slices"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/news"
	"newsbot/internal/radar"
	"newsbot/internal/schedule"
)

const (
	briefTrends   = 8
	launchWithin  = 4
	briefEvents   = 30
	briefCalendar = 36 * time.Hour
	eventsInBrief = 8
	markWoke      = "woke_up"
	markLaunch    = "launched_from_quiet"
	markTrend     = "steady_trend"
	longCrowded   = "лонг переповнений"
	shortCrowded  = "шорт переповнений"
)

type BriefTool struct {
	Radar   Radar
	Recent  Events
	Agenda  Agenda
	Crowded float64
}

func (BriefTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "desk_brief",
		Description: "Готове зведення столу про те, що відбувається просто зараз, одним викликом: лише позначені контракти " +
			"— криптомонети й окремо TradFi (сировина, акції й фонди США) — і по кожному лише ключові фактори. marks — " +
			"вердикти столу: launched_from_quiet (за останні години після двох діб майже без руху була годинна свічка в кілька разів " +
			"більша за звичайну на зрослому обсязі; скільки годин тому, на скільки і у скільки разів), woke_up (контракт ожив за останню годину: за п'ять хвилин помітно змінився відкритий інтерес і " +
			"водночас зсунулась частка акаунтів у лонг; woke_up каже, скільки хвилин тому і на скільки), steady_trend (стала тенденція на добу чи три). " +
			"Далі — напрям, скільки хвилин тому почався рух і скільки пройдено відтоді, зміни за 15 хвилин, годину, чотири " +
			"години й добу, обіг години й останньої чверті проти звичайного, частка ринкових покупок, фандинг і чи " +
			"переповнений бік, частка акаунтів у лонг і її зміна за годину (натовп заходить за рухом чи проти), стала " +
			"тенденція з її строком, свіжі новини про сам контракт, уже знайдені столом, події бота про нього і що про " +
			"такі випадки виміряно на історії. Окремо — рух BTC, найближчі події загального календаря й останні події " +
			"бота; поле time скрізь за UTC, а час усередині назви події бота — київський. Порожній список позначених означає, що ринок зараз тихий. Цього зведення досить, щоб відповісти, що " +
			"ожило і де тримається тенденція; глибше — картка монети й інші інструменти.",
		Parameters: object(map[string]any{}),
	}
}

type briefRow struct {
	Asset     string   `json:"asset"`
	Marks     []string `json:"marks"`
	Direction string   `json:"direction"`
	Price     float64  `json:"price_usd"`
	pulseMove
	Change24h    float64 `json:"change_24h_fraction"`
	QuarterSurge float64 `json:"volume_15m_vs_average"`
	Funding      float64 `json:"funding_8h_fraction"`
	Crowded      string  `json:"crowded_side,omitempty"`
	VolumeDay    float64 `json:"volume_24h_usd"`
	pulseMarks
	Found     []radar.Story `json:"researcher_found,omitempty"`
	BotEvents []cardEvent   `json:"bot_events,omitempty"`
	Measured  []string      `json:"measured_on_history,omitempty"`
}

type briefBoard struct {
	Scanned int        `json:"scanned"`
	Marked  []briefRow `json:"marked"`
}

type brief struct {
	At       time.Time        `json:"at"`
	BTCHour  float64          `json:"btc_1h_fraction"`
	Crypto   briefBoard       `json:"crypto"`
	TradFi   briefBoard       `json:"tradfi"`
	Calendar []schedule.Entry `json:"calendar_ahead"`
	Events   []cardEvent      `json:"bot_events_latest"`
}

func (t BriefTool) Call(_ context.Context, _ json.RawMessage, _ *agents.Turn) (string, error) {
	crypto, tradfi := t.Radar.Latest(), t.Radar.LatestTradFi()
	events := t.Recent.Latest(briefEvents)
	now := time.Now().UTC()

	out := brief{
		At:       crypto.At,
		BTCHour:  crypto.Bench,
		Crypto:   t.board(crypto, events),
		TradFi:   t.board(tradfi, events),
		Calendar: t.Agenda.Upcoming(now, briefCalendar),
	}

	for _, e := range events[:min(len(events), eventsInBrief)] {
		out.Events = append(out.Events, cardEvent{Time: e.Time, Kind: e.Kind, Title: e.Title})
	}

	return encode(out)
}

func (t BriefTool) board(board radar.Board, events []news.Event) briefBoard {
	out := briefBoard{Scanned: len(board.Coins), Marked: []briefRow{}}
	trends := 0

	for _, coin := range ranked(board.Coins) {
		if !coin.Active() && coin.Trend == nil {
			continue
		}

		if !coin.Active() {
			trends++
		}

		if coin.Active() || trends <= briefTrends {
			out.Marked = append(out.Marked, t.row(coin, events))
		}
	}

	return out
}

func ranked(coins []radar.Coin) []radar.Coin {
	sorted := slices.Clone(coins)

	slices.SortStableFunc(sorted, func(a, b radar.Coin) int {
		return cmp.Compare(rank(a), rank(b))
	})

	return sorted
}

func rank(c radar.Coin) int {
	switch {
	case c.Launch != nil:
		return c.Launch.Hours
	case c.Wake != nil:
		return launchWithin + c.Wake.Minutes
	default:
		return math.MaxInt
	}
}

func (t BriefTool) row(coin radar.Coin, events []news.Event) briefRow {
	p := coin.Pulse
	row := briefRow{
		Asset:        p.Asset,
		Marks:        marks(coin),
		Direction:    direction(coin),
		Price:        p.Price,
		pulseMove:    moveOf(p),
		Change24h:    p.Change24h,
		QuarterSurge: p.QuarterSurge,
		Funding:      p.Funding,
		Crowded:      t.crowded(p.Funding),
		VolumeDay:    p.VolumeDay,
		pulseMarks:   marksOf(coin.Wake, coin.Launch, coin.Trend),
		Found:        coin.Stories,
		Measured:     measured(coin),
	}

	for _, e := range events {
		if slices.Contains(e.Symbols, news.Symbol(p.Asset)) {
			row.BotEvents = append(row.BotEvents, cardEvent{Time: e.Time, Kind: e.Kind, Title: e.Title})
		}
	}

	return row
}

func marks(coin radar.Coin) []string {
	var out []string

	if coin.Launch != nil {
		out = append(out, markLaunch)
	}

	if coin.Wake != nil {
		out = append(out, markWoke)
	}

	if coin.Trend != nil {
		out = append(out, markTrend)
	}

	return out
}

func direction(coin radar.Coin) string {
	move := coin.Pulse.Change1h

	if !coin.Active() && coin.Trend != nil {
		move = coin.Trend.Move
	}

	if move < 0 {
		return "вниз"
	}

	return "вгору"
}

func (t BriefTool) crowded(funding float64) string {
	limit := math.Abs(t.Crowded)

	switch {
	case limit == 0:
		return ""
	case funding <= -limit:
		return shortCrowded
	case funding >= limit:
		return longCrowded
	default:
		return ""
	}
}
