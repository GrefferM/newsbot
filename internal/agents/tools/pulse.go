package tools

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"newsbot/internal/agents"
	"newsbot/internal/market"
	"newsbot/internal/radar"
)

const (
	pulseRows  = 12
	pulseViews = 8
)

var pulseFields = []string{
	"change_15m_fraction",
	"change_1h_fraction",
	"change_4h_fraction",
	"change_24h_fraction",
	"move_1h_size_fraction",
	"volume_1h_usd",
	"volume_1h_vs_average",
	"taker_buy_share_1h_fraction",
	"range_position_8h",
	"funding_8h_fraction",
	"volume_24h_usd",
}

type Pulses interface {
	Pulse(ctx context.Context) ([]market.Pulse, error)
}

type PulseTool struct {
	Market Pulses
	TradFi Pulses
	Rule   radar.Rule
}

func (PulseTool) Spec() agents.ToolSpec {
	view := map[string]any{
		"min":     map[string]any{"type": "object", "description": "нижні межі: поле → число"},
		"max":     map[string]any{"type": "object", "description": "верхні межі: поле → число"},
		"sort_by": enum(pulseFields...),
		"order":   enum("asc", "desc"),
		"limit":   map[string]any{"type": "integer", "description": "скільки рядків повернути"},
		"woke_up_only": map[string]any{
			"type":        "boolean",
			"description": "лише контракти, що ожили за останню годину (woke_up)",
		},
		"steady_trend_only": map[string]any{
			"type":        "boolean",
			"description": "лише монети зі сталою тенденцією на добу чи три",
		},
	}

	return agents.ToolSpec{
		Name: "market_pulse",
		Description: "Живий пульс усіх криптовалютних безстрокових ф'ючерсів USDT на Binance (відкритий інтерес і частка лонгів — лише в найліквідніших), порахований щойно з " +
			"п'ятихвилинних свічок: ціна, зміна за 15 хвилин, годину, чотири години і 24 години, розмір руху за годину " +
			"без знака і звичайний для цієї монети власний рух за годину (вже без руху BTC), обіг останньої години в доларах і " +
			"проти тієї самої години доби за минулий тиждень (понад 1 — обіг вищий за звичайний), частка ринкових покупок в обігу останньої години (понад половину — купують активніше, ніж " +
			"продають), де ціна в діапазоні останніх восьми годин (0 — на мінімумі, 1 — на максимумі), фандинг, " +
			"приведений до восьми годин (його платить той бік, якого більше: від'ємний — шорти платять лонгам), " +
			"обіг за 24 години, частка акаунтів у лонг і її зміна за годину (crowd, де є). woke_up є лише в контракту, що ожив за " +
			"останню годину: за п'ять хвилин відкритий інтерес помітно змінився і водночас за чверть години помітно " +
			"зсунулась частка акаунтів у лонг — скільки хвилин тому і на скільки (у частках одиниці). " +
			"launched_from_quiet є лише в контракту, який за останні години злетів із тиші: після двох діб майже без " +
			"руху годинна свічка в кілька разів більша за звичайну, на зрослому обсязі — скільки годин тому, на скільки " +
			"і у скільки разів більша за звичайну. " +
			"move_began_minutes_ago і move_since_then_fraction — скільки хвилин тому був край останніх чотирьох " +
			"годин, від якого йде рух у бік години, і скільки ціна пройшла відтоді. " +
			"steady_trend_hours і steady_trend_move_fraction є лише в монети зі сталою тенденцією: її власний рух за добу чи три " +
			"доби значно більший за звичайний для такого строку, йшов досить прямо, і ціна досі біля його краю; поле каже, " +
			"скільки годин він триває і скільки пройдено. " +
			"Це готові вердикти, однакові для всіх контрактів: вони означають, що на контракті " +
			"зараз відбувається незвичне, а не те, куди піде ціна. market=crypto (типово) — криптомонети; market=tradfi — " +
			"сировина й найліквідніші акції та фонди США на TradFi-ф'ючерсах Binance, правило те саме. Один виклик відповідає на кілька зрізів одразу: кожен зріз у views відбирає " +
			fmt.Sprintf("(min, max), сортує (sort_by, order) й обрізає (limit) окремо; зрізів — до %d. Поля: ", pulseViews) +
			strings.Join(pulseFields, ", ") + ".",
		Parameters: object(map[string]any{
			"market": enum("crypto", "tradfi"),
			"views": map[string]any{
				"type":        "array",
				"description": "зрізи пульсу, кожен зі своїм відбором і сортуванням",
				"items":       map[string]any{"type": "object", "properties": view},
			},
		}, "views"),
	}
}

type pulsed struct {
	Asset    string  `json:"asset"`
	Contract string  `json:"perp_contract"`
	Price    float64 `json:"price_usd"`
	pulseMove
	Change24h     float64 `json:"change_24h_fraction"`
	Move1h        float64 `json:"move_1h_size_fraction"`
	QuarterSurge  float64 `json:"volume_15m_vs_average"`
	QuarterBuy    float64 `json:"taker_buy_share_15m_fraction"`
	VolumeHour    float64 `json:"volume_1h_usd"`
	RangePosition float64 `json:"range_position_8h"`
	Funding       float64 `json:"funding_8h_fraction"`
	VolumeDay     float64 `json:"volume_24h_usd"`
	pulseMarks
}

func (r pulsed) value(field string) float64 {
	switch field {
	case "change_15m_fraction":
		return r.Change15m
	case "change_1h_fraction":
		return r.Change1h
	case "change_4h_fraction":
		return r.Change4h
	case "change_24h_fraction":
		return r.Change24h
	case "move_1h_size_fraction":
		return r.Move1h
	case "volume_1h_usd":
		return r.VolumeHour
	case "volume_1h_vs_average":
		return r.VolumeSurge
	case "taker_buy_share_1h_fraction":
		return r.BuyShare
	case "range_position_8h":
		return r.RangePosition
	case "funding_8h_fraction":
		return r.Funding
	default:
		return r.VolumeDay
	}
}

type pulseView struct {
	Min    map[string]float64 `json:"min,omitempty"`
	Max    map[string]float64 `json:"max,omitempty"`
	Only   bool               `json:"woke_up_only,omitempty"`
	Trend  bool               `json:"steady_trend_only,omitempty"`
	SortBy string             `json:"sort_by,omitempty"`
	Order  string             `json:"order,omitempty"`
	Limit  int                `json:"limit,omitempty"`
}

type viewed struct {
	View    pulseView `json:"view"`
	Matched int       `json:"matched"`
	Coins   []pulsed  `json:"coins"`
}

func (t PulseTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Market string      `json:"market"`
		Views  []pulseView `json:"views"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if len(args.Views) == 0 || len(args.Views) > pulseViews {
		return "", fmt.Errorf("потрібно від одного до %d зрізів у views", pulseViews)
	}

	for _, view := range args.Views {
		for _, field := range append(append(keys(view.Min), keys(view.Max)...), view.SortBy) {
			if field != "" && !slices.Contains(pulseFields, field) {
				return "", fmt.Errorf("невідоме поле %q; є: %s", field, strings.Join(pulseFields, ", "))
			}
		}
	}

	source := t.Market

	if args.Market == "tradfi" {
		source = t.TradFi
	}

	pulses, err := source.Pulse(ctx)
	if err != nil {
		return "", err
	}

	rows := make([]pulsed, 0, len(pulses))

	for _, p := range pulses {
		row := pulse(p)

		var (
			wake   *radar.Wake
			launch *radar.Launch
			trend  *market.Run
		)

		if found, woke := t.Rule.Woke(p); woke {
			wake = &found
		}

		if found, launched := t.Rule.Launched(p); launched {
			launch = &found
		}

		if found, steady := radar.Trend(p); steady {
			trend = &found
		}

		row.pulseMarks = marksOf(wake, launch, trend)

		rows = append(rows, row)
	}

	out := make([]viewed, 0, len(args.Views))

	for _, view := range args.Views {
		out = append(out, look(view, rows))
	}

	reply := map[string]any{"scanned": len(pulses), "views": out}

	if args.Market != "tradfi" {
		reply["btc_1h_fraction"] = radar.Bench(pulses)
	}

	return encode(reply)
}

func look(view pulseView, all []pulsed) viewed {
	var rows []pulsed

	for _, row := range all {
		if within(row, view.Min, view.Max) && (row.Woke != nil || !view.Only) && (row.TrendHours > 0 || !view.Trend) {
			rows = append(rows, row)
		}
	}

	if view.SortBy != "" {
		slices.SortStableFunc(rows, func(a, b pulsed) int {
			order := cmp.Compare(a.value(view.SortBy), b.value(view.SortBy))

			if view.Order == "desc" {
				return -order
			}

			return order
		})
	}

	limit := pulseRows

	if view.Limit > 0 {
		limit = min(view.Limit, pulseRows)
	}

	return viewed{View: view, Matched: len(rows), Coins: rows[:min(limit, len(rows))]}
}

func pulse(p market.Pulse) pulsed {
	return pulsed{
		Asset:         p.Asset,
		Contract:      p.Contract,
		Price:         p.Price,
		pulseMove:     moveOf(p),
		Change24h:     p.Change24h,
		Move1h:        math.Abs(p.Change1h),
		QuarterSurge:  p.QuarterSurge,
		QuarterBuy:    p.QuarterBuyShare,
		VolumeHour:    p.VolumeHour,
		RangePosition: p.RangePosition,
		Funding:       p.Funding,
		VolumeDay:     p.VolumeDay,
	}
}
