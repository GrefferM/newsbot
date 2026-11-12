package binance

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"newsbot/internal/trading"
)

type fakeExchange struct {
	mu       sync.Mutex
	calls    []string
	position string
	income   string
}

func (f *fakeExchange) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	q := r.URL.Query()

	if r.URL.Path != "/fapi/v1/exchangeInfo" && (q.Get("signature") == "" || r.Header.Get("X-MBX-APIKEY") != "key") {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":-2014,"msg":"API-key format invalid."}`))

		return
	}

	f.calls = append(f.calls, r.Method+" "+r.URL.Path+" "+describe(q))

	switch r.URL.Path {
	case "/fapi/v1/exchangeInfo":
		_, _ = w.Write([]byte(`{"symbols":[{"symbol":"MOVEUSDT","status":"TRADING","filters":[
			{"filterType":"PRICE_FILTER","tickSize":"0.0001"},
			{"filterType":"LOT_SIZE","stepSize":"1"},
			{"filterType":"MIN_NOTIONAL","notional":"5"}]}]}`))
	case "/fapi/v1/order":
		_, _ = w.Write([]byte(`{"avgPrice":"0.4120","executedQty":"485"}`))
	case "/fapi/v2/positionRisk":
		_, _ = w.Write([]byte(`[{"positionAmt":"` + f.position + `"}]`))
	case "/fapi/v2/account":
		_, _ = w.Write([]byte(`{"totalWalletBalance":"4990.10","totalMarginBalance":"4998.50"}`))
	case "/fapi/v1/income":
		_, _ = w.Write([]byte(f.income))
	case "/fapi/v1/userTrades":
		_, _ = w.Write([]byte(`[{"side":"SELL","price":"0.4120"},{"side":"BUY","price":"0.5360"}]`))
	default:
		_, _ = w.Write([]byte(`{}`))
	}
}

func describe(q url.Values) string {
	var parts []string

	for _, key := range []string{"algoType", "side", "type", "quantity", "triggerPrice", "reduceOnly", "leverage"} {
		if v := q.Get(key); v != "" {
			parts = append(parts, key+"="+v)
		}
	}

	return strings.Join(parts, " ")
}

func newFakeTestnet(t *testing.T, f *fakeExchange) *Testnet {
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)

	tn := NewTestnet("key", "secret")
	tn.base = server.URL

	return tn
}

func position() trading.Position {
	return trading.Position{
		Contract: "MOVEUSDT",
		Stake:    200,
		Entry:    0.412,
		Stop:     0.5356,
		Target:   0.2884,
		Quantity: 485,
		Opened:   time.Now(),
	}
}

func TestTestnetEntersShortAndProtectsIt(t *testing.T) {
	f := &fakeExchange{}
	tn := newFakeTestnet(t, f)

	fill, err := tn.Enter(context.Background(), position())
	if err != nil {
		t.Fatal(err)
	}

	if err := tn.Protect(context.Background(), position()); err != nil {
		t.Fatal(err)
	}

	if fill.Price != 0.412 || fill.Quantity != 485 {
		t.Errorf("fill %+v", fill)
	}

	want := []string{
		"POST /fapi/v1/leverage leverage=1",
		"POST /fapi/v1/order side=SELL type=MARKET quantity=485",
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=BUY type=STOP_MARKET triggerPrice=0.5356",
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=BUY type=TAKE_PROFIT_MARKET triggerPrice=0.2884",
	}

	for _, call := range want {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}

func TestTestnetReportsAStopAndClosesOnTime(t *testing.T) {
	f := &fakeExchange{position: "-485"}
	tn := newFakeTestnet(t, f)

	if _, done, err := tn.Exited(context.Background(), position()); err != nil || done {
		t.Fatalf("open position reported done=%v err=%v", done, err)
	}

	f.position = "0"

	exit, done, err := tn.Exited(context.Background(), position())
	if err != nil || !done || exit.Price != 0.536 || exit.Reason != trading.ExitStop {
		t.Fatalf("exit %+v done=%v err=%v", exit, done, err)
	}

	price, err := tn.Close(context.Background(), position())
	if err != nil || price != 0.412 {
		t.Fatalf("close price %v err %v", price, err)
	}

	for _, call := range []string{
		"DELETE /fapi/v1/algoOpenOrders ",
		"DELETE /fapi/v1/allOpenOrders ",
		"POST /fapi/v1/order side=BUY type=MARKET quantity=485 reduceOnly=true",
	} {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}

func TestTestnetRefusesAContractItDoesNotList(t *testing.T) {
	tn := newFakeTestnet(t, &fakeExchange{})

	p := position()
	p.Contract = "BLURUSDT"

	if _, err := tn.Enter(context.Background(), p); err == nil || !strings.Contains(err.Error(), "не торгується на testnet") {
		t.Errorf("err %v", err)
	}

	if _, err := tn.Smallest(context.Background(), "BLURUSDT", 0.2); err == nil {
		t.Error("a contract the testnet does not list has a smallest order")
	}
}

func TestTheAccountAndATradesResultAreReadFromTheExchange(t *testing.T) {
	f := &fakeExchange{income: `[{"incomeType":"COMMISSION","income":"-0.08"},{"incomeType":"FUNDING_FEE","income":"0.03"},
		{"incomeType":"REALIZED_PNL","income":"-1.20"},{"incomeType":"TRANSFER","income":"500"},{"incomeType":"COMMISSION","income":"-0.08"}]`}
	tn := newFakeTestnet(t, f)

	balance, err := tn.Balance(context.Background())
	if err != nil || balance != 4998.5 {
		t.Errorf("the equity with the open result: %v, %v", balance, err)
	}

	earned, err := tn.Earned(context.Background(), position())
	if err != nil || math.Abs(earned+1.33) > 1e-9 {
		t.Errorf("the result with both fees and the funding, a transfer left out: %v, %v", earned, err)
	}

	f.income = `[{"incomeType":"COMMISSION","income":"-0.08"}]`

	if _, err := tn.Earned(context.Background(), position()); err == nil {
		t.Error("a trade the exchange has not booked yet has no result")
	}
}

func TestTheSmallestOrderIsTheMinimumRoundedUpToAWholeStep(t *testing.T) {
	tn := newFakeTestnet(t, &fakeExchange{})

	smallest, err := tn.Smallest(context.Background(), position().Contract, 0.33)
	if err != nil || smallest < 5 || smallest > 5.3 {
		t.Errorf("smallest %v, err %v", smallest, err)
	}

	coarse := contractRules{step: 0.001, minNotional: 50}

	if got := coarse.smallest(67000); got != 67 {
		t.Errorf("one step of a dear coin is the smallest order: %v", got)
	}
}

func TestTestnetEntersALongWithTheSidesTurnedRound(t *testing.T) {
	f := &fakeExchange{}
	tn := newFakeTestnet(t, f)

	long := position()
	long.Side = trading.Long
	long.Stop, long.Target = 0.3708, 0.4944

	if _, err := tn.Enter(context.Background(), long); err != nil {
		t.Fatal(err)
	}

	if err := tn.Protect(context.Background(), long); err != nil {
		t.Fatal(err)
	}

	if _, err := tn.Close(context.Background(), long); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"POST /fapi/v1/order side=BUY type=MARKET quantity=485",
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=SELL type=STOP_MARKET triggerPrice=0.3708",
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=SELL type=TAKE_PROFIT_MARKET triggerPrice=0.4944",
		"POST /fapi/v1/order side=SELL type=MARKET quantity=485 reduceOnly=true",
	}

	for _, call := range want {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}
