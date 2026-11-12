package tools

import (
	"strings"
	"testing"

	"newsbot/internal/agents"
)

func TestNewsNeedsAPublicationDayAndAFactOnceInATurn(t *testing.T) {
	kept := &shelf{}
	tool := NewsTool{Keeper: kept}
	turn := &agents.Turn{}

	if _, err := use(t, tool, `{"published":"скоро","symbol":"X","text":"?","source":"x"}`, turn); err == nil {
		t.Error("a day that is not a date passed")
	}

	english := `{"published":"2026-11-09","symbol":"banana","text":"Mainnet went live on Banana Gun","source":"blog"}`

	if _, err := use(t, tool, english, turn); err == nil {
		t.Error("a piece of news in English passed")
	}

	named := `{"published":"2026-10-18","symbol":"WLD","text":"World запустила World ID для банків разом із Cloudflare Trusted Agent Delegation","source":"w"}`

	if _, err := use(t, NewsTool{Keeper: &shelf{}}, named, &agents.Turn{}); err != nil {
		t.Errorf("Ukrainian text full of Latin names refused: %v", err)
	}

	first := `{"published":"2026-11-09T13:11:07Z","symbol":"banana","text":"Запуск","source":"blog","outlet":"Blog"}`
	again := `{"published":"2026-11-09","symbol":"BANANA","text":"Мейннет","source":"x","other_sources":["blog"]}`

	if out, err := use(t, tool, first, turn); err != nil || out != "передано" {
		t.Fatalf("first report: %q, %v", out, err)
	}

	if out, err := use(t, tool, again, turn); err != nil || strings.HasPrefix(out, "передано") {
		t.Fatalf("repeat report: %q, %v", out, err)
	}

	if len(turn.Findings) != 1 || len(turn.Findings[0].Sources()) != 2 || turn.Findings[0].Asset != "BANANA" || turn.Findings[0].Outlet != "Blog" {
		t.Errorf("findings %+v", turn.Findings)
	}

	if len(kept.kept) != 2 {
		t.Errorf("every report goes to the keeper, which folds repeats: %+v", kept.kept)
	}
}
