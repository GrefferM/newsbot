package deribit

import (
	"context"
	"slices"
	"strings"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/schedule"
)

const (
	instrumentsURL = "https://www.deribit.com/api/v2/public/get_instruments?kind=option&expired=false&currency="
	pageURL        = "https://www.deribit.com/statistics/BTC/options-expiry"
	sourceName     = "Deribit"
	week           = 7 * 24 * time.Hour
	quarter        = 3
)

var currencies = []string{"BTC", "ETH"}

type Expiries struct {
	http *httpclient.Client
	url  string
}

func NewExpiries(http *httpclient.Client) Expiries {
	return Expiries{http: http, url: instrumentsURL}
}

func (e Expiries) Name() string {
	return "deribit"
}

type instruments struct {
	Result []struct {
		Expiration int64 `json:"expiration_timestamp"`
	} `json:"result"`
}

func (e Expiries) Entries(ctx context.Context) ([]schedule.Entry, error) {
	expiring := map[time.Time][]string{}

	for _, currency := range currencies {
		var list instruments

		if err := e.http.JSON(ctx, e.url+currency, &list); err != nil {
			return nil, err
		}

		seen := map[int64]bool{}

		for _, i := range list.Result {
			if seen[i.Expiration] {
				continue
			}

			seen[i.Expiration] = true
			at := time.UnixMilli(i.Expiration).UTC()
			expiring[at] = append(expiring[at], currency)
		}
	}

	return monthly(expiring), nil
}

func monthly(expiring map[time.Time][]string) []schedule.Entry {
	out := []schedule.Entry{}

	for at, coins := range expiring {
		if at.Weekday() != time.Friday || at.Add(week).Month() == at.Month() {
			continue
		}

		kind := "Місячна"

		if int(at.Month())%quarter == 0 {
			kind = "Квартальна"
		}

		slices.Sort(coins)

		out = append(out, schedule.NewEntry(sourceName, at, kind+" експірація опціонів "+strings.Join(coins, ", "), pageURL).Of(schedule.Expiry, coins...))
	}

	return out
}
