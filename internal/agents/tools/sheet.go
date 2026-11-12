package tools

import (
	"context"
	"encoding/json"
	"sync"

	"newsbot/internal/agents"
	"newsbot/internal/watchlist"
)

const (
	weekHours = 168
	dayHours  = 24
)

var levelHours = []int{4, dayHours, 72, weekHours}

type SheetTool struct {
	Server ServerTool
	Market Market
}

func (SheetTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "plan_sheet",
		Description: "Лист для планів одним викликом: уся картка інструмента — та сама, що дає server_get (what=coin), — і " +
			"ринок перпа, від якого ставлять межі: найвища й найнижча ціна за 4 години, добу, три доби й тиждень; рух за " +
			"годину й добу; обіг останньої години й доби проти звичайного і частка ринкових купівель у них; відкритий " +
			"інтерес і частка акаунтів у лонг зараз і їхня зміна за годину й добу; рух BTC за годину й добу. Частки — " +
			"дробом: 0.05 — це 5%.",
		Parameters: object(map[string]any{
			"asset": text("тікер інструмента"),
		}, "asset"),
	}
}

type planSheet struct {
	Card   coinCard    `json:"card"`
	Market *planMarket `json:"market,omitempty"`
	Unread []string    `json:"not_read,omitempty"`
}

type planLevel struct {
	Hours int     `json:"hours"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
}

type planMarket struct {
	Levels       []planLevel `json:"price_range"`
	Change1h     float64     `json:"change_1h_fraction"`
	Change24h    float64     `json:"change_24h_fraction"`
	VolumeHour   float64     `json:"volume_last_hour_usd"`
	UsualHour    float64     `json:"volume_usual_hour_usd"`
	VolumeDay    float64     `json:"volume_last_day_usd"`
	UsualDay     float64     `json:"volume_usual_day_usd"`
	BuyHour      float64     `json:"taker_buy_share_1h_fraction"`
	BuyDay       float64     `json:"taker_buy_share_24h_fraction"`
	Interest     float64     `json:"open_interest_usd,omitempty"`
	Interest1h   float64     `json:"open_interest_change_1h_fraction,omitempty"`
	Interest24h  float64     `json:"open_interest_change_24h_fraction,omitempty"`
	Longs        float64     `json:"long_accounts_fraction,omitempty"`
	Longs1h      float64     `json:"long_accounts_shift_1h,omitempty"`
	Longs24h     float64     `json:"long_accounts_shift_24h,omitempty"`
	Benchmark1h  float64     `json:"btc_change_1h_fraction"`
	Benchmark24h float64     `json:"btc_change_24h_fraction"`
}

func (t SheetTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Asset string `json:"asset"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	sheet := planSheet{Card: t.Server.card(ctx, watchlist.Normalize(args.Asset))}

	if sheet.Card.Contract == "" {
		sheet.Unread = append(sheet.Unread, "перп не знайдено: ринку для меж немає")

		return encode(sheet)
	}

	sheet.Market, sheet.Unread = t.market(ctx, sheet.Card.Contract)

	return encode(sheet)
}

func (t SheetTool) market(ctx context.Context, contract string) (*planMarket, []string) {
	var (
		wg                 sync.WaitGroup
		own, bench         []candle
		interest, crowd    []map[string]any
		ownErr, benchErr   error
		interestErr, crErr error
	)

	wg.Go(func() {
		own, ownErr = klines(ctx, t.Market, contract, "1h", weekHours+1)
	})

	wg.Go(func() {
		bench, benchErr = klines(ctx, t.Market, benchmark, "1h", dayHours+1)
	})

	wg.Go(func() {
		interest, interestErr = history(ctx, t.Market, openInterestPath, contract, "1h", dayHours+1)
	})

	wg.Go(func() {
		crowd, crErr = history(ctx, t.Market, longAccountsPath, contract, "1h", dayHours+1)
	})

	wg.Wait()

	if ownErr != nil {
		return nil, []string{"свічки: " + ownErr.Error()}
	}

	m := &planMarket{}

	m.ranged(own)

	var unread []string

	if benchErr == nil {
		m.Benchmark1h, m.Benchmark24h = changes(bench)
	}

	if interestErr == nil && len(interest) > 1 {
		m.Interest, m.Interest1h, m.Interest24h = drift(interest, "sumOpenInterestValue", true)
	} else {
		unread = append(unread, "відкритий інтерес")
	}

	if crErr == nil && len(crowd) > 1 {
		m.Longs, m.Longs1h, m.Longs24h = drift(crowd, "longAccount", false)
	} else {
		unread = append(unread, "частка акаунтів у лонг")
	}

	return m, unread
}

func (m *planMarket) ranged(bars []candle) {
	for _, hours := range levelHours {
		from := bars[max(0, len(bars)-hours):]
		level := planLevel{Hours: hours, High: from[0].h, Low: from[0].l}

		for _, b := range from {
			level.High, level.Low = max(level.High, b.h), min(level.Low, b.l)
		}

		m.Levels = append(m.Levels, level)
	}

	m.Change1h, m.Change24h = changes(bars)

	last := bars[len(bars)-1]
	m.VolumeHour, m.BuyHour = last.quote, share(bars[len(bars)-1:])

	before := bars[:len(bars)-1]

	if len(before) > 0 {
		m.UsualHour = sum(before) / float64(len(before))
	}

	day := bars[max(0, len(bars)-dayHours):]
	m.VolumeDay, m.BuyDay = sum(day), share(day)

	if earlier := bars[:max(0, len(bars)-dayHours)]; len(earlier) > 0 {
		m.UsualDay = sum(earlier) / float64(len(earlier)) * dayHours
	}
}

func changes(bars []candle) (float64, float64) {
	last := bars[len(bars)-1].c
	hour, day := 0.0, 0.0

	if len(bars) > 1 && bars[len(bars)-2].c > 0 {
		hour = last/bars[len(bars)-2].c - 1
	}

	if from := bars[max(0, len(bars)-dayHours-1)].c; from > 0 {
		day = last/from - 1
	}

	return hour, day
}

func sum(bars []candle) float64 {
	total := 0.0

	for _, b := range bars {
		total += b.quote
	}

	return total
}

func share(bars []candle) float64 {
	quote, bought := 0.0, 0.0

	for _, b := range bars {
		quote += b.quote
		bought += b.boughtUSD
	}

	if quote <= 0 {
		return 0
	}

	return bought / quote
}

func drift(rows []map[string]any, field string, relative bool) (float64, float64, float64) {
	now := numeric(rows[len(rows)-1][field])
	hour := numeric(rows[len(rows)-2][field])
	day := numeric(rows[0][field])

	if !relative {
		return now, now - hour, now - day
	}

	if hour <= 0 || day <= 0 {
		return now, 0, 0
	}

	return now, now/hour - 1, now/day - 1
}
