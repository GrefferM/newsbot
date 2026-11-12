package watchlist

import (
	"slices"
	"strings"
	"time"
)

type Pick struct {
	Asset string
	Note  string
	Added time.Time
}

type List struct {
	Picks []Pick
}

func Normalize(asset string) string {
	return strings.ToUpper(strings.TrimSpace(asset))
}

func (l List) Has(asset string) bool {
	return slices.ContainsFunc(l.Picks, func(p Pick) bool {
		return p.Asset == Normalize(asset)
	})
}

func (l List) Add(asset, note string, now time.Time) (List, bool) {
	asset = Normalize(asset)

	if asset == "" || l.Has(asset) {
		return l, false
	}

	picks := append(slices.Clone(l.Picks), Pick{Asset: asset, Note: strings.TrimSpace(note), Added: now})

	return List{Picks: picks}, true
}

func (l List) Remove(asset string) (List, bool) {
	if !l.Has(asset) {
		return l, false
	}

	picks := slices.DeleteFunc(slices.Clone(l.Picks), func(p Pick) bool {
		return p.Asset == Normalize(asset)
	})

	return List{Picks: picks}, true
}
