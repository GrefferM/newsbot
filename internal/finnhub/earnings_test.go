package finnhub

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

func TestOnlyTheTrackedCompaniesReportsComeWithTheirHour(t *testing.T) {
	failing := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing || r.URL.Query().Get("token") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		_, _ = w.Write([]byte(`{"earningsCalendar":[
{"symbol":"NVDA","date":"2026-11-17","hour":"amc","quarter":3},
{"symbol":"MRVL","date":"2026-11-26","hour":"","quarter":3},
{"symbol":"DRI","date":"2026-12-11","hour":"bmo","quarter":2}]}`))
	}))
	defer server.Close()

	reports := Earnings{http: httpclient.New(), url: server.URL, key: "secret"}
	now := time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)

	got, err := reports.Entries(context.Background(), map[string]bool{"NVDA": true, "MRVL": true}, now)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}

	if want := time.Date(2026, 11, 17, 21, 0, 0, 0, time.UTC); !got[0].Time.Equal(want) || got[0].Title != "NVDA: квартальний звіт, після закриття біржі" || got[0].URL != "https://www.nasdaq.com/market-activity/stocks/nvda/earnings" || got[0].Kind != schedule.Earnings || got[0].Assets[0] != "NVDA" {
		t.Errorf("nvidia %+v", got[0])
	}

	if got[1].Title != "MRVL: квартальний звіт" || got[1].Time.Hour() != 5 {
		t.Errorf("a report without its hour %+v", got[1])
	}

	failing = true

	if _, err := reports.Entries(context.Background(), map[string]bool{"NVDA": true}, now); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error %v", err)
	}
}
