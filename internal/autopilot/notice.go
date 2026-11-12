package autopilot

import (
	"context"
	"fmt"
	"time"

	"newsbot/internal/news"
)

func (s *Service) flush(ctx context.Context, out chan<- news.Event) {
	s.mu.Lock()
	events := s.pending
	s.pending = nil
	s.mu.Unlock()

	for _, e := range events {
		select {
		case out <- e:
		case <-ctx.Done():
			return
		}
	}
}

const (
	supersededNote = "по монеті відкрито угоду за іншим планом"
	checkedName    = "перевірено, чекаю"
)

var stateNames = map[State]string{
	Fired:     "спрацював",
	Refuted:   "спростовано",
	Expired:   "термін минув",
	Cancelled: "знято",
}

func notice(w Watch, kind string, now time.Time) news.Event {
	severity := news.Normal

	if w.State == Fired {
		severity = news.High
	}

	details := []string{w.Plan.Action}

	if w.Woke != "" && w.Checks > 0 {
		details = append(details, fmt.Sprintf("перевірка %d: %s", w.Checks, w.Woke))
	}

	if w.Note != "" {
		details = append(details, w.Note)
	}

	if w.State == Armed && w.After.After(now) {
		details = append(details, "наступна перевірка не раніше "+w.After.UTC().Format("02.01 15:04 UTC"))
	}

	return news.Event{
		ID:       fmt.Sprintf("plan:%s:%s:%d", w.ID, kind, now.UnixNano()),
		Source:   venue,
		Category: news.Trade,
		Kind:     "план · " + kind,
		Title:    fmt.Sprintf("%s: %s", w.Asset, w.Plan.Condition),
		Time:     now,
		Symbols:  []news.Symbol{news.Symbol(w.Asset)},
		Severity: severity,
		Details:  details,
	}
}
