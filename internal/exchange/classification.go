package exchange

import "newsbot/internal/news"

type Classification struct {
	Category news.Category
	Kind     string
	Symbols  []news.Symbol
	Severity news.Severity
	Trend    news.Direction
}

func (c Classification) Event(e news.Event) news.Event {
	e.Category = c.Category
	e.Kind = c.Kind
	e.Symbols = c.Symbols
	e.Severity = c.Severity
	e.Trend = c.Trend

	return e
}

func shortTrigger(kind string, severity news.Severity, syms []news.Symbol) (Classification, bool) {
	return Classification{
		Category: news.ShortTrigger,
		Kind:     kind,
		Symbols:  syms,
		Severity: severity,
		Trend:    news.Down,
	}, true
}

func ignored() (Classification, bool) {
	return Classification{}, false
}
