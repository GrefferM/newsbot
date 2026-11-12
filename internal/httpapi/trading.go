package httpapi

import (
	"io"
	"net/http"
	"slices"
	"time"

	"newsbot/internal/autopilot"
	"newsbot/internal/autotrade"
	"newsbot/internal/trading"
)

func routeTrading(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /positions", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, positions(deps.Trader.Venue(), deps.Trader.Book()))
	})

	mux.HandleFunc("GET /auto", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, strategies(deps.Auto.Strategies()))
	})

	mux.HandleFunc("PUT /auto/{id}", func(w http.ResponseWriter, r *http.Request) {
		var wanted switchDTO

		if !readJSON(w, r, &wanted) {
			return
		}

		writeResult(w, deps.Auto.Switch(r.PathValue("id"), wanted.On))
	})

	mux.HandleFunc("PUT /auto/{id}/rule", func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, ruleBytes))
		if err != nil {
			writeResult(w, err)

			return
		}

		writeResult(w, deps.Auto.Tune(r.PathValue("id"), raw))
	})

	mux.HandleFunc("GET /results", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, resultsDTO{Plans: deps.Autopilot.Tally(), Trades: deps.Trader.Book().Record()})
	})

	mux.HandleFunc("POST /positions/{id}/close", func(w http.ResponseWriter, r *http.Request) {
		writeResult(w, deps.Trader.ClosePosition(r.Context(), r.PathValue("id")))
	})
}

const ruleBytes = 1 << 16

type autoDTO struct {
	Strategies []strategyDTO `json:"strategies"`
}

type strategyDTO struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	About  string         `json:"about"`
	Terms  []termDTO      `json:"terms"`
	On     bool           `json:"on"`
	Record trading.Record `json:"record"`
	Rule   any            `json:"rule"`
}

type termDTO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type switchDTO struct {
	On bool `json:"on"`
}

func strategies(all []autotrade.Strategy) autoDTO {
	dto := autoDTO{Strategies: make([]strategyDTO, 0, len(all))}

	for _, s := range all {
		terms := make([]termDTO, 0, len(s.Terms))

		for _, t := range s.Terms {
			terms = append(terms, termDTO{Name: t.Name, Value: t.Value})
		}

		dto.Strategies = append(dto.Strategies, strategyDTO{ID: s.ID, Name: s.Name, About: s.About, Terms: terms, On: s.On, Record: s.Record, Rule: s.Rule})
	}

	return dto
}

type resultsDTO struct {
	Plans  autopilot.Tally `json:"plans"`
	Trades trading.Record  `json:"trades"`
}

type bookDTO struct {
	Venue     string        `json:"venue"`
	Start     float64       `json:"start"`
	Equity    float64       `json:"equity"`
	Exposure  float64       `json:"exposure"`
	Positions []positionDTO `json:"positions"`
}

type positionDTO struct {
	ID       string    `json:"id"`
	Symbol   string    `json:"symbol"`
	Contract string    `json:"contract"`
	Side     string    `json:"side"`
	Kind     string    `json:"kind"`
	Size     float64   `json:"size"`
	Stake    float64   `json:"stake"`
	Quantity float64   `json:"quantity"`
	Entry    float64   `json:"entry"`
	Stop     float64   `json:"stop"`
	Target   float64   `json:"target"`
	Opened   time.Time `json:"opened"`
	Deadline time.Time `json:"deadline"`
	Open     bool      `json:"open"`
	Exit     float64   `json:"exit,omitempty"`
	ClosedAt time.Time `json:"closed_at,omitzero"`
	Reason   string    `json:"reason,omitempty"`
	Return   float64   `json:"return,omitempty"`
	PnL      float64   `json:"pnl,omitempty"`
	Booked   bool      `json:"booked_by_exchange"`
}

func positions(venue string, book trading.Book) bookDTO {
	dto := bookDTO{
		Venue:     venue,
		Start:     book.Start,
		Equity:    book.Equity,
		Exposure:  book.Exposure(),
		Positions: make([]positionDTO, 0, len(book.Positions)),
	}

	for _, p := range slices.Backward(book.Positions) {
		dto.Positions = append(dto.Positions, positionDTO{
			ID:       p.ID,
			Symbol:   string(p.Symbol),
			Contract: p.Contract,
			Side:     sideOf(p.Side),
			Kind:     p.Kind,
			Size:     p.Size,
			Stake:    p.Stake,
			Quantity: p.Quantity,
			Entry:    p.Entry,
			Stop:     p.Stop,
			Target:   p.Target,
			Opened:   p.Opened,
			Deadline: p.Deadline,
			Open:     p.Open(),
			Exit:     p.Exit,
			ClosedAt: p.ClosedAt,
			Reason:   p.Reason,
			Return:   p.Return,
			PnL:      p.PnL,
			Booked:   p.Booked,
		})
	}

	return dto
}

func sideOf(side trading.Side) string {
	if side == trading.Long {
		return string(trading.Long)
	}

	return string(trading.Short)
}
