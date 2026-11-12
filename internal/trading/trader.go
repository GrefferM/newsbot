package trading

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"newsbot/internal/news"
)

const (
	traderSource  = "trader"
	outcomeTrade  = "торгівля"
	traderTimeout = 30 * time.Second
	bookingWindow = 24 * time.Hour
)

var (
	ErrNoMarket     = errors.New("ринок монети недоступний")
	ErrHalted       = errors.New("денний ліміт збитку вичерпано")
	ErrPositionGone = errors.New("відкритої позиції з таким id немає")
)

type TraderDeps struct {
	Market   MarketData
	Broker   Broker
	Store    BookStore
	Strategy Strategy
	Limits   Limits
	Capital  float64
	Every    time.Duration
	Journal  Journal
	Log      *slog.Logger
}

type Trader struct {
	deps TraderDeps
	act  sync.Mutex
	mu   sync.Mutex
	book Book
	out  chan<- news.Event
}

func NewTrader(deps TraderDeps, book Book) *Trader {
	return &Trader{deps: deps, book: book}
}

func (t *Trader) Venue() string {
	return t.deps.Broker.Venue()
}

func (t *Trader) Run(ctx context.Context, out chan<- news.Event) {
	t.mu.Lock()
	t.out = out
	t.mu.Unlock()

	ticker := time.NewTicker(t.deps.Every)
	defer ticker.Stop()

	t.Watch(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.Watch(ctx)
		}
	}
}

func (t *Trader) Book() Book {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.book
}

func (t *Trader) Place(ctx context.Context, draft Draft) error {
	t.act.Lock()
	defer t.act.Unlock()

	book := t.fund(ctx)

	if book.Equity <= 0 {
		return errors.New("рахунок порожній")
	}

	market, ok := t.market(ctx, draft.Symbol)
	if !ok {
		return ErrNoMarket
	}

	if book.Halted(t.deps.Limits, time.Now().UTC()) {
		return ErrHalted
	}

	decision := Decision{
		Symbol: draft.Symbol,
		Side:   draft.Side,
		Kind:   KindPlan,
		Size:   draft.Stake / book.Equity,
		Market: market,
		Checks: t.deps.Limits.Checks(market, draft.Side, draft.Stake/book.Equity, book.Exposure()),
	}

	for _, c := range decision.Checks {
		if !c.OK {
			return fmt.Errorf("%s: %s", c.Name, c.Value)
		}
	}

	t.enter(ctx, t.deps.Strategy.For(draft), decision)

	return nil
}

func (t *Trader) ClosePosition(ctx context.Context, id string) error {
	t.act.Lock()
	defer t.act.Unlock()

	for _, p := range t.Book().OpenPositions() {
		if p.ID != id {
			continue
		}

		ctx, cancel := context.WithTimeout(ctx, traderTimeout)
		defer cancel()

		price, err := t.deps.Broker.Close(ctx, p)
		if err != nil {
			return err
		}

		t.close(ctx, p, Exit{Price: price, Reason: ExitManual})

		return nil
	}

	return ErrPositionGone
}

func (t *Trader) Watch(ctx context.Context) {
	t.act.Lock()
	defer t.act.Unlock()

	for _, p := range t.Book().OpenPositions() {
		t.watch(ctx, p)
	}

	for _, p := range t.Book().Unbooked(time.Now().UTC().Add(-bookingWindow)) {
		t.collect(ctx, p)
	}

	t.fund(ctx)
}

func (t *Trader) collect(ctx context.Context, p Position) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	if earned, err := t.deps.Broker.Earned(ctx, p); err == nil {
		t.update(p.Settled(earned))
	}
}

func (t *Trader) fund(ctx context.Context) Book {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	balance, err := t.deps.Broker.Balance(ctx)
	if err != nil {
		if !errors.Is(err, ErrNoVenue) {
			t.deps.Log.Warn("account balance failed", "err", err)
		}

		return t.Book()
	}

	t.mu.Lock()
	before := t.book.Equity
	t.book = t.book.Funded(balance, t.deps.Capital)
	book := t.book
	changed := book.Equity != before
	t.mu.Unlock()

	if changed {
		t.save(book)
	}

	return book
}

func (t *Trader) watch(ctx context.Context, p Position) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	exit, done, err := t.deps.Broker.Exited(ctx, p)
	if err != nil {
		t.deps.Log.Warn("trade watch failed", "position", p.ID, "err", err)

		return
	}

	if !done && p.Expired(time.Now()) {
		exit.Reason = ExitTime

		exit.Price, err = t.deps.Broker.Close(ctx, p)
		if err != nil {
			t.notify(ctx, Failed(t.Venue(), p.Symbol, "закриття", err, time.Now().UTC()))

			return
		}

		done = true
	}

	if done {
		t.close(ctx, p, exit)
	}
}

func (t *Trader) enter(ctx context.Context, strategy Strategy, d Decision) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	now := time.Now().UTC()
	planned := strategy.Open(d, t.Book(), now)

	fill, err := t.deps.Broker.Enter(ctx, planned)
	if err != nil {
		t.notify(ctx, Failed(t.Venue(), d.Symbol, "вхід", err, now))

		return
	}

	position := strategy.Filled(planned, fill.Price, fill.Quantity)

	t.update(position)

	book := t.fund(ctx)

	t.notify(ctx, Opened(t.Venue(), position, book, d))

	if err := t.deps.Broker.Protect(ctx, position); err != nil {
		t.notify(ctx, Failed(t.Venue(), d.Symbol, "стоп і тейк", err, now))

		price, closeErr := t.deps.Broker.Close(ctx, position)
		if closeErr != nil {
			t.notify(ctx, Failed(t.Venue(), d.Symbol, "закриття без стопу", closeErr, now))

			return
		}

		t.close(ctx, position, Exit{Price: price, Reason: ExitUnprotected})
	}
}

func (t *Trader) close(ctx context.Context, p Position, exit Exit) {
	closed := t.deps.Strategy.Close(p, exit.Price, exit.Reason, time.Now().UTC())

	if earned, err := t.deps.Broker.Earned(ctx, closed); err == nil {
		closed = closed.Settled(earned)
	}

	t.update(closed)

	book := t.fund(ctx)

	t.notify(ctx, Closed(t.Venue(), closed, book))
}

func (t *Trader) market(ctx context.Context, symbol news.Symbol) (Market, bool) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	ticker, listed, err := t.deps.Market.Ticker(ctx, symbol)
	if err != nil {
		t.deps.Log.Warn("trade market failed", "symbol", symbol, "err", err)

		return Market{}, false
	}

	return t.Sized(ctx, Market{
		Listed:      listed,
		Contract:    ticker.Contract,
		Price:       ticker.Price,
		Funding:     ticker.Funding,
		QuoteVolume: ticker.QuoteVolume,
	}), true
}

func (t *Trader) Sized(ctx context.Context, m Market) Market {
	equity := t.Book().Equity

	if !m.Listed || equity <= 0 {
		return m
	}

	smallest, err := t.deps.Broker.Smallest(ctx, m.Contract, m.Price)
	if err != nil {
		m.Listed, m.Refused = false, err.Error()

		return m
	}

	m.MinSize = smallest / equity

	return m
}

func (t *Trader) update(p Position) Book {
	t.mu.Lock()
	t.book = t.book.With(p)
	book := t.book
	t.mu.Unlock()

	t.save(book)

	return book
}

func (t *Trader) save(book Book) {
	if err := t.deps.Store.Save(book); err != nil {
		t.deps.Log.Error("save trading book", "err", err)
		t.deps.Journal.Fail(news.Failure{
			At:     time.Now().UTC(),
			Source: traderSource,
			Err:    err,
		})
	}
}

func (t *Trader) notify(ctx context.Context, e news.Event) {
	t.deps.Journal.Record(news.Observation{
		SeenAt:  time.Now().UTC(),
		Source:  traderSource,
		Event:   e,
		Outcome: outcomeTrade,
	})
	t.deps.Log.Info("trade", "venue", e.Source, "kind", e.Kind, "title", e.Title)

	t.mu.Lock()
	out := t.out
	t.mu.Unlock()

	if out == nil {
		return
	}

	select {
	case out <- e:
	case <-ctx.Done():
	}
}
