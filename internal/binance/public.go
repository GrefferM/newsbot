package binance

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"newsbot/internal/httpclient"
)

var readHosts = map[string]string{
	"spot":    spotURL,
	"futures": fapiURL,
}

var readPrefixes = []string{"/api/v3/", "/fapi/v1/", "/fapi/v2/", "/futures/data/"}

var ErrMarket = errors.New("market має бути spot або futures")

type Public struct {
	http *httpclient.Client
}

func NewPublic(http *httpclient.Client) Public {
	return Public{http: http}
}

func (p Public) Get(ctx context.Context, market, address string) ([]byte, error) {
	host, ok := readHosts[market]
	if !ok {
		return nil, ErrMarket
	}

	path, inline, _ := strings.Cut(address, "?")

	if !allowed(path) {
		return nil, fmt.Errorf("шлях має починатися з %s", strings.Join(readPrefixes, ", "))
	}

	query, err := url.ParseQuery(inline)
	if err != nil {
		return nil, err
	}

	endpoint := host + path

	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	return p.http.Bytes(ctx, endpoint)
}

func allowed(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}

	for _, prefix := range readPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
