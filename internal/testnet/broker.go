package testnet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"newsbot/internal/trading"
)

const (
	baseURL        = "https://testnet.binancefuture.com"
	venueName      = "testnet"
	receiveWindow  = "5000"
	requestTimeout = 20 * time.Second
	bodyLimit      = 1 << 20
	incomeMargin   = 2 * time.Second
	incomeRows     = "1000"
	stopTag        = "sl"
	targetTag      = "tp"
	shortestHold   = time.Minute
	firstProbe     = 2 * time.Minute
)

var (
	errUnfilled = errors.New("not filled")
	bannedUntil = regexp.MustCompile(`banned until (\d{13})`)
)

type Broker struct {
	http    *http.Client
	base    string
	stream  string
	key     string
	secret  string
	mu      sync.Mutex
	rules   map[string]contractRules
	held    time.Time
	levered map[string]int
	hedged  bool
	probe   time.Duration
}

func New(key, secret string) *Broker {
	return &Broker{
		http:    &http.Client{Timeout: requestTimeout},
		base:    baseURL,
		stream:  streamURL,
		levered: map[string]int{},
		key:     key,
		secret:  secret,
	}
}

func (t *Broker) Venue() string {
	return venueName
}

func (t *Broker) Balance(ctx context.Context) (float64, error) {
	var account struct {
		Margin string `json:"totalMarginBalance"`
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v2/account", url.Values{}, &account); err != nil {
		return 0, err
	}

	return parse(account.Margin), nil
}

func (t *Broker) Earned(ctx context.Context, p trading.Position) (float64, error) {
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

func (t *Broker) Smallest(ctx context.Context, contract string, price float64) (float64, error) {
	rules, err := t.contract(ctx, contract)
	if err != nil {
		return 0, err
	}

	return rules.smallest(price), nil
}

func (t *Broker) Enter(ctx context.Context, p trading.Position) (trading.Fill, error) {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return trading.Fill{}, err
	}

	quantity := floorTo(p.Stake/p.Entry, rules.step)

	if quantity*p.Entry < rules.minNotional {
		return trading.Fill{}, fmt.Errorf("%s: $%.2f менше мінімуму $%.2f", p.Contract, quantity*p.Entry, rules.minNotional)
	}

	if err := t.prepare(ctx, p); err != nil {
		return trading.Fill{}, err
	}

	params := url.Values{
		"symbol":           {p.Contract},
		"side":             {opening(p.Side)},
		"positionSide":     {holding(p.Side)},
		"type":             {"MARKET"},
		"quantity":         {decimal(quantity, rules.step)},
		"newOrderRespType": {"RESULT"},
	}

	if p.Limit > 0 {
		params.Set("type", "LIMIT")
		params.Set("timeInForce", "IOC")
		params.Set("price", decimal(roundTo(p.Limit, rules.tick), rules.tick))
	}

	fill, err := t.market(ctx, params)
	if p.Limit > 0 && errors.Is(err, errUnfilled) {
		return trading.Fill{}, trading.ErrMissed
	}

	return fill, err
}

func (t *Broker) Protect(ctx context.Context, p trading.Position) error {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return err
	}

	guards := []struct {
		kind  string
		tag   string
		price float64
	}{
		{"STOP_MARKET", stopTag, p.Stop},
		{"TAKE_PROFIT_MARKET", targetTag, p.Target},
	}

	for _, g := range guards {
		if _, err := t.send(ctx, http.MethodPost, "/fapi/v1/algoOrder", url.Values{
			"algoType":      {"CONDITIONAL"},
			"symbol":        {p.Contract},
			"side":          {closing(p.Side)},
			"positionSide":  {holding(p.Side)},
			"type":          {g.kind},
			"triggerPrice":  {decimal(roundTo(g.price, rules.tick), rules.tick)},
			"closePosition": {"true"},
			"workingType":   {"MARK_PRICE"},
			"clientAlgoId":  {guardID(g.tag, p)},
		}); err != nil {
			return fmt.Errorf("%s: %w", g.kind, err)
		}
	}

	return nil
}

func (t *Broker) Reprotect(ctx context.Context, was, next trading.Position) error {
	if err := t.unguard(ctx, was); err != nil {
		return fmt.Errorf("cancel the stop and the target: %w", err)
	}

	return t.Protect(ctx, next)
}

func (t *Broker) unguard(ctx context.Context, p trading.Position) error {
	for _, tag := range []string{stopTag, targetTag} {
		_, err := t.send(ctx, http.MethodDelete, "/fapi/v1/algoOrder", url.Values{
			"symbol":       {p.Contract},
			"clientAlgoId": {guardID(tag, p)},
		})
		if err != nil && !strings.Contains(err.Error(), "(-2011)") {
			return err
		}
	}

	return nil
}

func guardID(tag string, p trading.Position) string {
	sum := sha256.Sum256([]byte(p.ID))

	return fmt.Sprintf("%s-%s-%d", tag, hex.EncodeToString(sum[:8]), p.Guard)
}

func (t *Broker) Exited(ctx context.Context, p trading.Position) (trading.Exit, bool, error) {
	var risks []struct {
		Amount string `json:"positionAmt"`
		Side   string `json:"positionSide"`
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v2/positionRisk", url.Values{"symbol": {p.Contract}}, &risks); err != nil {
		return trading.Exit{}, false, err
	}

	for _, r := range risks {
		if r.Side == holding(p.Side) && parse(r.Amount) != 0 {
			return trading.Exit{}, false, nil
		}
	}

	price, err := t.lastClose(ctx, p)
	if err != nil {
		return trading.Exit{}, false, err
	}

	if err := t.unguard(ctx, p); err != nil {
		return trading.Exit{}, false, err
	}

	return trading.Exit{Price: price, Reason: exitReason(p, price)}, true, nil
}

func (t *Broker) Close(ctx context.Context, p trading.Position) (float64, error) {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return 0, err
	}

	if err := t.unguard(ctx, p); err != nil {
		return 0, err
	}

	fill, err := t.market(ctx, url.Values{
		"symbol":           {p.Contract},
		"side":             {closing(p.Side)},
		"positionSide":     {holding(p.Side)},
		"type":             {"MARKET"},
		"quantity":         {decimal(p.Quantity, rules.step)},
		"newOrderRespType": {"RESULT"},
	})

	return fill.Price, err
}

func (t *Broker) Rest(ctx context.Context, p trading.Position, tag string) (string, error) {
	rules, err := t.contract(ctx, p.Contract)
	if err != nil {
		return "", err
	}

	price := floorTo(p.Limit, rules.tick)
	if p.Side == trading.Short {
		price = math.Ceil(p.Limit/rules.tick) * rules.tick
	}

	quantity := floorTo(p.Stake/price, rules.step)

	if quantity*price < rules.minNotional {
		return "", fmt.Errorf("%s: $%.2f менше мінімуму $%.2f", p.Contract, quantity*price, rules.minNotional)
	}

	if err := t.prepare(ctx, p); err != nil {
		return "", err
	}

	var order rawOrder

	if err := t.call(ctx, http.MethodPost, "/fapi/v1/order", url.Values{
		"symbol":           {p.Contract},
		"side":             {opening(p.Side)},
		"positionSide":     {holding(p.Side)},
		"type":             {"LIMIT"},
		"timeInForce":      {"GTC"},
		"price":            {decimal(price, rules.tick)},
		"quantity":         {decimal(quantity, rules.step)},
		"newClientOrderId": {tag},
	}, &order); err != nil {
		return "", err
	}

	return strconv.FormatInt(order.ID, 10), nil
}

func (t *Broker) prepare(ctx context.Context, p trading.Position) error {
	if err := t.hedge(ctx); err != nil {
		return err
	}

	return t.lever(ctx, p.Contract, max(1, p.Leverage))
}

func (t *Broker) hedge(ctx context.Context) error {
	t.mu.Lock()
	done := t.hedged
	t.mu.Unlock()

	if done {
		return nil
	}

	var mode struct {
		Dual bool `json:"dualSidePosition"`
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v1/positionSide/dual", url.Values{}, &mode); err != nil {
		return err
	}

	if !mode.Dual {
		if _, err := t.send(ctx, http.MethodPost, "/fapi/v1/positionSide/dual", url.Values{"dualSidePosition": {"true"}}); err != nil {
			return fmt.Errorf("рахунок не в режимі хеджування: %w", err)
		}
	}

	t.mu.Lock()
	t.hedged = true
	t.mu.Unlock()

	return nil
}

func (t *Broker) lever(ctx context.Context, contract string, leverage int) error {
	t.mu.Lock()
	set := t.levered[contract] == leverage
	t.mu.Unlock()

	if set {
		return nil
	}

	if _, err := t.send(ctx, http.MethodPost, "/fapi/v1/leverage", url.Values{
		"symbol":   {contract},
		"leverage": {strconv.Itoa(leverage)},
	}); err != nil {
		return err
	}

	t.mu.Lock()
	t.levered[contract] = leverage
	t.mu.Unlock()

	return nil
}

func (t *Broker) Orders(ctx context.Context) ([]trading.Order, error) {
	var raws []struct {
		ID       int64  `json:"orderId"`
		Client   string `json:"clientOrderId"`
		Symbol   string `json:"symbol"`
		Price    string `json:"price"`
		Executed string `json:"executedQty"`
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v1/openOrders", url.Values{}, &raws); err != nil {
		return nil, err
	}

	out := make([]trading.Order, 0, len(raws))

	for _, r := range raws {
		out = append(out, trading.Order{
			ID:       strconv.FormatInt(r.ID, 10),
			Tag:      r.Client,
			Contract: r.Symbol,
			Price:    parse(r.Price),
			Executed: parse(r.Executed),
		})
	}

	return out, nil
}

func (t *Broker) Cancel(ctx context.Context, contract, id string) (trading.Fill, error) {
	var order rawOrder

	err := t.call(ctx, http.MethodDelete, "/fapi/v1/order", url.Values{"symbol": {contract}, "orderId": {id}}, &order)
	if err != nil && strings.Contains(err.Error(), "(-2011)") {
		return t.Filled(ctx, contract, id)
	}

	if err != nil {
		return trading.Fill{}, err
	}

	return order.fill(), nil
}

func (t *Broker) Filled(ctx context.Context, contract, id string) (trading.Fill, error) {
	var order rawOrder

	if err := t.call(ctx, http.MethodGet, "/fapi/v1/order", url.Values{"symbol": {contract}, "orderId": {id}}, &order); err != nil {
		return trading.Fill{}, err
	}

	return order.fill(), nil
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

func (t *Broker) market(ctx context.Context, params url.Values) (trading.Fill, error) {
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
		return trading.Fill{}, fmt.Errorf("order %d, price %q, quantity %q: %w", order.ID, order.AvgPrice, order.Executed, errUnfilled)
	}

	return fill, nil
}

func (t *Broker) lastClose(ctx context.Context, p trading.Position) (float64, error) {
	var trades []struct {
		Side    string `json:"side"`
		Holding string `json:"positionSide"`
		Price   string `json:"price"`
	}

	params := url.Values{
		"symbol":    {p.Contract},
		"startTime": {strconv.FormatInt(p.Opened.UnixMilli(), 10)},
	}

	if err := t.call(ctx, http.MethodGet, "/fapi/v1/userTrades", params, &trades); err != nil {
		return 0, err
	}

	for _, trade := range slices.Backward(trades) {
		if trade.Side == closing(p.Side) && trade.Holding == holding(p.Side) {
			return parse(trade.Price), nil
		}
	}

	return 0, errors.New("position closed without a closing trade")
}

func (t *Broker) contract(ctx context.Context, contract string) (contractRules, error) {
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

func (t *Broker) exchangeRules(ctx context.Context) (map[string]contractRules, error) {
	var info exchangeInfo

	if err := t.do(ctx, http.MethodGet, t.base+"/fapi/v1/exchangeInfo", false, &info); err != nil {
		return nil, err
	}

	return info.rules(), nil
}

func (t *Broker) send(ctx context.Context, method, path string, params url.Values) (json.RawMessage, error) {
	var raw json.RawMessage

	err := t.call(ctx, method, path, params, &raw)

	return raw, err
}

func (t *Broker) call(ctx context.Context, method, path string, params url.Values, v any) error {
	params.Set("recvWindow", receiveWindow)
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	query := params.Encode()
	endpoint := t.base + path + "?" + query + "&signature=" + sign(t.secret, query)

	return t.do(ctx, method, endpoint, true, v)
}

func (t *Broker) do(ctx context.Context, method, endpoint string, signed bool, v any) error {
	if until, held := t.waiting(); held {
		return fmt.Errorf("testnet asked to wait until %s", until.UTC().Format(time.TimeOnly))
	}

	answer, err := t.ask(ctx, method, endpoint, signed)

	if err == nil && answer.status == http.StatusTeapot {
		t.http.CloseIdleConnections()

		answer, err = t.ask(ctx, method, endpoint, signed)
	}

	if err != nil {
		return err
	}

	switch answer.status {
	case http.StatusOK:
		t.answered()

		return json.Unmarshal(answer.body, v)
	case http.StatusTooManyRequests:
		t.hold(heldFor(answer.retryAfter, answer.body))
	case http.StatusTeapot:
		t.banned(heldFor(answer.retryAfter, answer.body))
	}

	return apiError(answer.status, answer.body)
}

type answer struct {
	status     int
	retryAfter string
	body       []byte
}

func (t *Broker) ask(ctx context.Context, method, endpoint string, signed bool) (answer, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return answer{}, err
	}

	if signed {
		req.Header.Set("X-MBX-APIKEY", t.key)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return answer{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, bodyLimit))
	if err != nil {
		return answer{}, err
	}

	return answer{status: resp.StatusCode, retryAfter: resp.Header.Get("Retry-After"), body: body}, nil
}

func (t *Broker) answered() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.probe = 0
}

func (t *Broker) banned(until time.Time) {
	t.mu.Lock()

	if time.Now().Before(t.held) {
		t.mu.Unlock()

		return
	}

	t.probe = max(firstProbe, t.probe*2)

	if next := time.Now().Add(t.probe); next.Before(until) {
		until = next
	}

	t.held = until
	t.mu.Unlock()
}

func (t *Broker) waiting() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.held, time.Now().Before(t.held)
}

func (t *Broker) hold(until time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.held = until
}

func heldFor(retryAfter string, body []byte) time.Time {
	until := time.Now().Add(shortestHold)

	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		until = later(until, time.Now().Add(time.Duration(seconds)*time.Second))
	}

	if m := bannedUntil.FindSubmatch(body); m != nil {
		if ms, err := strconv.ParseInt(string(m[1]), 10, 64); err == nil {
			until = later(until, time.UnixMilli(ms))
		}
	}

	return until
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}

	return a
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

func holding(side trading.Side) string {
	if side == trading.Long {
		return "LONG"
	}

	return "SHORT"
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
