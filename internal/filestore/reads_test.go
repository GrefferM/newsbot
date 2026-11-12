package filestore

import (
	"path/filepath"
	"testing"
	"time"
)

type kept struct {
	Asset string `json:"asset"`
}

func TestWhatASourceReadIsRecalledAfterARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reads.json")
	read := time.Date(2026, 11, 13, 7, 42, 0, 0, time.UTC)

	if err := NewReadsFile(path).Remember("unlocks", read, []kept{{Asset: "KAITO"}}); err != nil {
		t.Fatal(err)
	}

	if err := NewReadsFile(path).Remember("exchange-listings", read.Add(time.Hour), []kept{}); err != nil {
		t.Fatal(err)
	}

	var got []kept

	at, err := NewReadsFile(path).Recall("unlocks", &got)
	if err != nil || !at.Equal(read) || len(got) != 1 || got[0].Asset != "KAITO" {
		t.Fatalf("recalled %v %+v, err %v", at, got, err)
	}

	var none []kept

	if at, err := NewReadsFile(path).Recall("unlock-table", &none); err != nil || !at.IsZero() || none != nil {
		t.Errorf("a source never read recalls nothing: %v %+v %v", at, none, err)
	}
}
