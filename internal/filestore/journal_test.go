package filestore

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"newsbot/internal/agents"
)

func TestTheLastRecordsOfAnAgentAreReadBack(t *testing.T) {
	journal := NewJournal(filepath.Join(t.TempDir(), "agents.jsonl"), func(err error) {
		t.Error(err)
	})

	if got, err := journal.Last("binance", "", 10); err != nil || len(got) != 0 {
		t.Fatalf("an empty journal: %+v, %v", got, err)
	}

	for _, question := range []string{"a", "b", "c"} {
		journal.Keep(agents.Record{Agent: "binance", Question: question})
		journal.Keep(agents.Record{Agent: "risk", Question: question})
		journal.Keep(agents.Record{Agent: "binance", Topic: "BLUR", Question: question})
	}

	got, err := journal.Last("binance", "", 2)
	if err != nil || len(got) != 2 || got[0].Question != "b" || got[1].Question != "c" {
		t.Errorf("last two: %+v, %v", got, err)
	}

	if about, _ := journal.Last("binance", "BLUR", 10); len(about) != 3 {
		t.Errorf("about one coin: %+v", about)
	}

	journal.Keep(agents.Record{Agent: "binance", Cleared: true})
	journal.Keep(agents.Record{Agent: "binance", Question: "d"})

	if after, _ := journal.Last("binance", "", 10); len(after) != 1 || after[0].Question != "d" {
		t.Errorf("after clearing: %+v", after)
	}

	if about, _ := journal.Last("binance", "BLUR", 10); len(about) != 3 {
		t.Errorf("clearing one topic touched another: %+v", about)
	}
}

func TestALongJournalIsReadFromItsEndAcrossBlocks(t *testing.T) {
	journal := NewJournal(filepath.Join(t.TempDir(), "agents.jsonl"), func(err error) {
		t.Error(err)
	})
	answer := strings.Repeat("я", 30_000)

	for i := range 40 {
		journal.Keep(agents.Record{Agent: "trader", Topic: "ZEC", Question: fmt.Sprint(i), Answer: answer})
		journal.Keep(agents.Record{Agent: "osinter", Topic: "ZEC", Question: fmt.Sprint(i), Answer: answer})
	}

	got, err := journal.Last("trader", "ZEC", 10)
	if err != nil || len(got) != 10 || got[0].Question != "30" || got[9].Question != "39" || got[9].Answer != answer {
		t.Fatalf("last ten of a long journal: %d records, first %q, %v", len(got), got[0].Question, err)
	}

	if all, _ := journal.Last("osinter", "ZEC", 100); len(all) != 40 || all[0].Question != "0" {
		t.Errorf("the whole journal read back: %d records", len(all))
	}
}
