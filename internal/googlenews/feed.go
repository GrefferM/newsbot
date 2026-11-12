package googlenews

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"newsbot/internal/httpclient"
)

const (
	feedURL   = "https://news.google.com/rss/search"
	region    = "&hl=en-US&gl=US&ceid=US:en"
	headlines = 15
)

type Feed struct {
	http *httpclient.Client
	url  string
}

func New(http *httpclient.Client) Feed {
	return Feed{http: http, url: feedURL}
}

type Headline struct {
	Published string `json:"published"`
	Title     string `json:"title"`
	Outlet    string `json:"outlet"`
	URL       string `json:"url"`
}

type rss struct {
	Items []struct {
		Title  string `xml:"title"`
		Link   string `xml:"link"`
		Date   string `xml:"pubDate"`
		Source string `xml:"source"`
	} `xml:"channel>item"`
}

func (f Feed) Headlines(ctx context.Context, query string, days int) ([]Headline, error) {
	asked := url.QueryEscape(fmt.Sprintf("%s when:%dd", query, days))

	body, err := f.http.Bytes(ctx, f.url+"?q="+asked+region)
	if err != nil {
		return nil, err
	}

	return parse(body)
}

func parse(body []byte) ([]Headline, error) {
	var feed rss

	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	out := []Headline{}

	for _, item := range feed.Items[:min(headlines, len(feed.Items))] {
		out = append(out, Headline{
			Published: stamp(item.Date),
			Title:     strings.TrimSuffix(item.Title, " - "+item.Source),
			Outlet:    item.Source,
			URL:       item.Link,
		})
	}

	return out, nil
}

func stamp(published string) string {
	at, err := time.Parse(time.RFC1123, published)
	if err != nil {
		return published
	}

	return at.UTC().Format(time.RFC3339)
}
