package httpapi

import (
	"net/http"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/radar"
)

func routeRadar(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /pulse", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, pulseOf(deps.Radar.Latest()))
	})

	mux.HandleFunc("GET /tradfi", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, pulseOf(deps.Radar.LatestTradFi()))
	})
}

type pulseDTO struct {
	At    time.Time      `json:"at,omitzero"`
	Bench float64        `json:"btc_1h"`
	Coins []pulseCoinDTO `json:"coins"`
}

type pulseCoinDTO struct {
	Asset         string        `json:"asset"`
	Contract      string        `json:"contract"`
	Price         float64       `json:"price"`
	Change15m     float64       `json:"change_15m"`
	Change1h      float64       `json:"change_1h"`
	Change4h      float64       `json:"change_4h"`
	Change24h     float64       `json:"change_24h"`
	VolumeHour    float64       `json:"volume_1h"`
	VolumeSurge   float64       `json:"volume_surge"`
	BuyShare      float64       `json:"buy_share"`
	RangePosition float64       `json:"range_position"`
	Funding       float64       `json:"funding_8h"`
	VolumeDay     float64       `json:"volume_24h"`
	Woke          *wakeDTO      `json:"woke,omitempty"`
	Launch        *launchDTO    `json:"launch,omitempty"`
	LegMinutes    int           `json:"leg_minutes"`
	LegMove       float64       `json:"leg_move"`
	TrendHours    int           `json:"trend_hours,omitempty"`
	TrendMove     float64       `json:"trend_move,omitempty"`
	CrowdLong     *float64      `json:"crowd_long,omitempty"`
	CrowdChange   *float64      `json:"crowd_change_1h,omitempty"`
	Stories       []radar.Story `json:"headlines"`
}

type wakeDTO struct {
	Minutes  int     `json:"minutes"`
	Interest float64 `json:"interest_5m"`
	Longs    float64 `json:"longs_15m"`
}

type launchDTO struct {
	Hours int     `json:"hours"`
	Move  float64 `json:"move"`
	Times float64 `json:"times"`
}

func pulseOf(board radar.Board) pulseDTO {
	dto := pulseDTO{At: board.At, Bench: board.Bench, Coins: make([]pulseCoinDTO, 0, len(board.Coins))}

	for _, c := range board.Coins {
		headlines := c.Stories

		if headlines == nil {
			headlines = []radar.Story{}
		}

		var crowdLong, crowdChange *float64

		if crowd := c.Pulse.Crowd; crowd != nil {
			crowdLong, crowdChange = &crowd.LongShare, &crowd.HourChange
		}

		var woke *wakeDTO

		if c.Wake != nil {
			woke = &wakeDTO{Minutes: c.Wake.Minutes, Interest: c.Wake.Interest, Longs: c.Wake.Longs}
		}

		var launch *launchDTO

		if c.Launch != nil {
			launch = &launchDTO{Hours: c.Launch.Hours, Move: c.Launch.Move, Times: c.Launch.Times}
		}

		var trend market.Run

		if c.Trend != nil {
			trend = *c.Trend
		}

		dto.Coins = append(dto.Coins, pulseCoinDTO{
			TrendHours:    trend.Hours,
			TrendMove:     trend.Move,
			LegMinutes:    c.Pulse.LegMinutes,
			LegMove:       c.Pulse.LegMove,
			CrowdLong:     crowdLong,
			CrowdChange:   crowdChange,
			Asset:         c.Pulse.Asset,
			Contract:      c.Pulse.Contract,
			Price:         c.Pulse.Price,
			Change15m:     c.Pulse.Change15m,
			Change1h:      c.Pulse.Change1h,
			Change4h:      c.Pulse.Change4h,
			Change24h:     c.Pulse.Change24h,
			VolumeHour:    c.Pulse.VolumeHour,
			VolumeSurge:   c.Pulse.VolumeSurge,
			BuyShare:      c.Pulse.BuyShare,
			RangePosition: c.Pulse.RangePosition,
			Funding:       c.Pulse.Funding,
			VolumeDay:     c.Pulse.VolumeDay,
			Woke:          woke,
			Launch:        launch,
			Stories:       headlines,
		})
	}

	return dto
}
