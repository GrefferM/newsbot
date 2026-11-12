package trading

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
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
	ErrMissed       = errors.New("ціна пішла за ліміт, ордер не виконано")
	ErrTaken        = errors.New("по цьому контракту вже є позиція в цей бік")
)

type TraderDeps struct {
	Market  MarketData
	Broker  Broker
	Store   BookStore
	Exits   Exits
	Limits  Limits
	Capital float64
	Every   time.Duration
	Journal Journal
	Log     *slog.Logger
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

	if book.Holds(market.Contract, draft.Side) {
		return ErrTaken
	}

	limits := t.deps.Limits
	if draft.AnyFunding {
		limits.CrowdedFunding = math.Inf(-1)
	}

	decision := t.decision(draft, book, market)
	decision.Checks = limits.Checks(market, draft.Side, decision.Size, book.Exposure())

	for _, c := range decision.Checks {
		if !c.OK {
			return fmt.Errorf("%s: %s", c.Name, c.Value)
		}
	}

	return t.enter(ctx, t.deps.Exits.For(draft), decision)
}

func (t *Trader) Admits(size float64, now time.Time) bool {
	book := t.Book()

	return !book.Halted(t.deps.Limits, now) && book.Exposure()+size <= t.deps.Limits.MaxExposure+1e-9
}

func (t *Trader) ClosePosition(ctx context.Context, id string) error {
	t.act.Lock()
	defer t.act.Unlock()

	p, found := t.openPosition(id)
	if !found {
		return ErrPositionGone
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

func (t *Trader) Restop(ctx context.Context, id string, stop float64) error {
	t.act.Lock()
	defer t.act.Unlock()

	p, found := t.openPosition(id)
	if !found {
		return ErrPositionGone
	}

	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	was := p
	p.Stop = stop
	p.Guard++

	err := t.deps.Broker.Reprotect(ctx, was, p)
	if err == nil {
		t.update(p)

		return nil
	}

	t.notify(ctx, Failed(t.Venue(), p.Symbol, "перенесення стопу", err, time.Now().UTC()))

	if closeErr := t.leave(ctx, p, ExitUnprotected, "закриття без стопу"); closeErr != nil {
		return closeErr
	}

	return err
}

func (t *Trader) openPosition(id string) (Position, bool) {
	for _, p := range t.Book().OpenPositions() {
		if p.ID == id {
			return p, true
		}
	}

	return Position{}, false
}

func (t *Trader) Watch(ctx context.Context) {
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

func (t *Trader) watch(ctx context.Context, seen Position) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	_, done, err := t.deps.Broker.Exited(ctx, seen)
	if err != nil {
		t.deps.Log.Warn("trade watch failed", "position", seen.ID, "err", err)

		return
	}

	if !done && !seen.Bare && !seen.Expired(time.Now()) {
		return
	}

	t.act.Lock()
	defer t.act.Unlock()

	p, open := t.openPosition(seen.ID)
	if !open {
		return
	}

	exit, done, err := t.deps.Broker.Exited(ctx, p)
	if err != nil {
		t.deps.Log.Warn("trade watch failed", "position", p.ID, "err", err)

		return
	}

	if done {
		t.close(ctx, p, exit)

		return
	}

	if p.Expired(time.Now()) {
		t.leave(ctx, p, ExitTime, "закриття")

		return
	}

	if p.Bare {
		t.guard(ctx, p)
	}
}

func (t *Trader) guard(ctx context.Context, p Position) {
	if err := t.deps.Broker.Reprotect(ctx, p, p); err != nil {
		t.leave(ctx, p, ExitUnprotected, "закриття без стопу")

		return
	}

	p.Bare = false

	t.update(p)
}

func (t *Trader) leave(ctx context.Context, p Position, reason, step string) error {
	price, err := t.deps.Broker.Close(ctx, p)
	if err != nil {
		t.notify(ctx, Failed(t.Venue(), p.Symbol, step, err, time.Now().UTC()))

		if !p.Bare && reason == ExitUnprotected {
			p.Bare = true

			t.update(p)
		}

		return err
	}

	t.close(ctx, p, Exit{Price: price, Reason: reason})

	return nil
}

func (t *Trader) enter(ctx context.Context, exits Exits, d Decision) error {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	now := time.Now().UTC()
	planned := exits.Open(d, t.Book(), now)

	fill, err := t.deps.Broker.Enter(ctx, planned)
	if errors.Is(err, ErrMissed) {
		return err
	}

	if err != nil {
		t.notify(ctx, Failed(t.Venue(), d.Symbol, "вхід", err, now))

		return err
	}

	t.opened(ctx, exits.Filled(planned, fill.Price, fill.Quantity), d, now)

	return nil
}

func (t *Trader) opened(ctx context.Context, position Position, d Decision, now time.Time) {
	t.update(position)

	unprotected := t.deps.Broker.Protect(ctx, position)

	book := t.fund(ctx)

	t.notify(ctx, Opened(t.Venue(), position, book, d))

	if unprotected == nil {
		return
	}

	t.notify(ctx, Failed(t.Venue(), d.Symbol, "стоп і тейк", unprotected, now))

	_ = t.leave(ctx, position, ExitUnprotected, "закриття без стопу")
}

func (t *Trader) Rest(ctx context.Context, draft Draft, tag string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	book := t.Book()
	if book.Equity <= 0 {
		return "", errors.New("рахунок порожній")
	}

	planned := t.deps.Exits.For(draft).Open(t.resting(draft, book), book, time.Now().UTC())

	return t.deps.Broker.Rest(ctx, planned, tag)
}

func (t *Trader) Orders(ctx context.Context) ([]Order, error) {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	return t.deps.Broker.Orders(ctx)
}

func (t *Trader) Fills(ctx context.Context, each func(Order)) error {
	return t.deps.Broker.Fills(ctx, func(o Order) {
		for _, p := range t.Book().OpenPositions() {
			if p.Contract == o.Contract {
				t.watch(ctx, p)
			}
		}

		each(o)
	})
}

func (t *Trader) Cancel(ctx context.Context, contract, id string) error {
	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	_, err := t.deps.Broker.Cancel(ctx, contract, id)

	return err
}

func (t *Trader) Adopt(ctx context.Context, draft Draft, id string) error {
	t.act.Lock()
	defer t.act.Unlock()

	ctx, cancel := context.WithTimeout(ctx, traderTimeout)
	defer cancel()

	fill, err := t.deps.Broker.Cancel(ctx, draft.Contract, id)
	if err != nil {
		return err
	}

	if fill.Quantity <= 0 {
		return ErrMissed
	}

	now := time.Now().UTC()
	book := t.Book()
	exits := t.deps.Exits.For(draft)
	d := t.resting(draft, book)
	planned := exits.Open(d, book, now)

	t.opened(ctx, exits.Filled(planned, fill.Price, fill.Quantity), d, now)

	return nil
}

func (t *Trader) resting(draft Draft, book Book) Decision {
	return t.decision(draft, book, Market{Listed: true, Contract: draft.Contract, Price: draft.Limit})
}

func (t *Trader) decision(draft Draft, book Book, market Market) Decision {
	return Decision{
		Symbol:   draft.Symbol,
		Side:     draft.Side,
		Kind:     cmp.Or(draft.Kind, KindPlan),
		Strategy: draft.Strategy,
		Size:     draft.Stake / book.Equity,
		Limit:    draft.Limit,
		Leverage: draft.Leverage,
		Market:   market,
	}
}

func (t *Trader) close(ctx context.Context, p Position, exit Exit) {
	closed := t.deps.Exits.Close(p, exit.Price, exit.Reason, time.Now().UTC())

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
