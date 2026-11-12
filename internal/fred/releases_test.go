package fred

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"newsbot/internal/httpclient"
	"newsbot/internal/schedule"
)

func TestReleaseDatesGetTheirHourAndTheKeyStaysOutOfErrors(t *testing.T) {
	failing := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing || r.URL.Query().Get("api_key") != "secret" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		_, _ = w.Write([]byte(`{"release_dates":[{"release_id":10,"date":"2026-12-10"}]}`))
	}))
	defer server.Close()

	releases := Releases{http: httpclient.New(), url: server.URL, key: "secret"}

	got, err := releases.Entries(context.Background())
	if err != nil || len(got) != len(watched) {
		t.Fatalf("got %+v, %v", got, err)
	}

	if want := time.Date(2026, 12, 10, 13, 30, 0, 0, time.UTC); !got[0].Time.Equal(want) || got[0].Source != sourceName || got[0].Kind != schedule.Inflation || got[2].Kind != "" {
		t.Errorf("entry %+v", got[0])
	}

	failing = true

	if _, err := releases.Entries(context.Background()); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error %v", err)
	}
}
