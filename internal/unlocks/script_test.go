package unlocks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

const csvTable = "symbol,name,rank,price_usd,market_cap_usd,unlocked_percent,next_unlock_time,next_unlock_tokens,next_unlock_usd,next_unlock_percent\n" +
	"KAITO,KAITO,344,0.2082,50260000,49.95,2026-11-20T00:00:00+00:00,18100000,3770000,1.81\n" +
	"$MVRK,Mavryk Network,4644,0.005604,,46.52,2026-11-18T13:00:00+00:00,30760000,172386.8,3.01\n" +
	"IDLE,Idle,,,,10,,,,\n" +
	"FNTR,Fundtir,,,,67.72,2026-11-21T00:00:00+00:00,10940000,0,1.36\n"

func TestTheScriptsTableGivesEachUnlockItsHourAndItsShareOfTheCirculatingSupply(t *testing.T) {
	found, err := Table([]byte(csvTable))
	if err != nil || len(found) != 2 {
		t.Fatalf("unlocks %+v, err %v", found, err)
	}

	entry := found[0].Entry(tableURL)

	if entry.ID != "unlock:KAITO:2026-11-20T00:00:00Z" || entry.AllDay || entry.Share != 0.075 {
		t.Errorf("entry %+v", entry)
	}

	if entry.Title != "Розблокування 7.50% обігу — $3.8M" {
		t.Errorf("the share is of the circulating supply, not of the total: %q", entry.Title)
	}

	if other := found[1].Entry(tableURL); other.Source != "MVRK" || other.Title != "Розблокування $172K" {
		t.Errorf("a coin without a market value has no share: %+v", other)
	}

	if found[1].At.Format(time.RFC3339) != "2026-11-18T13:00:00Z" {
		t.Errorf("moment %v", found[1].At)
	}

	if _, err := Table([]byte("symbol,next_unlock_date\nKAITO,2026-11-20\n")); err == nil {
		t.Error("a table without the expected columns is refused")
	}
}

type priced map[string]float64

func (p priced) Ticker(_ context.Context, s news.Symbol) (market.Ticker, bool, error) {
	price, found := p[string(s)]

	return market.Ticker{Price: price}, found, nil
}

func TestTheTableKeepsAWeightyUnlockOfTheVeryCoinThePerpetualTrades(t *testing.T) {
	at := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	found := []Unlock{
		{Asset: "KAITO", Price: 0.2082, Value: "3.77M", Share: "7.50", At: at},
		{Asset: "ARB", Price: 0.21, Value: "70.2K", Share: "0.01", At: at},
		{Asset: "KO", Price: 0.00068, Value: "5.1K", Share: "3.33", At: at},
		{Asset: "BONK", Price: 0.00001, Value: "2.00M", Share: "2.50", At: at},
		{Asset: "GONE", Price: 1, Value: "2.00M", Share: "9.00", At: at},
		{Asset: "KAITO", Price: 0.2082, Value: "1.00M", Share: "2.00", At: at.AddDate(0, 1, 0)},
	}

	script := NewScript(ScriptDeps{Listed: priced{"KAITO": 0.21, "ARB": 0.21, "KO": 71.4, "BONK": 0.0102}, Floor: 0.01, Memory: memory{}})
	kept := script.weighty(context.Background(), found)

	if len(kept) != 2 || kept[0].Source != "KAITO" || kept[1].Source != "BONK" || script.Name() != "unlock-table" {
		t.Errorf("a small unlock, another coin under the same ticker and a coin without a perpetual are left out: %+v", kept)
	}
}

type remembered struct {
	read time.Time
	kept []byte
}

type memory map[string]remembered

func (m memory) Recall(name string, kept any) (time.Time, error) {
	last, found := m[name]
	if !found {
		return time.Time{}, nil
	}

	return last.read, json.Unmarshal(last.kept, kept)
}

func (m memory) Remember(name string, read time.Time, kept any) error {
	data, err := json.Marshal(kept)
	if err != nil {
		return err
	}

	m[name] = remembered{read: read, kept: data}

	return nil
}
