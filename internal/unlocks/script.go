package unlocks

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"newsbot/internal/news"
	"newsbot/internal/schedule"
)

const (
	tableURL      = "https://coinmarketcap.com/token-unlocks/"
	scriptName    = "unlock-table"
	scriptTimeout = 2 * time.Minute
	scriptAhead   = "90"
	priceGap      = 0.2
)

var scriptColumns = []string{"symbol", "price_usd", "next_unlock_time", "next_unlock_usd", "market_cap_usd"}

var priceScales = []float64{1, 1e3, 1e6}

type ScriptDeps struct {
	Python string
	Path   string
	Listed Listed
	Every  time.Duration
	Floor  float64
	Memory Memory
}

type Script struct {
	deps ScriptDeps
	mu   sync.Mutex
	read time.Time
	kept []schedule.Entry
}

func NewScript(deps ScriptDeps) *Script {
	s := &Script{deps: deps}
	s.read, s.kept = recall(deps.Memory, scriptName)

	return s
}

func (s *Script) Name() string {
	return scriptName
}

func (s *Script) Entries(ctx context.Context) ([]schedule.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.read.IsZero() && time.Since(s.read) < s.deps.Every {
		return slices.Clone(s.kept), nil
	}

	ctx, cancel := context.WithTimeout(ctx, scriptTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, s.deps.Python, s.deps.Path, "--csv", "--limit", "0", "--days", scriptAhead).Output()
	if err != nil {
		return nil, fmt.Errorf("run the unlock table script: %w", err)
	}

	found, err := Table(out)
	if err != nil {
		return nil, err
	}

	s.read, s.kept = time.Now(), s.weighty(ctx, found)

	_ = s.deps.Memory.Remember(scriptName, s.read, s.kept)

	return slices.Clone(s.kept), nil
}

func (s *Script) weighty(ctx context.Context, found []Unlock) []schedule.Entry {
	entries := []schedule.Entry{}
	known := map[string]bool{}

	for _, unlock := range found {
		entry := unlock.Entry(tableURL)

		if known[unlock.Asset] || entry.Share < s.deps.Floor || !s.traded(ctx, unlock) {
			continue
		}

		known[unlock.Asset] = true
		entries = append(entries, entry)
	}

	return entries
}

func (s *Script) traded(ctx context.Context, unlock Unlock) bool {
	ticker, found, err := s.deps.Listed.Ticker(ctx, news.Symbol(unlock.Asset))
	if err != nil || !found || unlock.Price <= 0 {
		return false
	}

	for _, scale := range priceScales {
		if math.Abs(ticker.Price/(unlock.Price*scale)-1) <= priceGap {
			return true
		}
	}

	return false
}

func Table(data []byte) ([]Unlock, error) {
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read the unlock table: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("the unlock table is empty")
	}

	at := map[string]int{}

	for _, name := range scriptColumns {
		at[name] = slices.Index(rows[0], name)

		if at[name] < 0 {
			return nil, fmt.Errorf("the unlock table has no column %s", name)
		}
	}

	var out []Unlock

	for _, row := range rows[1:] {
		when, err := time.Parse(time.RFC3339, row[at["next_unlock_time"]])
		if err != nil {
			continue
		}

		price, _ := strconv.ParseFloat(row[at["price_usd"]], 64)
		value, _ := strconv.ParseFloat(row[at["next_unlock_usd"]], 64)
		worth, _ := strconv.ParseFloat(row[at["market_cap_usd"]], 64)

		if value <= 0 {
			continue
		}

		out = append(out, Unlock{
			Asset: strings.TrimPrefix(row[at["symbol"]], "$"),
			Price: price,
			Value: strings.TrimPrefix(news.FormatUSD(value), "$"),
			Share: share(value, worth),
			At:    when.UTC(),
		})
	}

	return out, nil
}

func share(value, worth float64) string {
	if worth <= 0 {
		return ""
	}

	return strconv.FormatFloat(value/worth*100, 'f', 2, 64)
}
