package trading

import (
	"fmt"
	"time"

	"newsbot/internal/news"
)

func Opened(venue string, p Position, book Book, d Decision) news.Event {
	return news.Event{
		ID:       "trade:open:" + p.ID,
		Source:   venue,
		Category: news.Trade,
		Kind:     "вхід",
		Title:    fmt.Sprintf("%s %s %.0f%% після «%s»", titled(p.Side), p.Symbol, p.Size*100, p.Kind),
		Time:     p.Opened,
		Symbols:  []news.Symbol{p.Symbol},
		Severity: news.High,
		Details: append([]string{
			fmt.Sprintf(
				"Вхід %s · SL %s · TP %s",
				news.FormatPrice(p.Entry),
				news.FormatPrice(p.Stop),
				news.FormatPrice(p.Target),
			),
			fmt.Sprintf(
				"Розмір %s з %s · закриття %s",
				news.FormatUSD(p.Stake),
				news.FormatUSD(book.Equity),
				p.Deadline.UTC().Format("02.01 15:04 UTC"),
			),
		}, checkLines(d.Checks)...),
	}
}

func Closed(venue string, p Position, book Book) news.Event {
	return news.Event{
		ID:       "trade:close:" + p.ID,
		Source:   venue,
		Category: news.Trade,
		Kind:     "вихід · " + p.Reason,
		Title:    fmt.Sprintf("Закрито %s %s: %s", p.Side.Name(), p.Symbol, news.FormatPercent(p.Return*100)),
		Time:     p.ClosedAt,
		Symbols:  []news.Symbol{p.Symbol},
		Details: []string{
			fmt.Sprintf("Вхід %s · вихід %s", news.FormatPrice(p.Entry), news.FormatPrice(p.Exit)),
			fmt.Sprintf("Результат %s · рахунок %s", signedUSD(p.PnL), news.FormatUSD(book.Equity)),
		},
	}
}

func Failed(venue string, symbol news.Symbol, action string, err error, now time.Time) news.Event {
	return news.Event{
		ID:       fmt.Sprintf("trade:fail:%s:%s", symbol, now.UTC().Format("20060102T150405")),
		Source:   venue,
		Category: news.Trade,
		Kind:     "помилка",
		Title:    fmt.Sprintf("%s: %s не вдалося", symbol, action),
		Time:     now,
		Symbols:  []news.Symbol{symbol},
		Severity: news.Critical,
		Details:  []string{err.Error()},
	}
}

func titled(side Side) string {
	if side == Long {
		return "Лонг"
	}

	return "Шорт"
}

func checkLines(checks []Check) []string {
	lines := make([]string, 0, len(checks))

	for _, c := range checks {
		mark := "✓"

		if !c.OK {
			mark = "✗"
		}

		lines = append(lines, fmt.Sprintf("%s %s: %s", mark, c.Name, c.Value))
	}

	return lines
}

func signedUSD(v float64) string {
	if v < 0 {
		return "−" + news.FormatUSD(-v)
	}

	return "+" + news.FormatUSD(v)
}
