package binance

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"newsbot/internal/exchange"
	"newsbot/internal/httpclient"
)

const (
	noticesPerCatalog = 50
	textsPerCall      = 20
	textPause         = 200 * time.Millisecond
	mentionLimit      = 300
	sectionDelisting  = "delisting"
	detailURL         = "https://www.binance.com/bapi/composite/v1/public/cms/article/detail/query?articleCode="
)

var noticeCatalogs = map[string]int{
	sectionDelisting: CatalogDelisting,
	"news":           CatalogNews,
	"listing":        CatalogNewListing,
}

var pairQuotes = []string{"USDT", "USDC", "FDUSD", "BTC"}

type Notices struct {
	http *httpclient.Client

	mu    sync.Mutex
	texts map[string]string
}

func NewNotices(http *httpclient.Client) *Notices {
	return &Notices{http: http, texts: map[string]string{}}
}

type Notice struct {
	Section   string `json:"section"`
	Published string `json:"published"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Mention   string `json:"mention_in_text,omitempty"`
}

func (n *Notices) About(ctx context.Context, asset string) ([]Notice, error) {
	notices := []Notice{}

	for section, catalog := range noticeCatalogs {
		announcements, err := latest(ctx, n.http, catalog, noticesPerCatalog)
		if err != nil {
			return nil, err
		}

		notices = append(notices, pick(section, announcements, asset)...)

		if section == sectionDelisting && asset != "" {
			notices = append(notices, n.inTexts(ctx, announcements, asset)...)
		}
	}

	slices.SortFunc(notices, newestFirst)

	return notices, nil
}

func (n *Notices) inTexts(ctx context.Context, announcements []exchange.Announcement, asset string) []Notice {
	var out []Notice

	read := 0

	for _, a := range announcements {
		if written(a.Title, asset) {
			continue
		}

		if read == textsPerCall {
			break
		}

		read++

		text, err := n.text(ctx, strings.TrimPrefix(a.URL, articleURL))
		if err != nil {
			break
		}

		if found, ok := mention(text, asset); ok {
			out = append(out, Notice{
				Section:   sectionDelisting,
				Published: a.Published.UTC().Format(time.RFC3339),
				Title:     a.Title,
				URL:       a.URL,
				Mention:   found,
			})
		}
	}

	return out
}

func (n *Notices) text(ctx context.Context, code string) (string, error) {
	n.mu.Lock()
	known, ok := n.texts[code]
	n.mu.Unlock()

	if ok {
		return known, nil
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(textPause):
	}

	var reply struct {
		Data struct {
			Body string `json:"body"`
		} `json:"data"`
	}

	if err := n.http.JSON(ctx, detailURL+code, &reply); err != nil {
		return "", err
	}

	text := plain(reply.Data.Body)

	n.mu.Lock()
	n.texts[code] = text
	n.mu.Unlock()

	return text, nil
}

type bodyNode struct {
	Text  string     `json:"text"`
	Tag   string     `json:"tag"`
	Child []bodyNode `json:"child"`
}

func plain(body string) string {
	var root bodyNode

	if err := json.Unmarshal([]byte(body), &root); err != nil {
		return body
	}

	var lines []string

	collect(root, &lines)

	return strings.Join(lines, "\n")
}

func collect(n bodyNode, lines *[]string) {
	if text := strings.TrimSpace(n.Text); text != "" {
		*lines = append(*lines, text)
	}

	for _, child := range n.Child {
		collect(child, lines)
	}
}

func mention(text, asset string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		if !written(line, asset) {
			continue
		}

		if len(line) > mentionLimit {
			line = strings.ToValidUTF8(line[:mentionLimit], "") + "…"
		}

		return line, true
	}

	return "", false
}

func pick(section string, announcements []exchange.Announcement, asset string) []Notice {
	var out []Notice

	for _, a := range announcements {
		if asset != "" && !written(a.Title, asset) {
			continue
		}

		out = append(out, Notice{
			Section:   section,
			Published: a.Published.UTC().Format(time.RFC3339),
			Title:     a.Title,
			URL:       a.URL,
		})
	}

	return out
}

func written(text, asset string) bool {
	for _, word := range strings.FieldsFunc(text, notTicker) {
		if word == asset {
			return true
		}

		if quote, found := strings.CutPrefix(word, asset); found && slices.Contains(pairQuotes, quote) {
			return true
		}
	}

	return false
}

func notTicker(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func newestFirst(a, b Notice) int {
	return cmp.Compare(b.Published, a.Published)
}
