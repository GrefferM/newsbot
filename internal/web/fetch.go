package web

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	fetchTimeout = 30 * time.Second
	pageLimit    = 2 << 20
	agentName    = "newsbot-agent/1.0"
)

var (
	anchors = regexp.MustCompile(`(?is)<a\s[^>]*?href\s*=\s*["']([^"']+)["'][^>]*>(.*?)</a>`)
	scripts = regexp.MustCompile(`(?is)<(script|style|noscript|svg)[^>]*>.*?</(script|style|noscript|svg)>`)
	tags    = regexp.MustCompile(`(?s)<[^>]+>`)
	blank   = regexp.MustCompile(`[ \t\r\f\v]+`)
	breaks  = strings.NewReplacer(
		"<br>", "\n", "</p>", "\n", "</div>", "\n", "</li>", "\n",
		"</h1>", "\n", "</h2>", "\n", "</h3>", "\n", "</tr>", "\n",
	)
)

var ErrAddress = errors.New("потрібна адреса http(s)")

type Fetcher struct {
	http *http.Client
}

func NewFetcher() Fetcher {
	return Fetcher{http: &http.Client{Timeout: fetchTimeout}}
}

func (f Fetcher) Fetch(ctx context.Context, address string) (string, error) {
	target, err := url.Parse(address)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return "", ErrAddress
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", agentName)

	resp, err := f.http.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, pageLimit))
	if err != nil {
		return "", err
	}

	page := string(body)

	if strings.Contains(resp.Header.Get("Content-Type"), "html") {
		page = Text(linked(page, target))
	}

	return "HTTP " + resp.Status + "\n" + page, nil
}

func linked(page string, base *url.URL) string {
	return anchors.ReplaceAllStringFunc(page, func(anchor string) string {
		parts := anchors.FindStringSubmatch(anchor)

		target, err := base.Parse(html.UnescapeString(strings.TrimSpace(parts[1])))
		if err != nil || (target.Scheme != "http" && target.Scheme != "https") {
			return parts[2]
		}

		target.Fragment = ""

		if target.String() == base.String() {
			return parts[2]
		}

		return parts[2] + " (" + target.String() + ")"
	})
}

func Text(page string) string {
	page = scripts.ReplaceAllString(page, " ")
	page = breaks.Replace(page)
	page = html.UnescapeString(tags.ReplaceAllString(page, " "))
	page = strings.ReplaceAll(page, "\u00a0", " ")

	var out []string

	for _, line := range strings.Split(page, "\n") {
		line = strings.TrimSpace(blank.ReplaceAllString(line, " "))

		if line != "" {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n\n")
}
