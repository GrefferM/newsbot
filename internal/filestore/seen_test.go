package filestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeenStorePersistsAcrossOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "state.json")

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if !store.MarkNew("a") || store.MarkNew("a") {
		t.Fatal("MarkNew must accept once")
	}

	store.MarkSeeded("binance-stream")

	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if reopened.MarkNew("a") || !reopened.IsSeeded("binance-stream") {
		t.Fatal("state lost after reopen")
	}
}

func TestOpenReadsLegacyStateFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	legacy := `{"seen":{"binance:abc":"2026-11-08T12:00:00Z"},"booted":{"rss":true}}`

	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if store.MarkNew("binance:abc") || !store.IsSeeded("rss") {
		t.Fatal("legacy state not understood")
	}
}
