package coingecko

import (
	"cmp"
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"sync"

	"newsbot/internal/httpclient"
)

const (
	apiURL     = "https://api.coingecko.com/api/v3"
	coinQuery  = "?localization=false&tickers=false&market_data=false&community_data=false&developer_data=false"
	unranked   = 1 << 30
	xAccountAt = "https://x.com/"
)

var ErrUnknown = errors.New("CoinGecko не знає монети з таким тікером")

type Directory struct {
	http *httpclient.Client
	url  string

	mu    sync.Mutex
	known map[string]Links
}

func NewDirectory(http *httpclient.Client) *Directory {
	return &Directory{http: http, url: apiURL, known: map[string]Links{}}
}

type candidate struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Rank   *int   `json:"market_cap_rank"`
}

type coinReply struct {
	Name  string `json:"name"`
	Links struct {
		Homepage   []string `json:"homepage"`
		Whitepaper string   `json:"whitepaper"`
		X          string   `json:"twitter_screen_name"`
		Telegram   string   `json:"telegram_channel_identifier"`
		Chats      []string `json:"chat_url"`
		Announce   []string `json:"announcement_url"`
	} `json:"Links"`
}

type Links struct {
	Project    string   `json:"project"`
	Sites      []string `json:"sites"`
	Docs       string   `json:"docs,omitempty"`
	XAccount   string   `json:"x_account,omitempty"`
	XURL       string   `json:"x_url,omitempty"`
	Telegram   string   `json:"telegram_channel,omitempty"`
	Other      []string `json:"other_official_pages,omitempty"`
	SameTicker []string `json:"other_coins_with_this_ticker,omitempty"`
}

func (d *Directory) Links(ctx context.Context, asset string) (Links, error) {
	if found, ok := d.recall(asset); ok {
		return found, nil
	}

	var search struct {
		Coins []candidate `json:"coins"`
	}

	if err := d.http.JSON(ctx, d.url+"/search?query="+url.QueryEscape(asset), &search); err != nil {
		return Links{}, err
	}

	matches := named(search.Coins, asset)
	if len(matches) == 0 {
		return Links{}, ErrUnknown
	}

	var coin coinReply
	if err := d.http.JSON(ctx, d.url+"/coins/"+url.PathEscape(matches[0].ID)+coinQuery, &coin); err != nil {
		return Links{}, err
	}

	found := describe(coin, matches[1:])

	d.remember(asset, found)

	return found, nil
}

func named(coins []candidate, asset string) []candidate {
	var out []candidate

	for _, c := range coins {
		if strings.EqualFold(c.Symbol, asset) {
			out = append(out, c)
		}
	}

	slices.SortStableFunc(out, byRank)

	return out
}

func byRank(a, b candidate) int {
	return cmp.Compare(rank(a), rank(b))
}

func rank(c candidate) int {
	if c.Rank == nil {
		return unranked
	}

	return *c.Rank
}

func describe(coin coinReply, rest []candidate) Links {
	out := Links{
		Project:  coin.Name,
		Sites:    filled(coin.Links.Homepage),
		Docs:     coin.Links.Whitepaper,
		XAccount: coin.Links.X,
		Telegram: coin.Links.Telegram,
		Other:    filled(append(coin.Links.Announce, coin.Links.Chats...)),
	}

	if out.XAccount != "" {
		out.XURL = xAccountAt + out.XAccount
	}

	for _, c := range rest {
		out.SameTicker = append(out.SameTicker, c.Name)
	}

	return out
}

func filled(values []string) []string {
	out := []string{}

	for _, v := range values {
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}

func (d *Directory) recall(asset string) (Links, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	found, ok := d.known[asset]

	return found, ok
}

func (d *Directory) remember(asset string, found Links) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.known[asset] = found
}
