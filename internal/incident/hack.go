package incident

import (
	"fmt"
	"strings"
	"time"

	"newsbot/internal/news"
)

type Hack struct {
	Name           string
	Classification string
	Technique      string
	Chains         []string
	SourceURL      string
	Date           time.Time
	Amount         float64
	Returned       float64
	AmountKnown    bool
}

type HackPolicy struct {
	MinAmount float64
	MaxAge    time.Duration
}

func (p HackPolicy) Events(origin news.Origin, hacks []Hack, now time.Time) []news.Event {
	var out []news.Event

	for _, h := range hacks {
		if h.Date.Before(now.Add(-p.MaxAge)) {
			continue
		}

		e := h.event(origin)

		if !h.AmountKnown || h.Amount < p.MinAmount {
			e.Muted = "нижче порогу"
		}

		out = append(out, e)
	}

	return out
}

func (h Hack) event(origin news.Origin) news.Event {
	return news.Event{
		ID:       fmt.Sprintf("hack:%s:%d", h.Name, h.Date.Unix()),
		Source:   origin.Name,
		Category: news.Incident,
		Kind:     "Злам · " + h.Classification,
		Title:    h.Name,
		URL:      h.SourceURL,
		Time:     h.Date,
		Details:  h.details(),
		Severity: news.SeverityOf(h.Amount, 5e6, 50e6),
		Trend:    news.Down,
	}
}

func (h Hack) details() []string {
	details := []string{"Втрати: " + news.FormatUSD(h.Amount)}

	if h.Technique != "" {
		details = append(details, "Метод: "+h.Technique)
	}

	if len(h.Chains) > 0 {
		details = append(details, "Мережа: "+strings.Join(h.Chains, ", "))
	}

	if h.Returned > 0 {
		details = append(details, "Повернуто: "+news.FormatUSD(h.Returned))
	}

	return details
}
