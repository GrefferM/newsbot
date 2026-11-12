package defillama

import (
	"context"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/incident"
	"newsbot/internal/news"
)

const (
	stablecoinsURL = "https://stablecoins.llama.fi/stablecoins?includePrices=true"
	usdPeg         = "peggedUSD"
)

var stablecoinsOrigin = news.Origin{Name: "DefiLlama", URL: "https://defillama.com/stablecoins"}

type Stablecoins struct {
	http     *httpclient.Client
	detector *incident.DepegDetector
	every    time.Duration
}

func NewStablecoins(http *httpclient.Client, rules incident.DepegRules, every time.Duration) *Stablecoins {
	return &Stablecoins{
		http:     http,
		detector: incident.NewDepegDetector(rules, stablecoinsOrigin),
		every:    every,
	}
}

func (s *Stablecoins) Name() string {
	return "stablecoin-depeg"
}

func (s *Stablecoins) Interval() time.Duration {
	return s.every
}

type stablecoinsResponse struct {
	PeggedAssets []struct {
		Name        string             `json:"name"`
		Symbol      string             `json:"symbol"`
		PegType     string             `json:"pegType"`
		Price       *float64           `json:"price"`
		Circulating map[string]float64 `json:"circulating"`
	} `json:"peggedAssets"`
}

func (s *Stablecoins) Poll(ctx context.Context) ([]news.Event, error) {
	var resp stablecoinsResponse
	if err := s.http.JSON(ctx, stablecoinsURL, &resp); err != nil {
		return nil, err
	}

	return s.detector.Detect(resp.dollars(), time.Now()), nil
}

func (r stablecoinsResponse) dollars() []incident.Stablecoin {
	var coins []incident.Stablecoin

	for _, a := range r.PeggedAssets {
		if a.PegType != usdPeg || a.Price == nil {
			continue
		}

		coins = append(coins, incident.Stablecoin{
			Name:   a.Name,
			Symbol: a.Symbol,
			Price:  *a.Price,
			Supply: a.Circulating[usdPeg],
		})
	}

	return coins
}
