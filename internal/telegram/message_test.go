package telegram

import (
	"strings"
	"testing"
	"time"

	"newsbot/internal/news"
)

func TestMessageLayout(t *testing.T) {
	e := news.Event{
		Source:   "Binance",
		Category: news.ShortTrigger,
		Kind:     "Делістинг перпа Binance",
		Title:    "Binance Futures Will Delist <PTB>",
		URL:      "https://example.com/a?b=1&c=2",
		Time:     time.Date(2026, 11, 8, 12, 27, 0, 0, time.UTC),
		Details:  []string{"PTB: перпа на Binance немає"},
	}

	want := "<b>🔴 Шорт-тригер · Делістинг перпа Binance</b>\n" +
		"Binance Futures Will Delist &lt;PTB&gt;\n" +
		"• PTB: перпа на Binance немає\n" +
		"\n" +
		"<a href=\"https://example.com/a?b=1&amp;c=2\">Binance</a> · 08.11 14:27 Київ"

	if got := Message(e); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestMessageWithoutURL(t *testing.T) {
	e := news.Event{
		Source:   "Rekt",
		Category: news.Incident,
		Kind:     "Новина",
		Title:    "x",
		Time:     time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC),
	}

	if got, want := Message(e), "<b>🚨 Інцидент · Новина</b>\nx\n\nRekt · 02.01 05:04 Київ"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMessageShowsSeverityAndTrend(t *testing.T) {
	e := news.Event{
		Source:   "Binance",
		Category: news.ShortTrigger,
		Kind:     "Monitoring Tag Binance",
		Title:    "Binance Will Extend the Monitoring Tag to Include ABC",
		Severity: news.Critical,
		Trend:    news.Down,
		Time:     time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC),
	}

	want := strings.Join([]string{
		"🚨🚨🚨 <b>КРИТИЧНО</b>",
		"<b>🔴 Шорт-тригер · Monitoring Tag Binance</b>",
		"📉 Binance Will Extend the Monitoring Tag to Include ABC",
		"",
		"Binance · 02.01 05:04 Київ",
	}, "\n")

	if got := Message(e); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
