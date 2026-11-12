package research

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"newsbot/internal/schedule"
)

const (
	RiskMinimal = "мінімальний"
	RiskMedium  = "середній"
	RiskHigh    = "високий"
)

var Risks = []string{RiskMinimal, RiskMedium, RiskHigh}

type Finding struct {
	ID          string   `json:"id"`
	Asset       string   `json:"asset"`
	Published   string   `json:"published,omitempty"`
	Day         string   `json:"day,omitempty"`
	Hour        string   `json:"hour,omitempty"`
	Text        string   `json:"text"`
	Source      string   `json:"source"`
	Outlet      string   `json:"outlet,omitempty"`
	Others      []string `json:"others,omitempty"`
	Unconfirmed bool     `json:"unconfirmed"`
	Approximate bool     `json:"approximate,omitempty"`
}

const HourLayout = "15:04"

var hourInText = regexp.MustCompile(`\b\d{2}:\d{2} UTC`)

func Stamped(published string) bool {
	if _, err := time.Parse(time.DateOnly, published); err == nil {
		return true
	}

	_, err := time.Parse(time.RFC3339, published)

	return err == nil
}

func (f Finding) Dated() string {
	if f.Published != "" {
		return f.Published
	}

	return f.Day
}

func (f Finding) Starts(zone *time.Location) (time.Time, bool) {
	hour := f.Hour

	if hour == "" {
		hour = strings.TrimSuffix(hourInText.FindString(f.Text), " UTC")
	}

	if at, err := time.Parse(time.DateOnly+" "+HourLayout, f.Day+" "+hour); err == nil {
		return at, true
	}

	day, _ := time.ParseInLocation(time.DateOnly, f.Day, zone)

	return day, false
}

func (p Archive) Dates(zone *time.Location) []schedule.Entry {
	var out []schedule.Entry

	for _, f := range p.Items {
		at, timed := f.Starts(zone)
		if f.Day == "" || at.IsZero() {
			continue
		}

		out = append(out, schedule.Entry{
			ID:     f.ID + ":" + f.Day,
			Time:   at.UTC(),
			Title:  f.Text,
			Source: f.Asset,
			URL:    f.Source,
			Kind:   schedule.Coin,
			Assets: []string{f.Asset},
			AllDay: !timed,
			Rough:  f.Approximate,
			Outlet: f.Outlet,
		})
	}

	return out
}

func (f Finding) Sources() []string {
	return append([]string{f.Source}, f.Others...)
}

func (f Finding) retells(other Finding) bool {
	if f.Asset != other.Asset || f.Day != other.Day {
		return false
	}

	return slices.ContainsFunc(f.Sources(), func(source string) bool {
		return source != "" && slices.Contains(other.Sources(), source)
	})
}

func Report(found []Finding, f Finding) ([]Finding, bool) {
	for i, known := range found {
		if !known.retells(f) {
			continue
		}

		merged := slices.Clone(found)

		for _, source := range f.Sources() {
			if source != "" && !slices.Contains(merged[i].Sources(), source) {
				merged[i].Others = append(slices.Clone(merged[i].Others), source)
			}
		}

		return merged, false
	}

	return append(slices.Clone(found), f), true
}

const (
	DirectionShort = "шорт"
	DirectionLong  = "лонг"
	DirectionNone  = "не входити"
)

var Directions = []string{DirectionShort, DirectionLong, DirectionNone}

type Scenario struct {
	Condition string  `json:"condition"`
	Action    string  `json:"action"`
	Risk      string  `json:"risk"`
	Why       string  `json:"why"`
	Direction string  `json:"direction,omitempty"`
	WakeAbove float64 `json:"wake_price_above,omitempty"`
	WakeBelow float64 `json:"wake_price_below,omitempty"`
	SizePct   float64 `json:"size_pct,omitempty"`
	Stop      float64 `json:"stop_price,omitempty"`
	Take      float64 `json:"take_price,omitempty"`
	WakeAfter string  `json:"wake_after,omitempty"`
}

func (s Scenario) Terms() string {
	parts := []string{s.Direction}

	switch {
	case s.WakeAbove > 0:
		parts = append(parts, "від "+number(s.WakeAbove))
	case s.WakeBelow > 0:
		parts = append(parts, "до "+number(s.WakeBelow))
	}

	if s.WakeAfter != "" {
		parts = append(parts, "не раніше "+s.WakeAfter)
	}

	return strings.Join(parts, " ") + fmt.Sprintf(
		", %s%% рахунку, стоп %s, тейк %s",
		number(s.SizePct),
		number(s.Stop),
		number(s.Take),
	)
}

func (s Scenario) Retimed(numbers Scenario) Scenario {
	s.WakeAbove, s.WakeBelow, s.WakeAfter = numbers.WakeAbove, numbers.WakeBelow, numbers.WakeAfter
	s.SizePct, s.Stop, s.Take = numbers.SizePct, numbers.Stop, numbers.Take
	s.Action = s.Terms()

	return s
}

func number(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (s Scenario) Long() bool {
	return s.Direction == DirectionLong
}

func (s Scenario) Wake() float64 {
	return max(s.WakeAbove, s.WakeBelow)
}

func (s Scenario) Measured() bool {
	return s.Direction != "" || s.Wake() > 0 || s.SizePct > 0 || s.Stop > 0 || s.Take > 0
}

func (s Scenario) NotBefore() time.Time {
	return Moment(s.WakeAfter)
}

func Moment(stamp string) time.Time {
	if at, err := time.Parse(time.RFC3339, stamp); err == nil {
		return at
	}

	at, _ := time.Parse(time.DateOnly, stamp)

	return at
}

func (s Scenario) Armable() error {
	wake := s.Wake()

	switch {
	case s.Direction != DirectionShort && s.Direction != DirectionLong:
		return errors.New("для стеження потрібен direction: шорт або лонг")
	case s.WakeAbove > 0 && s.WakeBelow > 0:
		return errors.New("ціна-поріг може бути лише одна: wake_price_above або wake_price_below")
	case s.WakeAfter != "" && !Stamped(s.WakeAfter):
		return errors.New("wake_after має бути РРРР-ММ-ДД або РРРР-ММ-ДДTГГ:ХХ:ССZ")
	case wake <= 0:
		return errors.New("потрібен тригер: ціна-поріг wake_price_above або wake_price_below; дата wake_after — лише разом із ним")
	case s.SizePct <= 0 || s.SizePct > 100:
		return errors.New("size_pct має бути від 0 до 100")
	case s.Long() && !(s.Stop > 0 && s.Stop < s.Take):
		return errors.New("для лонга стоп має бути нижче тейку")
	case !s.Long() && !(s.Take > 0 && s.Take < s.Stop):
		return errors.New("для шорта тейк має бути нижче стопу")
	case wake > 0 && (wake <= min(s.Stop, s.Take) || wake >= max(s.Stop, s.Take)):
		return errors.New("ціна-поріг має лежати між стопом і тейком")
	default:
		return nil
	}
}

type Plan struct {
	Wish      string     `json:"wish,omitempty"`
	Summary   string     `json:"summary"`
	Scenarios []Scenario `json:"scenarios"`
	Made      time.Time  `json:"made"`
}

type Archive struct {
	Items []Finding
	Plans map[string]Plan
}

func (p Archive) Of(asset string) []Finding {
	out := make([]Finding, 0)

	for _, f := range p.Items {
		if f.Asset == asset {
			out = append(out, f)
		}
	}

	slices.SortStableFunc(out, func(a, b Finding) int {
		return cmp.Compare(b.Dated(), a.Dated())
	})

	return out
}

func (p Archive) Keep(f Finding) (Archive, bool) {
	if f.ID == "" || f.Asset == "" || f.Text == "" {
		return p, false
	}

	items, _ := Report(p.Items, f)

	return Archive{Items: items, Plans: p.Plans}, true
}

func (p Archive) Remove(id string) (Archive, bool) {
	items := slices.DeleteFunc(slices.Clone(p.Items), func(f Finding) bool {
		return f.ID == id
	})

	return Archive{Items: items, Plans: p.Plans}, len(items) != len(p.Items)
}

func (p Archive) Plan(asset string) (Plan, bool) {
	plan, kept := p.Plans[asset]

	return plan, kept
}

func (p Archive) Fix(asset string, plan Plan) (Archive, bool) {
	if asset == "" || (len(plan.Scenarios) == 0 && plan.Summary == "") {
		return p, false
	}

	plans := maps.Clone(p.Plans)

	if plans == nil {
		plans = map[string]Plan{}
	}

	plans[asset] = plan

	return Archive{Items: p.Items, Plans: plans}, true
}
