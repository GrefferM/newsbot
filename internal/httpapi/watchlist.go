package httpapi

import (
	"context"
	"net/http"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/watchlist"
)

type Live interface {
	Ticker(ctx context.Context, s news.Symbol) (market.Ticker, bool, error)
}

func routeWatchlist(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /coins", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, priced(r.Context(), deps.Live, kept(deps.Watchlist.Coins())))
	})

	mux.HandleFunc("GET /watchlist", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, priced(r.Context(), deps.Live, kept(deps.Watchlist.Kept())))
	})

	mux.HandleFunc("POST /watchlist", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Asset string `json:"asset"`
			Note  string `json:"note"`
		}

		if !readJSON(w, r, &req) {
			return
		}

		writeResult(w, deps.Watchlist.Add(req.Asset, req.Note))
	})

	mux.HandleFunc("DELETE /watchlist/{asset}", func(w http.ResponseWriter, r *http.Request) {
		writeResult(w, deps.Watchlist.Remove(r.PathValue("asset")))
	})
}

type watchlistDTO struct {
	Built time.Time `json:"built,omitzero"`
	Coins []coinDTO `json:"coins"`
}

type coinDTO struct {
	Asset        string     `json:"asset"`
	Note         string     `json:"note,omitempty"`
	Added        *time.Time `json:"added,omitempty"`
	Known        bool       `json:"known"`
	Reasons      []string   `json:"reasons"`
	Tagged       bool       `json:"tagged"`
	Price        float64    `json:"price,omitempty"`
	Volume       float64    `json:"volume_30d"`
	VolumeDay    float64    `json:"volume_day,omitempty"`
	Change1d     *float64   `json:"change_1d"`
	Change7d     *float64   `json:"change_7d"`
	TrendVsBTC   *float64   `json:"trend_90d_vs_btc"`
	Contract     string     `json:"contract,omitempty"`
	Funding      float64    `json:"funding,omitempty"`
	OpenInterest float64    `json:"open_interest,omitempty"`
	DayRange     float64    `json:"day_range,omitempty"`
	Depth        float64    `json:"depth,omitempty"`
	SpotVolume   *float64   `json:"spot_volume_30d"`
}

func kept(list []watchlist.Kept) watchlistDTO {
	dto := watchlistDTO{Coins: make([]coinDTO, 0, len(list))}

	for _, k := range list {
		c := coinOf(k.Coin, k.Reasons, k.Known)

		if k.Pick.Asset != "" {
			c.Asset = k.Pick.Asset
			c.Note = k.Pick.Note
			c.Added = &k.Pick.Added
		}

		dto.Coins = append(dto.Coins, c)
	}

	return dto
}

func priced(ctx context.Context, live Live, dto watchlistDTO) watchlistDTO {
	for i, c := range dto.Coins {
		if c.Contract == "" {
			continue
		}

		ticker, listed, err := live.Ticker(ctx, news.Symbol(c.Asset))
		if err == nil && listed && ticker.Price > 0 {
			dto.Coins[i].Price = ticker.Price
		}
	}

	return dto
}

func coinOf(c watchlist.Coin, reasons []string, known bool) coinDTO {
	if reasons == nil {
		reasons = []string{}
	}

	return coinDTO{
		Asset:        c.Asset,
		Known:        known,
		Reasons:      reasons,
		Tagged:       c.Tagged,
		Price:        c.Price,
		Volume:       c.Volume,
		VolumeDay:    c.VolumeDay,
		Change1d:     optional(c.Change1d, c.HasChanges),
		Change7d:     optional(c.Change7d, c.HasChanges),
		TrendVsBTC:   trend(c),
		Contract:     c.Contract,
		Funding:      c.Funding,
		OpenInterest: c.OpenInterest,
		DayRange:     c.DayRange,
		Depth:        c.Depth,
		SpotVolume:   optional(c.SpotVolume, c.HasSpot),
	}
}

func trend(c watchlist.Coin) *float64 {
	return optional(c.TrendVsBTC, c.HasTrend)
}

func optional(v float64, ok bool) *float64 {
	if !ok {
		return nil
	}

	return &v
}
