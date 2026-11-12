package filestore

import (
	"path/filepath"
	"testing"
	"time"

	"newsbot/internal/monitor"
	"newsbot/internal/news"
)

func TestTheEventsSurviveARestart(t *testing.T) {
	file := NewJSONFile[[]news.Event](filepath.Join(t.TempDir(), "events.json"))

	if saved, err := file.Load(nil); err != nil || len(saved) != 0 {
		t.Fatalf("no file yet: %v, %v", saved, err)
	}

	before := monitor.NewRecent(2, file, nil)

	for _, id := range []string{"a", "b", "c"} {
		if err := before.Add(news.Event{ID: id, Category: news.Trade, Time: time.Now().UTC(), Symbols: []news.Symbol{"ZEC"}}); err != nil {
			t.Fatal(err)
		}
	}

	saved, err := file.Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	after := monitor.NewRecent(2, file, saved).Latest(10)

	if len(after) != 2 || after[0].ID != "c" || after[1].ID != "b" || after[0].Symbols[0] != "ZEC" {
		t.Errorf("after a restart: %+v", after)
	}
}
