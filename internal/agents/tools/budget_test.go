package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/tavily"
	"newsbot/internal/xapi"
)

type countedSearch struct {
	queries []tavily.Query
}

func (c *countedSearch) Search(_ context.Context, q tavily.Query) ([]tavily.Result, error) {
	c.queries = append(c.queries, q)

	return nil, nil
}

type countedFeed struct {
	queries []string
}

func (c *countedFeed) Recent(_ context.Context, query, order string, _ time.Time) ([]xapi.Post, error) {
	c.queries = append(c.queries, strings.TrimSpace(query+" "+order))

	return nil, nil
}

func TestPaidToolsStopAtTheirBudget(t *testing.T) {
	search := &countedSearch{}
	tool := SearchTool{Search: search, Budget: 2}
	turn := &agents.Turn{Calls: []string{feedName}}

	if _, err := use(t, tool, `{"query":"blur"}`, &agents.Turn{Calls: []string{searchName}}); err == nil {
		t.Error("a paid search before the free feed passed")
	}

	if _, err := use(t, tool, `{"query":" "}`, &agents.Turn{Calls: []string{feedName, searchName}}); err == nil {
		t.Error("an empty query passed")
	}

	for range 3 {
		turn.Calls = append(turn.Calls, fetchName, searchName)

		_, _ = use(t, tool, `{"query":"blur","fresh":"week","news":true}`, turn)
	}

	if len(search.queries) != 2 || search.queries[0] != (tavily.Query{Text: "blur", Period: "week", News: true}) {
		t.Errorf("searches %+v with a budget of 2", search.queries)
	}
}

func TestAnAccountIsReadWithoutRepliesOrReposts(t *testing.T) {
	feed := &countedFeed{}
	tool := PostsTool{Posts: feed, Budget: 1}
	turn := &agents.Turn{Calls: []string{feedName, postsName}}

	if _, err := use(t, tool, `{}`, turn); err == nil {
		t.Error("a read without an account or a query passed")
	}

	if _, err := use(t, tool, `{"account":"@project"}`, turn); err != nil {
		t.Fatal(err)
	}

	turn.Calls = append(turn.Calls, postsName)

	if _, err := use(t, tool, `{"query":"$BANANA"}`, turn); err == nil {
		t.Error("a read beyond the budget passed")
	}

	if len(feed.queries) != 1 || feed.queries[0] != "from:project -is:retweet -is:reply" {
		t.Errorf("queries %q", feed.queries)
	}
}
