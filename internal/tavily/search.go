package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

const (
	searchURL  = "https://api.tavily.com/search"
	depth      = "basic"
	newsTopic  = "news"
	results    = 10
	timeout    = 30 * time.Second
	replyLimit = 1 << 20
)

var Periods = []string{"day", "week", "month", "year"}

type Client struct {
	http *http.Client
	url  string
	key  string
}

func New(key string) Client {
	return Client{http: &http.Client{Timeout: timeout}, url: searchURL, key: key}
}

type Query struct {
	Text   string
	Period string
	News   bool
}

type Result struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Excerpt   string `json:"excerpt"`
	Published string `json:"published,omitempty"`
}

type request struct {
	Query     string `json:"query"`
	Depth     string `json:"search_depth"`
	Results   int    `json:"max_results"`
	Dated     bool   `json:"include_published_date"`
	TimeRange string `json:"time_range,omitempty"`
	Topic     string `json:"topic,omitempty"`
}

type reply struct {
	Results []struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		Content   string `json:"content"`
		Published string `json:"published_date"`
	} `json:"results"`
}

func (c Client) Search(ctx context.Context, q Query) ([]Result, error) {
	ask := request{Query: q.Text, Depth: depth, Results: results, Dated: true}

	if slices.Contains(Periods, q.Period) {
		ask.TimeRange = q.Period
	}

	if q.News {
		ask.Topic = newsTopic
	}

	payload, err := json.Marshal(ask)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)

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
		return nil, fmt.Errorf("пошук відповів HTTP %d", resp.StatusCode)
	}

	var r reply
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	found := []Result{}

	for _, item := range r.Results {
		found = append(found, Result{Title: item.Title, URL: item.URL, Excerpt: item.Content, Published: item.Published})
	}

	return found, nil
}
