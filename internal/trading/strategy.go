package trading

import (
	"cmp"
	"fmt"
	"time"

	"newsbot/internal/news"
)

type Strategy struct {
	StopLoss     float64
	TakeProfit   float64
	Hold         time.Duration
	EstimatedFee float64
}

type Limits struct {
	MaxExposure    float64
	MinVolume      float64
	CrowdedFunding float64
	DailyLoss      float64
}

type Market struct {
	Listed      bool
	Refused     string
	Contract    string
	Price       float64
	Funding     float64
	QuoteVolume float64
	MinSize     float64
}

type Check struct {
	Name  string
	OK    bool
	Value string
}

type Decision struct {
	Symbol news.Symbol
	Side   Side
	Kind   string
	Size   float64
	Market Market
	Checks []Check
}

func (l Limits) Checks(m Market, side Side, size, exposure float64) []Check {
	if !m.Listed {
		return []Check{{Name: "Перп торгується", Value: cmp.Or(m.Refused, "немає")}}
	}

	checks := []Check{
		{Name: "Перп торгується", OK: true, Value: m.Contract},
		{
			Name:  "Експозиція після входу",
			OK:    exposure+size <= l.MaxExposure+1e-9 && exposure < l.MaxExposure-1e-9,
			Value: fmt.Sprintf("%.0f%% з %.0f%%", (exposure+size)*100, l.MaxExposure*100),
		},
		l.crowding(m, side),
		{
			Name:  "Ліквідність",
			OK:    m.QuoteVolume >= l.MinVolume,
			Value: "обсяг " + news.FormatUSD(m.QuoteVolume),
		},
	}

	if m.MinSize > 0 {
		checks = append(checks, Check{
			Name:  "Розмір не менший за найменший ордер біржі",
			OK:    size == 0 || size >= m.MinSize-1e-9,
			Value: fmt.Sprintf("найменший ордер %.2f%% рахунку", m.MinSize*100),
		})
	}

	return checks
}

func (l Limits) crowding(m Market, side Side) Check {
	check := Check{
		Name:  "Шорт не переповнений",
		OK:    m.Funding >= l.CrowdedFunding,
		Value: fmt.Sprintf("фандинг %+.3f%%", m.Funding*100),
	}

	if side == Long {
		check.Name = "Лонг не переповнений"
		check.OK = m.Funding <= -l.CrowdedFunding
	}

	return check
}

const KindPlan = "план на стеженні"

type Draft struct {
	ID         string
	Symbol     news.Symbol
	Side       Side
	Contract   string
	Stake      float64
	StopLoss   float64
	TakeProfit float64
	Hold       time.Duration
	Note       string
	Created    time.Time
}

func (s Strategy) For(d Draft) Strategy {
	s.StopLoss = d.StopLoss
	s.TakeProfit = d.TakeProfit
	s.Hold = d.Hold

	return s
}
