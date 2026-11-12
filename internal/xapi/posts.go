package xapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	searchURL  = "https://api.x.com/2/tweets/search/recent"
	postURL    = "https://x.com/i/status/"
	perRead    = "10"
	Relevant   = "relevancy"
	fields     = "created_at,public_metrics,note_tweet"
	timeout    = 30 * time.Second
	replyLimit = 1 << 20
)

type Client struct {
	http  *http.Client
	url   string
	token string
}

func New(token string) Client {
	return Client{http: &http.Client{Timeout: timeout}, url: searchURL, token: token}
}

type Post struct {
	Posted  string `json:"posted"`
	Text    string `json:"text"`
	URL     string `json:"url"`
	Likes   int    `json:"likes"`
	Reposts int    `json:"reposts"`
}

type reply struct {
	Data []struct {
		ID      string `json:"id"`
		Text    string `json:"text"`
		Created string `json:"created_at"`
		Note    struct {
			Text string `json:"text"`
		} `json:"note_tweet"`
		Metrics struct {
			Likes   int `json:"like_count"`
			Reposts int `json:"retweet_count"`
		} `json:"public_metrics"`
	} `json:"data"`
}

func (c Client) Recent(ctx context.Context, query, order string, since time.Time) ([]Post, error) {
	params := url.Values{"query": {query}, "max_results": {perRead}, "tweet.fields": {fields}}

	if !since.IsZero() {
		params.Set("start_time", since.UTC().Format(time.RFC3339))
	}

	if order != "" {
		params.Set("sort_order", order)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, replyLimit))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("X відповів HTTP %d", resp.StatusCode)
	}

	var r reply
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	posts := []Post{}

	for _, p := range r.Data {
		text := p.Text

		if p.Note.Text != "" {
			text = p.Note.Text
		}

		posts = append(posts, Post{
			Posted:  p.Created,
			Text:    text,
			URL:     postURL + p.ID,
			Likes:   p.Metrics.Likes,
			Reposts: p.Metrics.Reposts,
		})
	}

	return posts, nil
}
