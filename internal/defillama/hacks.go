package defillama

import (
	"context"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/incident"
	"newsbot/internal/news"
)

const hacksURL = "https://api.llama.fi/hacks"

var hacksOrigin = news.Origin{Name: "DefiLlama", URL: "https://defillama.com/hacks"}

type Hacks struct {
	http   *httpclient.Client
	policy incident.HackPolicy
	every  time.Duration
}

func NewHacks(http *httpclient.Client, policy incident.HackPolicy, every time.Duration) *Hacks {
	return &Hacks{
		http:   http,
		policy: policy,
		every:  every,
	}
}

func (h *Hacks) Name() string {
	return "defillama-hacks"
}

func (h *Hacks) Interval() time.Duration {
	return h.every
}

type rawHack struct {
	Date           int64    `json:"date"`
	Name           string   `json:"name"`
	Classification string   `json:"classification"`
	Technique      string   `json:"technique"`
	Amount         *float64 `json:"amount"`
	Chain          []string `json:"chain"`
	Source         string   `json:"source"`
	ReturnedFunds  *float64 `json:"returnedFunds"`
}

func (h *Hacks) Poll(ctx context.Context) ([]news.Event, error) {
	var raws []rawHack
	if err := h.http.JSON(ctx, hacksURL, &raws); err != nil {
		return nil, err
	}

	hacks := make([]incident.Hack, 0, len(raws))

	for _, r := range raws {
		hacks = append(hacks, r.toDomain())
	}

	return h.policy.Events(hacksOrigin, hacks, time.Now()), nil
}

func (r rawHack) toDomain() incident.Hack {
	return incident.Hack{
		Name:           r.Name,
		Classification: r.Classification,
		Technique:      r.Technique,
		Chains:         r.Chain,
		SourceURL:      r.Source,
		Date:           time.Unix(r.Date, 0).UTC(),
		Amount:         valueOf(r.Amount),
		Returned:       valueOf(r.ReturnedFunds),
		AmountKnown:    r.Amount != nil,
	}
}

func valueOf(p *float64) float64 {
	if p == nil {
		return 0
	}

	return *p
}
