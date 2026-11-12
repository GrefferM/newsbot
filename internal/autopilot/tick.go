package autopilot

import (
	"context"
	"sync"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

func (s *Service) Run(ctx context.Context, out chan<- news.Event) {
	ticker := time.NewTicker(s.deps.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx, time.Now().UTC())
			s.flush(ctx, out)
		}
	}
}

func (s *Service) Tick(ctx context.Context, now time.Time) {
	s.mu.Lock()
	groups := s.watches.ByAsset()
	s.mu.Unlock()

	slots := make(chan struct{}, max(s.deps.AtOnce, 1))

	var wg sync.WaitGroup

	for _, group := range groups {
		wg.Go(func() {
			slots <- struct{}{}

			defer func() {
				<-slots
			}()

			s.check(ctx, group, now)
		})
	}

	wg.Wait()
}

func (s *Service) check(ctx context.Context, group []Watch, now time.Time) {
	for _, w := range group {
		if !s.armed(w.ID) {
			continue
		}

		started := time.Now()
		looked := s.look(ctx, w, now)

		s.settle(looked, now.Add(time.Since(started)))
	}
}

func (s *Service) armed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, found := s.watches.Find(id)

	return found && w.State == Armed
}

func (s *Service) look(ctx context.Context, w Watch, now time.Time) Watch {
	if !now.Before(w.Expires) {
		return w.Settled(Expired, "термін стеження минув")
	}

	ticker, listed, err := s.deps.Prices.Ticker(ctx, news.Symbol(w.Asset))
	if err != nil || !listed || ticker.Price <= 0 {
		return w
	}

	if w.Asleep(now) {
		return w
	}

	cause, called := w.Called(s.reached(ctx, w, ticker, now), s.deps.Events.Latest(recentEvents), now, s.deps.Rest)
	if !called {
		return w
	}

	w = w.Stirred(cause, ticker.Price)

	verdict, err := s.deps.Judge.Judge(ctx, w, ticker.Price, cause)
	if err != nil {
		s.deps.Log.Warn("plan verdict failed", "asset", w.Asset, "err", err)

		return w.Asked(now, "агент не відповів: "+err.Error()).Postponed(now.Add(s.deps.Rest), s.deps.Lifetime)
	}

	if verdict.Decision == Confirmed {
		return s.enter(ctx, w, ticker, verdict, now)
	}

	return w.Asked(now, verdict.Why).Settled(Refuted, verdict.Why)
}

func (s *Service) reached(ctx context.Context, w Watch, ticker market.Ticker, now time.Time) float64 {
	if w.Woken(ticker.Price) {
		return ticker.Price
	}

	low, high, found := s.deps.Prices.Extremes(ctx, ticker.Contract, w.Since(now.Add(-touchWindow)))
	if !found {
		return ticker.Price
	}

	if w.Plan.WakeAbove > 0 {
		return max(ticker.Price, high)
	}

	return min(ticker.Price, low)
}

func (s *Service) enter(ctx context.Context, w Watch, ticker market.Ticker, verdict Verdict, now time.Time) Watch {
	if fresh, listed, err := s.deps.Prices.Ticker(ctx, news.Symbol(w.Asset)); err == nil && listed && fresh.Price > 0 {
		ticker = fresh
	}

	draft, err := w.Draft(w.ID, ticker.Contract, ticker.Price, s.deps.Desk.Book().Equity, s.deps.Hold, now)
	if err != nil {
		return w.Asked(now, verdict.Why).Settled(Refuted, err.Error())
	}

	if err := s.deps.Desk.Place(ctx, draft); err != nil {
		return w.Asked(now, verdict.Why).Settled(Refuted, "ордер не пройшов перевірки: "+err.Error())
	}

	return w.Asked(now, verdict.Why).Settled(Fired, verdict.Why)
}

func (s *Service) settle(w Watch, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, found := s.watches.Find(w.ID)
	if !found || old.State != Armed || (old.State == w.State && old.Checks == w.Checks) {
		return
	}

	if err := s.keep(s.watches.With(w)); err != nil {
		s.deps.Log.Error("save plan watches", "err", err)

		return
	}

	if w.State == Fired {
		s.supersede(w, now)
	}

	if w.State != Armed {
		s.pending = append(s.pending, notice(w, stateNames[w.State], now))

		return
	}

	if w.Checks > old.Checks {
		s.pending = append(s.pending, notice(w, checkedName, now))
	}
}

func (s *Service) supersede(fired Watch, now time.Time) {
	next := s.watches

	var dropped []Watch

	for _, other := range s.watches.Active() {
		if other.Asset == fired.Asset && other.ID != fired.ID {
			other = other.Settled(Cancelled, supersededNote)
			next = next.With(other)
			dropped = append(dropped, other)
		}
	}

	if len(dropped) == 0 {
		return
	}

	if err := s.keep(next); err != nil {
		s.deps.Log.Error("save plan watches", "err", err)

		return
	}

	for _, w := range dropped {
		s.pending = append(s.pending, notice(w, stateNames[Cancelled], now))
	}
}
