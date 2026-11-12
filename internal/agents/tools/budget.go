package tools

import (
	"errors"

	"newsbot/internal/agents"
)

const (
	fetchName   = "http_get"
	searchName  = "web_search"
	postsName   = "x_posts"
	feedName    = "news_feed"
	feedDays    = 7
	feedLimit   = 30
	feedQueries = 12
	ownPosts    = " -is:retweet -is:reply"
)

func paid(turn *agents.Turn, tool string, budget int) error {
	if spent(turn, feedName) == 0 {
		return errors.New("спершу безкоштовна news_feed: платний виклик — лише для того, чого вона не закрила")
	}

	if spent(turn, tool) > budget {
		return errors.New("платні виклики цього інструмента на це питання вичерпано: працюй з уже знайденим")
	}

	return nil
}

func spent(turn *agents.Turn, tool string) int {
	count := 0

	for _, name := range turn.Calls {
		if name == tool {
			count++
		}
	}

	return count
}
