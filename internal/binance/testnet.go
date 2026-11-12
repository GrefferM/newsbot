package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"newsbot/internal/trading"
)

const (
	testnetURL     = "https://testnet.binancefuture.com"
	testnetVenue   = "testnet"
	testnetWindow  = "5000"
	testnetTimeout = 20 * time.Second
	testnetBody    = 1 << 20
	incomeMargin   = 2 * time.Second
	incomeRows     = "1000"
)

type Testnet struct {
	http   *http.Client
	base   string
	key    string
	secret string
	mu     sync.Mutex
	rules  map[string]contractRules
}

func NewTestnet(key, secret string) *Testnet {
	return &Testnet{
		http:   &http.Client{Timeout: testnetTimeout},
		base:   testnetURL,
		key:    key,
		secret: secret,
	}
}

func (t *Testnet) Venue() string {
	return testnetVenue
}

func (t *Testnet) Balance(ctx context.Context) (float64, error) {
	var account struct {
		Margin string `json:"totalMarginBalance"`
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v2/account", url.Values{}, &account); err != nil {
		return 0, err
	}

	return parse(account.Margin), nil
}

func (t *Testnet) Earned(ctx context.Context, p trading.Position) (float64, error) {
	var incomes []struct {
		Kind   string `json:"incomeType"`
		Amount string `json:"income"`
	}

	params := url.Values{
		"symbol":    {p.Contract},
		"startTime": {strconv.FormatInt(p.Opened.Add(-incomeMargin).UnixMilli(), 10)},
		"limit":     {incomeRows},
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v1/income", params, &incomes); err != nil {
		return 0, err
	}

	earned, realised := 0.0, false

	for _, income := range incomes {
		switch income.Kind {
		case "REALIZED_PNL":
			earned += parse(income.Amount)
			realised = true
		case "COMMISSION", "FUNDING_FEE":
			earned += parse(income.Amount)
		}
	}

	if !realised {
		return 0, errors.New("the exchange has not booked the result yet")
	}

	return earned, nil
}

func (t *Testnet) Smallest(ctx context.Context, contract string, price float64) (float64, error) {
	rules, err := t.contract(ctx, contract)
	if err != nil {
		return 0, err
	}

	return rules.smallest(price), nil
}

func (t *Testnet) Enter(ctx context.Context, p trading.Position) (trading.Fill, error) {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return trading.Fill{}, err
	}

	quantity := floorTo(p.Stake/p.Entry, rules.step)

	if quantity*p.Entry < rules.minNotional {
		return trading.Fill{}, fmt.Errorf("%s: $%.2f менше мінімуму $%.2f", p.Contract, quantity*p.Entry, rules.minNotional)
	}

	if _, err := t.send(ctx, http.MethodPost, "/fapi/v1/leverage", url.Values{
		"symbol":   {p.Contract},
		"leverage": {"1"},
	}); err != nil {
		return trading.Fill{}, err
	}

	return t.market(ctx, url.Values{
		"symbol":           {p.Contract},
		"side":             {opening(p.Side)},
		"type":             {"MARKET"},
		"quantity":         {decimal(quantity, rules.step)},
		"newOrderRespType": {"RESULT"},
	})
}

func (t *Testnet) Protect(ctx context.Context, p trading.Position) error {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return err
	}

	for kind, price := range map[string]float64{"STOP_MARKET": p.Stop, "TAKE_PROFIT_MARKET": p.Target} {
		if _, err := t.send(ctx, http.MethodPost, "/fapi/v1/algoOrder", url.Values{
			"algoType":      {"CONDITIONAL"},
			"symbol":        {p.Contract},
			"side":          {closing(p.Side)},
			"type":          {kind},
			"triggerPrice":  {decimal(roundTo(price, rules.tick), rules.tick)},
			"closePosition": {"true"},
			"workingType":   {"MARK_PRICE"},
		}); err != nil {
			return fmt.Errorf("%s: %w", kind, err)
		}
	}

	return nil
}

func (t *Testnet) Exited(ctx context.Context, p trading.Position) (trading.Exit, bool, error) {
	var risks []struct {
		Amount string `json:"positionAmt"`
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v2/positionRisk", url.Values{"symbol": {p.Contract}}, &risks); err != nil {
		return trading.Exit{}, false, err
	}

	for _, r := range risks {
		if parse(r.Amount) != 0 {
			return trading.Exit{}, false, nil
		}
	}

	price, err := t.lastClose(ctx, p)
	if err != nil {
		return trading.Exit{}, false, err
	}

	if err := t.cancelAll(ctx, p.Contract); err != nil {
		return trading.Exit{}, false, err
	}

	return trading.Exit{Price: price, Reason: exitReason(p, price)}, true, nil
}

func (t *Testnet) Close(ctx context.Context, p trading.Position) (float64, error) {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return 0, err
	}

	if err := t.cancelAll(ctx, p.Contract); err != nil {
		return 0, err
	}

	fill, err := t.market(ctx, url.Values{
		"symbol":           {p.Contract},
		"side":             {closing(p.Side)},
		"type":             {"MARKET"},
		"quantity":         {decimal(p.Quantity, rules.step)},
		"reduceOnly":       {"true"},
		"newOrderRespType": {"RESULT"},
	})

	return fill.Price, err
}

type rawOrder struct {
	ID       int64  `json:"orderId"`
	AvgPrice string `json:"avgPrice"`
	Executed string `json:"executedQty"`
	Quote    string `json:"cumQuote"`
}

func (o rawOrder) fill() trading.Fill {
	quantity := parse(o.Executed)
	price := parse(o.AvgPrice)

	if price <= 0 && quantity > 0 {
		price = parse(o.Quote) / quantity
	}

	return trading.Fill{Price: price, Quantity: quantity}
}

func (t *Testnet) market(ctx context.Context, params url.Values) (trading.Fill, error) {
	symbol := params.Get("symbol")

	var order rawOrder

	if err := t.call(ctx, http.MethodPost, "/fapi/v1/order", params, &order); err != nil {
		return trading.Fill{}, err
	}

	fill := order.fill()

	if fill.Price <= 0 && order.ID != 0 {
		if err := t.call(ctx, http.MethodGet, "/fapi/v1/order", url.Values{
			"symbol":  {symbol},
			"orderId": {strconv.FormatInt(order.ID, 10)},
		}, &order); err != nil {
			return trading.Fill{}, err
		}

		fill = order.fill()
	}

	if fill.Price <= 0 || fill.Quantity <= 0 {
		return trading.Fill{}, fmt.Errorf("order %d not filled: price %q, quantity %q", order.ID, order.AvgPrice, order.Executed)
	}

	return fill, nil
}

func (t *Testnet) lastClose(ctx context.Context, p trading.Position) (float64, error) {
	var trades []struct {
		Side  string `json:"side"`
		Price string `json:"price"`
		Time  int64  `json:"time"`
	}

	params := url.Values{
		"symbol":    {p.Contract},
		"startTime": {strconv.FormatInt(p.Opened.UnixMilli(), 10)},
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v1/userTrades", params, &trades); err != nil {
		return 0, err
	}

	for i := len(trades) - 1; i >= 0; i-- {
		if trades[i].Side == closing(p.Side) {
			return parse(trades[i].Price), nil
		}
	}

	return 0, errors.New("position closed without a closing trade")
}

func (t *Testnet) cancelAll(ctx context.Context, contract string) error {
	for _, path := range []string{"/fapi/v1/algoOpenOrders", "/fapi/v1/allOpenOrders"} {
		if _, err := t.send(ctx, http.MethodDelete, path, url.Values{"symbol": {contract}}); err != nil {
			return fmt.Errorf("cancel %s: %w", path, err)
		}
	}

	return nil
}

func (t *Testnet) contract(ctx context.Context, contract string) (contractRules, error) {
	t.mu.Lock()
	rules, ok := t.rules[contract]
	loaded := t.rules != nil
	t.mu.Unlock()

	if ok {
		return rules, nil
	}

	if loaded {
		return contractRules{}, fmt.Errorf("%s не торгується на testnet", contract)
	}

	all, err := t.exchangeRules(ctx)
	if err != nil {
		return contractRules{}, err
	}

	t.mu.Lock()
	t.rules = all
	t.mu.Unlock()

	return t.contract(ctx, contract)
}

func (t *Testnet) exchangeRules(ctx context.Context) (map[string]contractRules, error) {
	var info exchangeInfo

	if err := t.do(ctx, http.MethodGet, t.base+"/fapi/v1/exchangeInfo", false, &info); err != nil {
		return nil, err
	}

	return info.rules(), nil
}

func (t *Testnet) send(ctx context.Context, method, path string, params url.Values) (json.RawMessage, error) {
	var raw json.RawMessage

	err := t.call(ctx, method, path, params, &raw)

	return raw, err
}

func (t *Testnet) call(ctx context.Context, method, path string, params url.Values, v any) error {
	params.Set("recvWindow", testnetWindow)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	query := params.Encode()
	endpoint := t.base + path + "?" + query + "&signature=" + sign(t.secret, query)

	return t.do(ctx, method, endpoint, true, v)
}

func (t *Testnet) do(ctx context.Context, method, endpoint string, signed bool, v any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return err
	}

	if signed {
		req.Header.Set("X-MBX-APIKEY", t.key)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, testnetBody))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return apiError(resp.StatusCode, body)
	}

	return json.Unmarshal(body, v)
}

func apiError(status int, body []byte) error {
	var e struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if json.Unmarshal(body, &e) == nil && e.Msg != "" {
		return fmt.Errorf("testnet %d: %s (%d)", status, e.Msg, e.Code)
	}

	return fmt.Errorf("testnet HTTP %d", status)
}

func opening(side trading.Side) string {
	if side == trading.Long {
		return "BUY"
	}

	return "SELL"
}

func closing(side trading.Side) string {
	if side == trading.Long {
		return "SELL"
	}

	return "BUY"
}

func exitReason(p trading.Position, price float64) string {
	if reason, hit := p.Triggered(price); hit {
		return reason
	}

	if math.Abs(price-p.Stop) < math.Abs(price-p.Target) {
		return trading.ExitStop
	}

	return trading.ExitTarget
}
