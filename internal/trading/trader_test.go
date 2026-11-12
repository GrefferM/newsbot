package trading

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

type fakeMarket struct {
	mu      sync.Mutex
	tickers map[news.Symbol]market.Ticker
}

func (m *fakeMarket) Ticker(_ context.Context, s news.Symbol) (market.Ticker, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tickers[s]

	return t, ok, nil
}

func (m *fakeMarket) OpenInterestChange(context.Context, string, time.Duration) (float64, bool) {
	return 0, false
}

func (m *fakeMarket) set(s news.Symbol, price float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := m.tickers[s]
	t.Price = price
	m.tickers[s] = t
}

type memoryBooks struct {
	mu    sync.Mutex
	saved []Book
}

func (b *memoryBooks) Save(book Book) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.saved = append(b.saved, book)

	return nil
}

func newTestTrader(m *fakeMarket, store *memoryBooks, out chan news.Event) *Trader {
	trader := NewTrader(TraderDeps{
		Market: m,
		Broker: newFakeVenue(m, 1000),
		Store:  store,
		Exits: Exits{
			StopLoss:   0.30,
			TakeProfit: 0.30,
			Hold:       24 * time.Hour,
		},
		Limits:  Limits{MaxExposure: 1, MinVolume: 1e6, CrowdedFunding: -0.0005, DailyLoss: 0.1},
		Every:   time.Hour,
		Journal: newMemoryJournal(),
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, Book{})

	trader.out = out

	return trader
}

func liquidMarket() *fakeMarket {
	return &fakeMarket{tickers: map[news.Symbol]market.Ticker{
		"MOVE": {Contract: "MOVEUSDT", Price: 1, QuoteVolume: 5e6},
	}}
}

func shortDraft(stake float64) Draft {
	return Draft{Symbol: "MOVE", Side: Short, Stake: stake, StopLoss: 0.30, TakeProfit: 0.30, Hold: 24 * time.Hour}
}

func TestAPlanEntersOnlyThroughTheChecks(t *testing.T) {
	out := make(chan news.Event, 4)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, out)

	if err := trader.Place(context.Background(), shortDraft(1200)); err == nil {
		t.Fatal("an order over the exposure cap passed")
	}

	if err := trader.Place(context.Background(), Draft{Symbol: "NONE", Side: Short, Stake: 100}); err == nil {
		t.Fatal("an order on a coin without a market passed")
	}

	if err := trader.Place(context.Background(), shortDraft(200)); err != nil {
		t.Fatal(err)
	}

	opened := <-out
	position := trader.Book().OpenPositions()[0]

	if opened.Kind != "вхід" || position.Kind != KindPlan || position.Stake != 200 || position.Size != 0.2 {
		t.Errorf("notice %q, position %+v", opened.Kind, position)
	}
}

type pickyBroker struct {
	*fakeVenue
	smallest float64
	unlisted string
}

func (b pickyBroker) Smallest(_ context.Context, contract string, _ float64) (float64, error) {
	if contract == b.unlisted {
		return 0, errors.New(contract + " не торгується на testnet")
	}

	return b.smallest, nil
}

func TestAnOrderSmallerThanTheExchangeTakesIsRefusedBeforeItIsSent(t *testing.T) {
	out := make(chan news.Event, 4)
	m := liquidMarket()
	trader := newTestTrader(m, &memoryBooks{}, out)
	trader.deps.Broker = pickyBroker{fakeVenue: newFakeVenue(m, 1000), smallest: 50}

	err := trader.Place(context.Background(), shortDraft(20))
	if err == nil || !strings.Contains(err.Error(), "найменший ордер 5.00% рахунку") {
		t.Fatalf("an order under the exchange's minimum: %v", err)
	}

	if err := trader.Place(context.Background(), shortDraft(60)); err != nil {
		t.Fatal(err)
	}

	sized := trader.Sized(context.Background(), Market{Listed: true, Contract: "GONEUSDT", Price: 1})
	trader.deps.Broker = pickyBroker{fakeVenue: newFakeVenue(m, 1000), unlisted: "GONEUSDT"}
	gone := trader.Sized(context.Background(), Market{Listed: true, Contract: "GONEUSDT", Price: 1})

	if !sized.Listed || gone.Listed || !strings.Contains(gone.Refused, "не торгується") {
		t.Errorf("a contract the venue does not trade is not listed for the desk: %+v", gone)
	}
}

func TestTheSizeTheResultAndTheEquityComeFromTheExchange(t *testing.T) {
	out := make(chan news.Event, 4)
	m := liquidMarket()
	venue := newFakeVenue(m, 5000)
	venue.fees = 1.5
	trader := newTestTrader(m, &memoryBooks{}, out)
	trader.deps.Broker = venue

	if err := trader.Place(context.Background(), shortDraft(500)); err != nil {
		t.Fatal(err)
	}

	position := trader.Book().OpenPositions()[0]

	if trader.Book().Start != 5000 || position.Size != 0.1 {
		t.Fatalf("the account is the exchange's balance: %+v, size %v", trader.Book(), position.Size)
	}

	if err := trader.ClosePosition(context.Background(), position.ID); err != nil {
		t.Fatal(err)
	}

	closed := trader.Book().Positions[0]

	if closed.PnL != -1.5 || closed.Return != -0.003 || trader.Book().Equity != 4998.5 {
		t.Errorf("fees the exchange took are in the result and the equity: %+v, equity %v", closed, trader.Book().Equity)
	}
}

func TestAWorkingCapitalSizesTheDeskWhateverTheExchangeHolds(t *testing.T) {
	out := make(chan news.Event, 4)
	m := liquidMarket()
	trader := newTestTrader(m, &memoryBooks{}, out)
	trader.deps.Capital = 100
	trader.deps.Broker = pickyBroker{fakeVenue: newFakeVenue(m, 5000), smallest: 5}

	if err := trader.Place(context.Background(), shortDraft(4)); err == nil || !strings.Contains(err.Error(), "найменший ордер 5.00% рахунку") {
		t.Fatalf("four dollars of a hundred is under the exchange's five: %v", err)
	}

	if err := trader.Place(context.Background(), shortDraft(6)); err != nil {
		t.Fatal(err)
	}

	book := trader.Book()

	if book.Start != 100 || book.Equity != 100 || book.OpenPositions()[0].Size != 0.06 {
		t.Errorf("the desk counts from its capital: %+v", book)
	}
}

func TestAResultTheExchangeBooksLateReplacesTheEstimateOnTheNextLook(t *testing.T) {
	out := make(chan news.Event, 4)
	m := liquidMarket()
	venue := newFakeVenue(m, 1000)
	venue.fees, venue.late = 2, 1
	trader := newTestTrader(m, &memoryBooks{}, out)
	trader.deps.Broker = venue

	if err := trader.Place(context.Background(), shortDraft(100)); err != nil {
		t.Fatal(err)
	}

	_ = trader.ClosePosition(context.Background(), trader.Book().OpenPositions()[0].ID)

	if p := trader.Book().Positions[0]; p.Booked || p.PnL == -2 {
		t.Fatalf("before the exchange books it the result is the desk's estimate: %+v", p)
	}

	trader.Watch(context.Background())

	if p := trader.Book().Positions[0]; !p.Booked || p.PnL != -2 || trader.Book().Equity != 998 {
		t.Errorf("the next look takes the exchange's result: %+v, equity %v", p, trader.Book().Equity)
	}
}

func TestADailyLossStopsEveryNewEntry(t *testing.T) {
	out := make(chan news.Event, 4)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, out)
	trader.book.Positions = []Position{{Size: 0.2, ClosedAt: time.Now().UTC().Add(-time.Hour), PnL: -150}}

	if err := trader.Place(context.Background(), shortDraft(100)); err != ErrHalted {
		t.Errorf("an entry after the daily loss: %v", err)
	}
}

func TestTraderClosesAtTheStop(t *testing.T) {
	m := liquidMarket()
	out := make(chan news.Event, 4)
	trader := newTestTrader(m, &memoryBooks{}, out)

	if err := trader.Place(context.Background(), shortDraft(200)); err != nil {
		t.Fatal(err)
	}

	<-out

	m.set("MOVE", 1.2)
	trader.Watch(context.Background())

	if len(out) != 0 {
		t.Fatalf("closed early: %+v", <-out)
	}

	m.set("MOVE", 1.35)
	trader.Watch(context.Background())

	closed := <-out
	book := trader.Book()

	if closed.Kind != "вихід · "+ExitStop || len(book.OpenPositions()) != 0 || book.Equity >= 1000 {
		t.Errorf("notice %q, open %d, equity %v", closed.Kind, len(book.OpenPositions()), book.Equity)
	}
}

func TestATakeIsFilledAtItsOwnPriceAndAStopAtTheMarket(t *testing.T) {
	m := liquidMarket()
	out := make(chan news.Event, 4)
	trader := newTestTrader(m, &memoryBooks{}, out)

	if err := trader.Place(context.Background(), shortDraft(200)); err != nil {
		t.Fatal(err)
	}

	<-out

	m.set("MOVE", 0.5)
	trader.Watch(context.Background())

	closed := <-out
	position := trader.Book().Positions[0]

	if closed.Kind != "вихід · "+ExitTarget || math.Abs(position.Exit-0.7) > 1e-9 {
		t.Errorf("a price that jumped past the take must not be credited: %q, exit %v", closed.Kind, position.Exit)
	}

	if record := trader.Book().Record(); record.Closed != 1 || record.Won != 1 || record.Lost != 0 || record.Open != 0 || record.PnL <= 0 {
		t.Errorf("the closed trade is counted as won: %+v", record)
	}
}

func TestAPositionClosesByHand(t *testing.T) {
	out := make(chan news.Event, 4)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, out)

	if err := trader.Place(context.Background(), shortDraft(200)); err != nil {
		t.Fatal(err)
	}

	<-out

	id := trader.Book().OpenPositions()[0].ID

	if err := trader.ClosePosition(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	if notice := <-out; notice.Kind != "вихід · "+ExitManual || len(trader.Book().OpenPositions()) != 0 {
		t.Errorf("notice %q", notice.Kind)
	}

	if err := trader.ClosePosition(context.Background(), id); err != ErrPositionGone {
		t.Errorf("second close: %v", err)
	}
}

func TestAPumpShortIsNotRefusedForACrowdedShort(t *testing.T) {
	m := liquidMarket()
	m.tickers["MOVE"] = market.Ticker{Contract: "MOVEUSDT", Price: 1, QuoteVolume: 5e6, Funding: -0.01}

	trader := newTestTrader(m, &memoryBooks{}, make(chan news.Event, 4))

	if err := trader.Place(context.Background(), shortDraft(100)); err == nil {
		t.Fatal("a plan into a crowded short passed")
	}

	pump := shortDraft(100)
	pump.Kind, pump.Strategy, pump.AnyFunding = "Памп Монети", "pump", true

	if err := trader.Place(context.Background(), pump); err != nil {
		t.Fatal(err)
	}

	if p := trader.Book().OpenPositions()[0]; p.Kind != "Памп Монети" || p.Strategy != "pump" {
		t.Errorf("position %+v", p)
	}
}

func TestAStopMovesOrThePositionIsClosed(t *testing.T) {
	out := make(chan news.Event, 8)
	venue := newFakeVenue(liquidMarket(), 1000)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, out)
	trader.deps.Broker = venue

	if err := trader.Place(context.Background(), shortDraft(200)); err != nil {
		t.Fatal(err)
	}

	<-out

	id := trader.Book().OpenPositions()[0].ID

	if err := trader.Restop(context.Background(), id, 0.95); err != nil {
		t.Fatal(err)
	}

	if stop := trader.Book().OpenPositions()[0].Stop; stop != 0.95 {
		t.Errorf("stop %v", stop)
	}

	venue.reprotectErr = errors.New("відмова")

	if err := trader.Restop(context.Background(), id, 0.9); err == nil {
		t.Fatal("a failed move was not reported")
	}

	if open := trader.Book().OpenPositions(); len(open) != 0 {
		t.Errorf("a position without a stop stays open: %+v", open)
	}
}

func TestAMissedLimitEntryIsReturnedAndNotToldAsAFailure(t *testing.T) {
	out := make(chan news.Event, 4)
	venue := newFakeVenue(liquidMarket(), 1000)
	venue.missed = true
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, out)
	trader.deps.Broker = venue

	draft := shortDraft(100)
	draft.Limit = 0.99

	if err := trader.Place(context.Background(), draft); !errors.Is(err, ErrMissed) {
		t.Fatalf("err %v", err)
	}

	if len(out) != 0 || len(trader.Book().OpenPositions()) != 0 {
		t.Errorf("told %d, open %d", len(out), len(trader.Book().OpenPositions()))
	}
}

func TestARestingBuyIsPlacedAndAdoptedOnceFilled(t *testing.T) {
	out := make(chan news.Event, 8)
	venue := newFakeVenue(liquidMarket(), 1000)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, out)
	trader.deps.Broker = venue
	trader.fund(context.Background())

	draft := Draft{Symbol: "MOVE", Side: Long, Contract: "MOVEUSDT", Stake: 100, Limit: 0.8, StopLoss: 0.1, TakeProfit: 0.15, Hold: 4 * time.Hour, Kind: "Відскок", Strategy: "rebound", Leverage: 10}

	id, err := trader.Rest(context.Background(), draft, "rb-1")
	if err != nil || id != "rb-1" || len(venue.resting) != 1 || venue.resting[0].Limit != 0.8 || venue.resting[0].Stake != 100 ||
		venue.resting[0].Leverage != 10 {
		t.Fatalf("id %q, err %v, resting %+v", id, err, venue.resting)
	}

	if err := trader.Adopt(context.Background(), draft, "rb-1"); !errors.Is(err, ErrMissed) {
		t.Fatalf("an order filled by nothing: %v", err)
	}

	venue.fills = map[string]Fill{"rb-1": {Price: 0.8, Quantity: 125}}

	if err := trader.Adopt(context.Background(), draft, "rb-1"); err != nil {
		t.Fatal(err)
	}

	p := trader.Book().OpenPositions()[0]

	if p.Strategy != "rebound" || p.Side != Long || p.Entry != 0.8 || math.Abs(p.Stop-0.72) > 1e-9 || math.Abs(p.Target-0.92) > 1e-9 {
		t.Errorf("position %+v", p)
	}

	if len(venue.cancelled) != 2 {
		t.Errorf("an order is taken off the book as it is adopted: %v", venue.cancelled)
	}
}

type memoryJournal struct {
	mu       sync.Mutex
	outcomes map[string]string
}

func newMemoryJournal() *memoryJournal {
	return &memoryJournal{outcomes: map[string]string{}}
}

func (j *memoryJournal) Record(o news.Observation) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.outcomes[o.Event.ID] = o.Outcome
}

func (j *memoryJournal) Fail(f news.Failure) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.outcomes[f.Source] = "помилка: " + f.Err.Error()
}

func TestAContractHoldsOnePositionOnEachSide(t *testing.T) {
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, make(chan news.Event, 8))

	if err := trader.Place(context.Background(), shortDraft(100)); err != nil {
		t.Fatal(err)
	}

	if err := trader.Place(context.Background(), shortDraft(100)); !errors.Is(err, ErrTaken) {
		t.Fatalf("a second short on the contract: %v", err)
	}

	long := shortDraft(100)
	long.Side = Long

	if err := trader.Place(context.Background(), long); err != nil {
		t.Fatalf("a long beside the short of the contract: %v", err)
	}

	if open := trader.Book().OpenPositions(); len(open) != 2 || open[0].ID == open[1].ID {
		t.Errorf("two positions of one contract are kept apart: %+v", open)
	}
}

func TestTheStopIsPlacedBeforeTheAccountIsReadAgain(t *testing.T) {
	venue := newFakeVenue(liquidMarket(), 1000)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, make(chan news.Event, 8))
	trader.deps.Broker = venue
	trader.fund(context.Background())

	venue.mu.Lock()
	venue.calls = nil
	venue.mu.Unlock()

	if err := trader.Place(context.Background(), shortDraft(100)); err != nil {
		t.Fatal(err)
	}

	venue.mu.Lock()
	defer venue.mu.Unlock()

	if !slices.Equal(venue.calls, []string{"balance", "protect", "balance"}) {
		t.Errorf("calls %v", venue.calls)
	}
}

func TestNoRoomIsLeftAfterTheDailyLossOrAtFullExposure(t *testing.T) {
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, make(chan news.Event, 8))
	now := time.Now().UTC()

	if !trader.Admits(0.1, now) {
		t.Fatal("an empty account admits an entry")
	}

	if trader.Admits(1.1, now) {
		t.Error("an entry beyond the whole account was admitted")
	}

	trader.update(Position{ID: "L", Stake: 100, PnL: -200, ClosedAt: now})

	if trader.Admits(0.1, now) {
		t.Error("an entry was admitted after the day's loss limit")
	}
}

func TestAPositionLeftBareIsProtectedOrClosedAtTheNextLook(t *testing.T) {
	venue := newFakeVenue(liquidMarket(), 1000)
	trader := newTestTrader(liquidMarket(), &memoryBooks{}, make(chan news.Event, 16))
	trader.deps.Broker = venue
	trader.fund(context.Background())

	venue.protectErr, venue.closeErr = errors.New("refused"), errors.New("refused")

	if err := trader.Place(context.Background(), shortDraft(100)); err != nil {
		t.Fatal(err)
	}

	if open := trader.Book().OpenPositions(); len(open) != 1 || !open[0].Bare {
		t.Fatalf("a position neither protected nor closed is marked bare: %+v", open)
	}

	venue.mu.Lock()
	venue.protectErr, venue.closeErr = nil, nil
	venue.mu.Unlock()

	trader.Watch(context.Background())

	if open := trader.Book().OpenPositions(); len(open) != 1 || open[0].Bare {
		t.Fatalf("the next look puts the stop back: %+v", open)
	}

	bare := trader.Book().OpenPositions()[0]
	bare.Bare = true
	trader.update(bare)

	venue.mu.Lock()
	venue.protectErr = errors.New("refused")
	venue.mu.Unlock()

	trader.Watch(context.Background())

	if open := trader.Book().OpenPositions(); len(open) != 0 {
		t.Errorf("a position that cannot be protected is closed: %+v", open)
	}
}
