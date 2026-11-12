package testnet

import (
	"context"
	"errors"
	"fmt"
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
	long     string
	income   string
	unfilled bool
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
		if f.unfilled {
			_, _ = w.Write([]byte(`{"orderId":7,"avgPrice":"0","executedQty":"0"}`))

			return
		}

		_, _ = w.Write([]byte(`{"avgPrice":"0.4120","executedQty":"485"}`))
	case "/fapi/v1/openOrders":
		_, _ = w.Write([]byte(`[{"orderId":9,"clientOrderId":"rb-MOVE","symbol":"MOVEUSDT","price":"0.3800","executedQty":"0"}]`))
	case "/fapi/v2/positionRisk":
		_, _ = w.Write([]byte(`[{"positionAmt":"` + f.position + `","positionSide":"SHORT"},{"positionAmt":"` + f.long + `","positionSide":"LONG"}]`))
	case "/fapi/v2/account":
		_, _ = w.Write([]byte(`{"totalWalletBalance":"4990.10","totalMarginBalance":"4998.50"}`))
	case "/fapi/v1/income":
		_, _ = w.Write([]byte(f.income))
	case "/fapi/v1/userTrades":
		_, _ = w.Write([]byte(`[{"side":"SELL","positionSide":"SHORT","price":"0.4120"},{"side":"BUY","positionSide":"LONG","price":"0.3000"},
			{"side":"BUY","positionSide":"SHORT","price":"0.5360"},{"side":"SELL","positionSide":"LONG","price":"0.3300"}]`))
	default:
		_, _ = w.Write([]byte(`{}`))
	}
}

func describe(q url.Values) string {
	var parts []string

	for _, key := range []string{
		"algoType", "side", "positionSide", "type", "timeInForce", "price", "quantity", "orderId", "newClientOrderId", "triggerPrice",
		"leverage", "dualSidePosition", "clientAlgoId",
	} {
		if v := q.Get(key); v != "" {
			parts = append(parts, key+"="+v)
		}
	}

	return strings.Join(parts, " ")
}

func newFakeBroker(t *testing.T, f *fakeExchange) *Broker {
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)

	tn := New("key", "secret")
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
	tn := newFakeBroker(t, f)

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
		"GET /fapi/v1/positionSide/dual ",
		"POST /fapi/v1/positionSide/dual dualSidePosition=true",
		"POST /fapi/v1/leverage leverage=1",
		"POST /fapi/v1/order side=SELL positionSide=SHORT type=MARKET quantity=485",
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=BUY positionSide=SHORT type=STOP_MARKET triggerPrice=0.5356 clientAlgoId=" + guardID(stopTag, position()),
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=BUY positionSide=SHORT type=TAKE_PROFIT_MARKET triggerPrice=0.2884 clientAlgoId=" + guardID(targetTag, position()),
	}

	if stop, target := slices.Index(f.calls, want[4]), slices.Index(f.calls, want[5]); stop < 0 || target < stop {
		t.Errorf("the stop goes first: %q", f.calls)
	}

	for _, call := range want {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}

func TestAnEntryWithALimitIsSentAsIOCAndAMissIsReported(t *testing.T) {
	f := &fakeExchange{}
	tn := newFakeBroker(t, f)
	p := position()
	p.Limit = 0.41

	if _, err := tn.Enter(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	if call := "POST /fapi/v1/order side=SELL positionSide=SHORT type=LIMIT timeInForce=IOC price=0.4100 quantity=485"; !slices.Contains(f.calls, call) {
		t.Errorf("missing %q in %q", call, f.calls)
	}

	f.unfilled = true

	if _, err := tn.Enter(context.Background(), p); !errors.Is(err, trading.ErrMissed) {
		t.Errorf("an unfilled IOC: %v", err)
	}
}

func TestAnEntrySetsItsStrategysLeverageOnce(t *testing.T) {
	f := &fakeExchange{}
	tn := newFakeBroker(t, f)
	p := position()
	p.Leverage = 5

	for range 2 {
		if _, err := tn.Enter(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}

	set := 0

	for _, call := range f.calls {
		if strings.HasPrefix(call, "POST /fapi/v1/leverage") {
			set++

			if call != "POST /fapi/v1/leverage leverage=5" {
				t.Errorf("leverage call %q", call)
			}
		}
	}

	if set != 1 {
		t.Errorf("leverage set %d times", set)
	}
}

func TestABuyRestsInTheBookAndIsFoundAndCancelled(t *testing.T) {
	f := &fakeExchange{}
	tn := newFakeBroker(t, f)
	p := position()
	p.Side, p.Limit, p.Stake, p.Leverage = trading.Long, 0.38123, 100, 5

	if _, err := tn.Rest(context.Background(), p, "rb-MOVE"); err != nil {
		t.Fatal(err)
	}

	orders, err := tn.Orders(context.Background())
	if err != nil || len(orders) != 1 || orders[0].Tag != "rb-MOVE" || orders[0].Price != 0.38 || orders[0].ID != "9" {
		t.Fatalf("orders %+v, err %v", orders, err)
	}

	if fill, err := tn.Cancel(context.Background(), "MOVEUSDT", "9"); err != nil || fill.Quantity != 485 || fill.Price != 0.412 {
		t.Fatalf("fill %+v, err %v", fill, err)
	}

	for _, call := range []string{
		"POST /fapi/v1/leverage leverage=5",
		"POST /fapi/v1/order side=BUY positionSide=LONG type=LIMIT timeInForce=GTC price=0.3812 quantity=262 newClientOrderId=rb-MOVE",
		"DELETE /fapi/v1/order orderId=9",
	} {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}

func TestTestnetReportsAStopAndClosesOnTime(t *testing.T) {
	f := &fakeExchange{position: "-485"}
	tn := newFakeBroker(t, f)

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
		"DELETE /fapi/v1/algoOrder clientAlgoId=" + guardID(stopTag, position()),
		"DELETE /fapi/v1/algoOrder clientAlgoId=" + guardID(targetTag, position()),
		"POST /fapi/v1/order side=BUY positionSide=SHORT type=MARKET quantity=485",
	} {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}

func TestTestnetRefusesAContractItDoesNotList(t *testing.T) {
	tn := newFakeBroker(t, &fakeExchange{})

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
	tn := newFakeBroker(t, f)

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
	tn := newFakeBroker(t, &fakeExchange{})

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
	tn := newFakeBroker(t, f)

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
		"POST /fapi/v1/order side=BUY positionSide=LONG type=MARKET quantity=485",
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=SELL positionSide=LONG type=STOP_MARKET triggerPrice=0.3708 clientAlgoId=" + guardID(stopTag, long),
		"POST /fapi/v1/algoOrder algoType=CONDITIONAL side=SELL positionSide=LONG type=TAKE_PROFIT_MARKET triggerPrice=0.4944 clientAlgoId=" + guardID(targetTag, long),
		"POST /fapi/v1/order side=SELL positionSide=LONG type=MARKET quantity=485",
	}

	for _, call := range want {
		if !slices.Contains(f.calls, call) {
			t.Errorf("missing %q in %q", call, f.calls)
		}
	}
}

func TestARequestOneNodeRefusesIsAskedAgainAtOnceOverAnother(t *testing.T) {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		if calls%2 == 1 {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte(`{"code":-1003,"msg":"Way too many requests; IP(1.2.3.4) banned until 1795027018649."}`))

			return
		}

		_, _ = w.Write([]byte(`{"totalMarginBalance":"100"}`))
	}))
	t.Cleanup(server.Close)

	tn := New("key", "secret")
	tn.base = server.URL

	balance, err := tn.Balance(context.Background())
	if err != nil || balance != 100 || calls != 2 {
		t.Fatalf("balance %v, err %v, calls %d", balance, err, calls)
	}

	if _, held := tn.waiting(); held {
		t.Error("an answer from another node leaves nothing to wait for")
	}
}

func TestABanOnEveryPathIsProbedWithAGrowingPause(t *testing.T) {
	calls, banned := 0, true
	until := time.Now().Add(8 * time.Hour).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		if !banned {
			_, _ = w.Write([]byte(`{"totalMarginBalance":"100"}`))

			return
		}

		w.WriteHeader(http.StatusTeapot)
		fmt.Fprintf(w, `{"code":-1003,"msg":"Way too many requests; IP(1.2.3.4) banned until %d."}`, until)
	}))
	t.Cleanup(server.Close)

	tn := New("key", "secret")
	tn.base = server.URL

	if _, err := tn.Balance(context.Background()); err == nil || calls != 2 {
		t.Fatalf("a ban is an error after one more try: calls %d, err %v", calls, err)
	}

	_, err := tn.Balance(context.Background())

	if calls != 2 || err == nil || !strings.Contains(err.Error(), "asked to wait until") {
		t.Fatalf("a banned exchange was asked again at once: calls %d, err %v", calls, err)
	}

	if held, _ := tn.waiting(); time.Until(held) > firstProbe || time.Until(held) < firstProbe-time.Second {
		t.Fatalf("the first probe comes %v from now, want %v", time.Until(held), firstProbe)
	}

	tn.hold(time.Now().Add(-time.Second))

	if _, err := tn.Balance(context.Background()); err == nil || calls != 4 {
		t.Fatalf("the probe was not sent: calls %d, err %v", calls, err)
	}

	if held, _ := tn.waiting(); time.Until(held) < 2*firstProbe-time.Second {
		t.Fatalf("a refused probe doubles the pause, got %v", time.Until(held))
	}

	banned = false
	tn.hold(time.Now().Add(-time.Second))

	if balance, err := tn.Balance(context.Background()); err != nil || balance != 100 {
		t.Fatalf("balance %v, err %v", balance, err)
	}

	banned = true

	if _, err := tn.Balance(context.Background()); err == nil {
		t.Fatal("a ban is an error")
	}

	if held, _ := tn.waiting(); time.Until(held) > firstProbe {
		t.Errorf("an answer between two bans starts the pause anew, got %v", time.Until(held))
	}
}

func TestAProbeNeverWaitsLongerThanTheBanItself(t *testing.T) {
	until := time.Now().Add(30 * time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		fmt.Fprintf(w, `{"code":-1003,"msg":"Way too many requests; IP(1.2.3.4) banned until %d."}`, until.UnixMilli())
	}))
	t.Cleanup(server.Close)

	tn := New("key", "secret")
	tn.base = server.URL

	_, _ = tn.Balance(context.Background())

	if held, _ := tn.waiting(); held.After(time.Now().Add(shortestHold + time.Second)) {
		t.Errorf("held until %v for a ban that ends at %v", held, until)
	}
}

func TestOnlyAnExecutedOrderUpdateIsAFill(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    trading.Order
		filled  bool
	}{
		{
			name:    "a resting buy filled in part",
			message: `{"e":"ORDER_TRADE_UPDATE","T":1794987000100,"E":1794987000105,"o":{"s":"MOVEUSDT","c":"rb-MOVEUSDT-1","S":"BUY","o":"LIMIT","f":"GTC","q":"20","p":"0.94","ap":"0.94","sp":"0","x":"TRADE","X":"PARTIALLY_FILLED","i":42,"l":"12","z":"12","L":"0.94","n":"0.001","N":"USDT","T":1794987000100,"t":7,"b":"0","a":"0","m":true,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0"}}`,
			want:    trading.Order{ID: "42", Tag: "rb-MOVEUSDT-1", Contract: "MOVEUSDT", Price: 0.94, Executed: 12},
			filled:  true,
		},
		{
			name:    "a new order with nothing bought",
			message: `{"e":"ORDER_TRADE_UPDATE","T":1794987000100,"E":1794987000105,"o":{"s":"MOVEUSDT","c":"rb-MOVEUSDT-1","S":"BUY","p":"0.94","x":"NEW","X":"NEW","i":42,"l":"0","z":"0","L":"0"}}`,
		},
		{
			name:    "another event",
			message: `{"e":"ACCOUNT_UPDATE","a":{}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, filled := parseFill([]byte(c.message))

			if filled != c.filled || got != c.want {
				t.Errorf("got %+v %v", got, filled)
			}
		})
	}
}

func TestALongAndAShortOnOneContractAreToldApart(t *testing.T) {
	f := &fakeExchange{position: "-485", long: "0"}
	tn := newFakeBroker(t, f)

	short := position()
	short.ID = "MOVEUSDT-1-short"

	long := position()
	long.ID, long.Side, long.Stop, long.Target = "MOVEUSDT-1-long", trading.Long, 0.33, 0.47

	if _, done, err := tn.Exited(context.Background(), short); err != nil || done {
		t.Fatalf("the open short reported done=%v err=%v", done, err)
	}

	exit, done, err := tn.Exited(context.Background(), long)
	if err != nil || !done || exit.Price != 0.33 || exit.Reason != trading.ExitStop {
		t.Fatalf("the closed long beside an open short: exit %+v done=%v err=%v", exit, done, err)
	}

	if guardID(stopTag, long) == guardID(stopTag, short) {
		t.Fatal("two positions share a stop's tag")
	}

	for _, call := range f.calls {
		if strings.HasPrefix(call, "DELETE") && !strings.HasSuffix(call, guardID(stopTag, long)) && !strings.HasSuffix(call, guardID(targetTag, long)) {
			t.Errorf("the long's close touched another order: %q", call)
		}
	}

	moved := short
	moved.Stop, moved.Guard = 0.39, short.Guard+1

	if err := tn.Reprotect(context.Background(), short, moved); err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(f.calls, "DELETE /fapi/v1/algoOrder clientAlgoId="+guardID(stopTag, short)) ||
		!slices.Contains(f.calls, "POST /fapi/v1/algoOrder algoType=CONDITIONAL side=BUY positionSide=SHORT type=STOP_MARKET triggerPrice=0.3900 clientAlgoId="+guardID(stopTag, moved)) {
		t.Errorf("a moved stop replaces the position's own stop: %q", f.calls)
	}
}
