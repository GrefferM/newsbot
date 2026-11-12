package binance

import (
	"context"
	"fmt"
	"time"

	"newsbot/internal/exchange"
	"newsbot/internal/httpclient"
)

const (
	CatalogNewListing = 48
	CatalogNews       = 49
	CatalogDelisting  = 161

	listURL    = "https://www.binance.com/bapi/composite/v1/public/cms/article/list/query" + listQuery
	listQuery  = "?type=1&catalogId=%d&pageNo=1&pageSize=%d"
	articleURL = "https://www.binance.com/en/support/announcement/detail/"
	catalogURL = "https://www.binance.com/en/support/announcement/list/%d"
)

type listResponse struct {
	Data struct {
		Catalogs []struct {
			Articles []struct {
				Code        string `json:"code"`
				Title       string `json:"title"`
				ReleaseDate int64  `json:"releaseDate"`
			} `json:"articles"`
		} `json:"catalogs"`
	} `json:"data"`
}

func latest(ctx context.Context, http *httpclient.Client, catalog, size int) ([]exchange.Announcement, error) {
	var resp listResponse
	if err := http.JSON(ctx, fmt.Sprintf(listURL, catalog, size), &resp); err != nil {
		return nil, err
	}

	var out []exchange.Announcement

	for _, catalog := range resp.Data.Catalogs {
		for _, article := range catalog.Articles {
			out = append(out, exchange.Announcement{
				Title:     article.Title,
				URL:       articleURL + article.Code,
				Published: time.UnixMilli(article.ReleaseDate),
			})
		}
	}

	return out, nil
}
