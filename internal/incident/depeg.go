package incident

import (
	"fmt"
	"math"
	"time"

	"newsbot/internal/news"
)

type Stablecoin struct {
	Name   string
	Symbol string
	Price  float64
	Supply float64
}

func (s Stablecoin) Deviation() float64 {
	return s.Price - 1
}

type DepegRules struct {
	MinSupply float64
	Threshold float64
	Dead      float64
	Cooldown  time.Duration
}

type DepegDetector struct {
	rules     DepegRules
	origin    news.Origin
	offPeg    map[string]bool
	lastAlert map[string]time.Time
}

func NewDepegDetector(rules DepegRules, origin news.Origin) *DepegDetector {
	return &DepegDetector{
		rules:     rules,
		origin:    origin,
		offPeg:    map[string]bool{},
		lastAlert: map[string]time.Time{},
	}
}

func (d *DepegDetector) Detect(coins []Stablecoin, now time.Time) []news.Event {
	var out []news.Event

	for _, c := range coins {
		if !d.breaksPeg(c, now) {
			continue
		}

		d.lastAlert[c.Symbol] = now
		out = append(out, d.event(c, now))
	}

	return out
}

func (d *DepegDetector) breaksPeg(c Stablecoin, now time.Time) bool {
	wasOff := d.offPeg[c.Symbol]
	d.offPeg[c.Symbol] = d.depegged(c)

	if !d.offPeg[c.Symbol] || wasOff || d.dead(c) {
		return false
	}

	return now.Sub(d.lastAlert[c.Symbol]) >= d.rules.Cooldown
}

func (d *DepegDetector) depegged(c Stablecoin) bool {
	return c.Supply >= d.rules.MinSupply && math.Abs(c.Deviation()) >= d.rules.Threshold
}

func (d *DepegDetector) dead(c Stablecoin) bool {
	return -c.Deviation() >= d.rules.Dead
}

func (d *DepegDetector) event(c Stablecoin, now time.Time) news.Event {
	return news.Event{
		ID:       fmt.Sprintf("depeg:%s:%d", c.Symbol, now.Unix()),
		Source:   d.origin.Name,
		Category: news.Incident,
		Kind:     "Депег стейблкоїна",
		Title:    fmt.Sprintf("%s (%s) = %s", c.Name, c.Symbol, news.FormatPrice(c.Price)),
		URL:      d.origin.URL,
		Time:     now.UTC(),
		Symbols:  []news.Symbol{news.Symbol(c.Symbol)},
		Details: []string{
			"Відхилення: " + news.FormatPercent(c.Deviation()*100),
			"В обігу: " + news.FormatUSD(c.Supply),
		},
		Severity: max(news.High, news.SeverityOf(math.Abs(c.Deviation()), 0.01, 0.03)),
		Trend:    news.Down,
	}
}
