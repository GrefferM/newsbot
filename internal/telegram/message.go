package telegram

import (
	"fmt"
	"html"
	"strings"
	_ "time/tzdata"

	"newsbot/internal/news"
)

var (
	headers = map[news.Category]string{
		news.ShortTrigger: "🔴 Шорт-тригер",
		news.Incident:     "🚨 Інцидент",
		news.Trade:        "📝 Торгівля",
		news.Reminder:     "📅 Календар",
	}

	severityBanners = map[news.Severity]string{
		news.Critical: "🚨🚨🚨 <b>КРИТИЧНО</b>",
		news.High:     "⚠️ <b>ВАЖЛИВО</b>",
	}

	trendMarks = map[news.Direction]string{
		news.Up:    "📈 ",
		news.Down:  "📉 ",
		news.Mixed: "↕️ ",
	}
)

func Message(e news.Event) string {
	var b strings.Builder

	if banner, ok := severityBanners[e.Severity]; ok {
		fmt.Fprintf(&b, "%s\n", banner)
	}

	fmt.Fprintf(&b, "<b>%s · %s</b>\n", headers[e.Category], html.EscapeString(e.Kind))
	fmt.Fprintf(&b, "%s%s\n", trendMarks[e.Trend], html.EscapeString(e.Title))

	for _, d := range e.Details {
		fmt.Fprintf(&b, "• %s\n", html.EscapeString(d))
	}

	b.WriteString("\n")
	b.WriteString(sourceLink(e))
	b.WriteString(e.Time.In(news.Kyiv).Format("02.01 15:04") + " Київ")

	return b.String()
}

func sourceLink(e news.Event) string {
	source := html.EscapeString(e.Source)

	if e.URL == "" {
		return source + " · "
	}

	return fmt.Sprintf("<a href=\"%s\">%s</a> · ", html.EscapeString(e.URL), source)
}
