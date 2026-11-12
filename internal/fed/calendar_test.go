package fed

import (
	"testing"
	"time"

	"newsbot/internal/schedule"
)

func TestOnlyCommitteeEventsAreKeptWithTheirTime(t *testing.T) {
	body := []byte("\xef\xbb\xbf" + `{"events":[
{"title":"FOMC Meeting","time":"2:00 p.m.","month":"2026-12","days":"8-9","type":"FOMC"},
{"title":"FOMC Minutes","time":"","month":"2026-11","days":"18","type":"FOMC"},
{"title":"Speech - Governor","time":"7:00 p.m.","month":"2026-11","days":"13","type":"Speeches"}]}`)

	got, err := parse(body)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}

	if want := time.Date(2026, 12, 9, 19, 0, 0, 0, time.UTC); !got[0].Time.Equal(want) || got[0].Title != "FOMC Meeting" || got[0].Kind != schedule.Rates {
		t.Errorf("meeting %+v", got[0])
	}

	if want := time.Date(2026, 11, 18, 5, 0, 0, 0, time.UTC); !got[1].Time.Equal(want) || got[1].Kind != "" {
		t.Errorf("minutes %+v", got[1])
	}
}
