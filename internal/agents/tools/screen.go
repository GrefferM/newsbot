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
	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/watchlist"
)

const (
	screenRows = 30
	freshNews  = 48 * time.Hour
)

var screenFields = []string{
	"perp_volume_30d_average_usd",
	"perp_volume_last_day_usd",
	"perp_volume_last_day_vs_average",
	"perp_change_24h_fraction",
	"change_1d_fraction",
	"change_7d_fraction",
	"change_90d_vs_btc_fraction",
	"mean_day_range_30d_fraction",
	"perp_taker_sell_share_1d_fraction",
	"perp_taker_sell_share_7d_fraction",
	"funding_8h_fraction",
	"open_interest_usd",
	"perp_depth_within_1pct_usd",
}

type Quotes interface {
	Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error)
}

type ScreenTool struct {
	Watchlist Coins
	Research  Shelf
	Live      Quotes
}

func (ScreenTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "market_screen",
		Description: "Готовий зріз усіх криптомонет із безстроковим ф'ючерсом Binance, зі скану бота за денними свічками перпа: позначка " +
			"Monitoring Tag і середній обіг споту за 30 днів (обидва є лише в монет зі спотовою парою), обіг перпа і його останню добу проти середнього, ціна перпа й зміна за останні 24 години наживо, зміни за " +
			"останній закритий день UTC і за сім закритих днів, тренд до BTC, середній денний діапазон за 30 днів, " +
			"частка ринкових продажів у обігу перпа за добу і за тиждень (понад половину — продають активніше, " +
			"ніж купують), фандинг, приведений до восьми годин, з його справжнім періодом нарахування, відкритий інтерес, глибина книги перпа в межах 1% від ціни, і скільки новин про монету " +
			"вже збережено та коли остання. Один виклик відбирає, " +
			"сортує й обрізає список: only_tagged лишає монети з позначкою, fresh_news_only — монети, про які дослідник " +
			"передав новину за останні дві доби, min і max задають межі для полів, " +
			"sort_by і order сортують, limit бере перші рядки. Поля для min, max і sort_by: " +
			strings.Join(screenFields, ", ") + ".",
		Parameters: object(map[string]any{
			"only_tagged":     map[string]any{"type": "boolean", "description": "true — лише монети з Monitoring Tag"},
			"fresh_news_only": map[string]any{"type": "boolean", "description": "true — лише монети зі свіжою новиною дослідника"},
			"min":             map[string]any{"type": "object", "description": "нижні межі: поле → число"},
			"max":             map[string]any{"type": "object", "description": "верхні межі: поле → число"},
			"sort_by":         enum(screenFields...),
			"order":           enum("asc", "desc"),
			"limit":           map[string]any{"type": "integer", "description": "скільки рядків повернути"},
		}),
	}
}

type screened struct {
	Asset         string  `json:"asset"`
	Contract      string  `json:"perp_contract"`
	MonitoringTag bool    `json:"monitoring_tag"`
	Price         float64 `json:"perp_price_last_close_usd"`
	PriceNow      float64 `json:"perp_price_now_usd"`
	Change24h     float64 `json:"perp_change_24h_fraction"`
	Volume        float64 `json:"perp_volume_30d_average_usd"`
	VolumeDay     float64 `json:"perp_volume_last_day_usd"`
	VolumeRatio   float64 `json:"perp_volume_last_day_vs_average"`
	Change1d      float64 `json:"change_1d_fraction"`
	Change7d      float64 `json:"change_7d_fraction"`
	TrendVsBTC    float64 `json:"change_90d_vs_btc_fraction"`
	DayRange      float64 `json:"mean_day_range_30d_fraction"`
	SellDay       float64 `json:"perp_taker_sell_share_1d_fraction"`
	SellWeek      float64 `json:"perp_taker_sell_share_7d_fraction"`
	Funding       float64 `json:"funding_8h_fraction"`
	FundingHours  int     `json:"funding_paid_every_hours"`
	OpenInterest  float64 `json:"open_interest_usd"`
	Depth         float64 `json:"perp_depth_within_1pct_usd"`
	SpotVolume    float64 `json:"spot_volume_30d_average_usd,omitempty"`
	KeptNews      int     `json:"kept_news"`
	LatestNews    string  `json:"latest_news_published,omitempty"`
}

func (r screened) value(field string) float64 {
	switch field {
	case "perp_volume_30d_average_usd":
		return r.Volume
	case "perp_volume_last_day_usd":
		return r.VolumeDay
	case "perp_volume_last_day_vs_average":
		return r.VolumeRatio
	case "perp_change_24h_fraction":
		return r.Change24h
	case "change_1d_fraction":
		return r.Change1d
	case "change_7d_fraction":
		return r.Change7d
	case "change_90d_vs_btc_fraction":
		return r.TrendVsBTC
	case "mean_day_range_30d_fraction":
		return r.DayRange
	case "perp_taker_sell_share_1d_fraction":
		return r.SellDay
	case "perp_taker_sell_share_7d_fraction":
		return r.SellWeek
	case "funding_8h_fraction":
		return r.Funding
	case "open_interest_usd":
		return r.OpenInterest
	default:
		return r.Depth
	}
}

func (t ScreenTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		OnlyTagged bool               `json:"only_tagged"`
		FreshNews  bool               `json:"fresh_news_only"`
		Min        map[string]float64 `json:"min"`
		Max        map[string]float64 `json:"max"`
		SortBy     string             `json:"sort_by"`
		Order      string             `json:"order"`
		Limit      int                `json:"limit"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	for _, field := range append(append(keys(args.Min), keys(args.Max)...), args.SortBy) {
		if field != "" && !slices.Contains(screenFields, field) {
			return "", fmt.Errorf("невідоме поле %q; є: %s", field, strings.Join(screenFields, ", "))
		}
	}

	var rows []screened

	since := time.Now().Add(-freshNews)

	for _, k := range t.Watchlist.Coins() {
		row := t.live(ctx, screen(k.Coin))
		fresh := false

		if kept := t.Research.Of(row.Asset); len(kept) > 0 {
			row.KeptNews = len(kept)
			row.LatestNews = kept[0].Dated()
			fresh = research.Moment(kept[0].Published).After(since)
		}

		if args.OnlyTagged && !row.MonitoringTag || args.FreshNews && !fresh {
			continue
		}

		if within(row, args.Min, args.Max) {
			rows = append(rows, row)
		}
	}

	if args.SortBy != "" {
		slices.SortStableFunc(rows, func(a, b screened) int {
			order := cmp.Compare(a.value(args.SortBy), b.value(args.SortBy))

			if args.Order == "desc" {
				return -order
			}

			return order
		})
	}

	limit := screenRows

	if args.Limit > 0 {
		limit = args.Limit
	}

	return encode(map[string]any{"matched": len(rows), "coins": rows[:min(limit, len(rows))]})
}

func screen(c watchlist.Coin) screened {
	ratio := 0.0

	if c.Volume > 0 {
		ratio = c.VolumeDay / c.Volume
	}

	return screened{
		VolumeRatio:   ratio,
		Asset:         c.Asset,
		Contract:      c.Contract,
		MonitoringTag: c.Tagged,
		Price:         c.Price,
		Volume:        c.Volume,
		VolumeDay:     c.VolumeDay,
		Change1d:      c.Change1d,
		Change7d:      c.Change7d,
		TrendVsBTC:    c.TrendVsBTC,
		DayRange:      c.DayRange,
		SellDay:       c.SellDay,
		SellWeek:      c.SellWeek,
		Funding:       c.Funding,
		FundingHours:  c.FundingHours,
		OpenInterest:  c.OpenInterest,
		Depth:         c.Depth,
		SpotVolume:    c.SpotVolume,
	}
}

func (t ScreenTool) live(ctx context.Context, row screened) screened {
	ticker, listed, err := t.Live.Ticker(ctx, news.Symbol(row.Asset))
	if err != nil || !listed {
		return row
	}

	row.PriceNow = ticker.Price
	row.Change24h = ticker.Change24h / 100
	row.Funding = ticker.Funding
	row.FundingHours = ticker.FundingHours

	return row
}

type valued interface {
	value(field string) float64
}

func within(row valued, lows, highs map[string]float64) bool {
	for field, low := range lows {
		if row.value(field) < low {
			return false
		}
	}

	for field, high := range highs {
		if row.value(field) > high {
			return false
		}
	}

	return true
}

func keys(m map[string]float64) []string {
	out := make([]string, 0, len(m))

	for k := range m {
		out = append(out, k)
	}

	return out
}
