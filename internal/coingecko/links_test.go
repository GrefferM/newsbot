package coingecko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"newsbot/internal/httpclient"
)

func TestLinksPickTheLargestCoinWithTheTickerAndAreAskedOnce(t *testing.T) {
	asked := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++

		if strings.HasSuffix(r.URL.Path, "/search") {
			_, _ = w.Write([]byte(`{"coins":[
				{"id":"banana-for-scale-2","name":"Banana For Scale","symbol":"BANANAS31","market_cap_rank":381},
				{"id":"banana","name":"Banana","symbol":"BANANA","market_cap_rank":4475},
				{"id":"banana-gun","name":"Banana Gun","symbol":"BANANA","market_cap_rank":1118},
				{"id":"banana-old","name":"Banana Old","symbol":"banana","market_cap_rank":null}]}`))

			return
		}

		if r.URL.Path != "/coins/banana-gun" {
			t.Errorf("asked for %s", r.URL.Path)
		}

		_, _ = w.Write([]byte(`{"name":"Banana Gun","links":{"homepage":["http://bananagun.io",""],"whitepaper":"https://docs.bananagun.io/","twitter_screen_name":"bananagun","telegram_channel_identifier":"","chat_url":[],"announcement_url":[]}}`))
	}))
	defer server.Close()

	directory := NewDirectory(httpclient.New())
	directory.url = server.URL

	for range 2 {
		got, err := directory.Links(context.Background(), "BANANA")
		if err != nil || got.Project != "Banana Gun" || got.XURL != "https://x.com/bananagun" || len(got.Sites) != 1 {
			t.Fatalf("got %+v, %v", got, err)
		}

		if !slices.Equal(got.SameTicker, []string{"Banana", "Banana Old"}) {
			t.Errorf("other coins %v", got.SameTicker)
		}
	}

	if asked != 2 {
		t.Errorf("asked CoinGecko %d times for one coin", asked)
	}
}
