package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"newsbot/internal/agents"
	"newsbot/internal/googlenews"
)

type cannedNews map[string][]googlenews.Headline

func (c cannedNews) Headlines(_ context.Context, query string, _ int) ([]googlenews.Headline, error) {
	found, ok := c[query]
	if !ok {
		return nil, errors.New("немає відповіді")
	}

	return found, nil
}

func TestTheFeedAnswersEveryQueryOfOneCallApart(t *testing.T) {
	tool := FeedTool{News: cannedNews{
		"Harmony ONE":    {{Title: "Harmony halts", Outlet: "Crypto News", Published: "2026-11-10"}},
		"Bubblemaps BMT": {},
	}}

	got, err := use(t, tool, `{"queries":["Harmony ONE"," ","Bubblemaps BMT","LayerZero ZRO"]}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`"query":"Harmony ONE","headlines":[{`, `"query":"Bubblemaps BMT","headlines":[]`, `"query":"LayerZero ZRO","headlines":null,"error":"немає відповіді"`} {
		if !strings.Contains(got, want) {
			t.Errorf("no %s in %s", want, got)
		}
	}

	if _, err := use(t, tool, `{"queries":[" "]}`, &agents.Turn{}); err == nil {
		t.Error("a call without a query passed")
	}
}
