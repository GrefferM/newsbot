package tools

import (
	"newsbot/internal/market"
	"newsbot/internal/radar"
)

type pulseMove struct {
	LegMinutes  int        `json:"move_began_minutes_ago"`
	LegMove     float64    `json:"move_since_then_fraction"`
	Change15m   float64    `json:"change_15m_fraction"`
	Change1h    float64    `json:"change_1h_fraction"`
	Change4h    float64    `json:"change_4h_fraction"`
	UsualMove   float64    `json:"usual_1h_move_fraction"`
	VolumeSurge float64    `json:"volume_1h_vs_average"`
	BuyShare    float64    `json:"taker_buy_share_1h_fraction"`
	Crowd       *cardCrowd `json:"crowd,omitempty"`
}

type pulseMarks struct {
	Woke       *cardWake   `json:"woke_up,omitempty"`
	Launch     *cardLaunch `json:"launched_from_quiet,omitempty"`
	TrendHours int         `json:"steady_trend_hours,omitempty"`
	TrendMove  float64     `json:"steady_trend_move_fraction,omitempty"`
}

type cardWake struct {
	Minutes  int     `json:"minutes_ago"`
	Interest float64 `json:"open_interest_5m_fraction"`
	Longs    float64 `json:"long_accounts_15m_shift"`
}

type cardLaunch struct {
	Hours int     `json:"hours_ago"`
	Move  float64 `json:"hour_candle_fraction"`
	Times float64 `json:"times_the_usual_hour"`
}

type cardCrowd struct {
	LongShare  float64 `json:"long_accounts_share_fraction"`
	HourChange float64 `json:"change_1h_fraction"`
}

func moveOf(p market.Pulse) pulseMove {
	move := pulseMove{
		LegMinutes:  p.LegMinutes,
		LegMove:     p.LegMove,
		Change15m:   p.Change15m,
		Change1h:    p.Change1h,
		Change4h:    p.Change4h,
		UsualMove:   p.Sigma,
		VolumeSurge: p.VolumeSurge,
		BuyShare:    p.BuyShare,
	}

	if p.Crowd != nil {
		move.Crowd = &cardCrowd{LongShare: p.Crowd.LongShare, HourChange: p.Crowd.HourChange}
	}

	return move
}

func marksOf(wake *radar.Wake, launch *radar.Launch, trend *market.Run) pulseMarks {
	var marks pulseMarks

	if wake != nil {
		marks.Woke = &cardWake{Minutes: wake.Minutes, Interest: wake.Interest, Longs: wake.Longs}
	}

	if launch != nil {
		marks.Launch = &cardLaunch{Hours: launch.Hours, Move: launch.Move, Times: launch.Times}
	}

	if trend != nil {
		marks.TrendHours, marks.TrendMove = trend.Hours, trend.Move
	}

	return marks
}
