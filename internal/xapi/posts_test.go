package xapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecentPostsComeWithTheirAddress(t *testing.T) {
	var asked *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r

		_, _ = w.Write([]byte(`{"data":[{"id":"42","text":"Mainnet is live","created_at":"2026-11-09T10:00:00.000Z","public_metrics":{"like_count":7,"retweet_count":3}}],"meta":{"result_count":1}}`))
	}))
	defer server.Close()

	client := Client{http: server.Client(), url: server.URL, token: "k"}

	got, err := client.Recent(context.Background(), "from:project", "", time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}

	if got[0].URL != "https://x.com/i/status/42" || got[0].Likes != 7 || got[0].Reposts != 3 {
		t.Errorf("post %+v", got[0])
	}

	if asked.URL.Query().Get("start_time") != "2026-11-10T08:00:00Z" {
		t.Errorf("the window was not asked: %q", asked.URL.Query().Get("start_time"))
	}

	if asked.Header.Get("Authorization") != "Bearer k" || asked.URL.Query().Get("query") != "from:project" {
		t.Errorf("asked %q with %q", asked.URL.Query().Get("query"), asked.Header.Get("Authorization"))
	}
}
