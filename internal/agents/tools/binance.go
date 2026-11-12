package tools

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"newsbot/internal/agents"
	"newsbot/internal/binance"
	"newsbot/internal/watchlist"
)

type Market interface {
	Get(ctx context.Context, market, address string) ([]byte, error)
}

type Announcements interface {
	About(ctx context.Context, asset string) ([]binance.Notice, error)
}

type ReadTool struct {
	Market Market
}

func (t ReadTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "binance_get",
		Description: "GET-запит до публічного REST API Binance без ключа. market=spot — api.binance.com " +
			"(/api/v3/…), market=futures — fapi.binance.com (/fapi/v1/…, /fapi/v2/…, /futures/data/…). " +
			"Відповідь — JSON Binance. Якщо відповідь — список, його можна стиснути на сервері: fields лишає " +
			"потрібні поля, sort_by і order сортують за числовим полем, limit бере перші записи. Ставка фандингу " +
			"в /fapi/v1/premiumIndex — за один період контракту, а він у різних контрактів 1, 4 або 8 годин: " +
			"період дає /fapi/v1/fundingInfo. Позиціонування в перпі за останні 30 днів з кроком від 5 хвилин до доби " +
			"(period 5m…1d) дає /futures/data/…: openInterestHist — відкритий інтерес, globalLongShortAccountRatio — " +
			"частка акаунтів у лонг і шорт, topLongShortPositionRatio — позиції найбільших трейдерів, takerlongshortRatio — " +
			"ринкові купівлі проти продажів.",
		Parameters: object(map[string]any{
			"market":  enum("spot", "futures"),
			"path":    text("шлях, наприклад /api/v3/ticker/24hr"),
			"query":   map[string]any{"type": "object", "description": "параметри запиту, наприклад {\"symbol\":\"BTCUSDT\"}"},
			"fields":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"sort_by": text("числове поле для сортування списку"),
			"order":   enum("asc", "desc"),
			"limit":   map[string]any{"type": "integer", "description": "скільки записів списку лишити"},
		}, "market", "path"),
	}
}

func (t ReadTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Market string         `json:"market"`
		Path   string         `json:"path"`
		Query  map[string]any `json:"query"`
		Shape  shape
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if err := json.Unmarshal(raw, &args.Shape); err != nil {
		return "", err
	}

	body, err := t.Market.Get(ctx, args.Market, withQuery(args.Path, args.Query))
	if err != nil {
		return "", err
	}

	return args.Shape.apply(body)
}

func withQuery(path string, extra map[string]any) string {
	if len(extra) == 0 {
		return path
	}

	query := url.Values{}

	for k, v := range extra {
		query.Set(k, fmt.Sprint(v))
	}

	return path + "?" + query.Encode()
}

type shape struct {
	Fields []string `json:"fields"`
	SortBy string   `json:"sort_by"`
	Order  string   `json:"order"`
	Limit  int      `json:"limit"`
}

func (s shape) apply(body []byte) (string, error) {
	var rows []map[string]any

	if json.Unmarshal(body, &rows) != nil || (len(s.Fields) == 0 && s.SortBy == "" && s.Limit == 0) {
		return string(body), nil
	}

	if s.SortBy != "" {
		slices.SortStableFunc(rows, func(a, b map[string]any) int {
			order := cmp.Compare(numeric(a[s.SortBy]), numeric(b[s.SortBy]))

			if s.Order == "desc" {
				return -order
			}

			return order
		})
	}

	if s.Limit > 0 && s.Limit < len(rows) {
		rows = rows[:s.Limit]
	}

	if len(s.Fields) > 0 {
		for i, row := range rows {
			kept := make(map[string]any, len(s.Fields))

			for _, f := range s.Fields {
				if v, ok := row[f]; ok {
					kept[f] = v
				}
			}

			rows[i] = kept
		}
	}

	data, err := json.Marshal(rows)

	return string(data), err
}

func numeric(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case string:
		value, _ := strconv.ParseFloat(n, 64)

		return value
	default:
		return 0
	}
}

type NoticesTool struct {
	Notices Announcements
}

func (NoticesTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "binance_announcements",
		Description: "Офіційні анонси Binance напряму з біржі: останні заголовки розділів delisting " +
			"(делістинги, позначки, зняття пар), news (новини біржі) і listing (нові лістинги) з датою " +
			"публікації й адресою. symbol лишає тільки анонси про цю монету: ті, що називають її в заголовку, " +
			"і свіжі анонси розділу delisting, що називають її лише в тексті, — з уривком, де її згадано.",
		Parameters: object(map[string]any{
			"symbol": text("тікер монети, наприклад MOVE"),
		}),
	}
}

func (t NoticesTool) Call(ctx context.Context, raw json.RawMessage, _ *agents.Turn) (string, error) {
	var args struct {
		Symbol string `json:"symbol"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	notices, err := t.Notices.About(ctx, watchlist.Normalize(args.Symbol))
	if err != nil {
		return "", err
	}

	return encode(notices)
}
