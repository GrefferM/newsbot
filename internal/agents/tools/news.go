package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"

	"newsbot/internal/agents"
	"newsbot/internal/research"
	"newsbot/internal/watchlist"
)

type Keeper interface {
	Keep(f research.Finding) error
}

const cyrillicShare = 4

type NewsTool struct {
	Keeper Keeper
}

func (NewsTool) Spec() agents.ToolSpec {
	return agents.ToolSpec{
		Name: "report_news",
		Description: "Передає власникові одну новину про монету: окрему подію з нею чи її проєктом, з датою " +
			"публікації і джерелом. Загальноринкова новина і переказ руху ціни сюди не йдуть. " +
			"Вона зберігається за монетою, і власник бачить її карткою. Один факт — одна новина: якщо про " +
			"нього пишуть кілька джерел, решту адрес перелічи в other_sources.",
		Parameters: object(map[string]any{
			"symbol":    text("тікер монети"),
			"published": text("коли джерело це опублікувало: РРРР-ММ-ДДTГГ:ХХ:ССZ; якщо джерело часу не дає — лише РРРР-ММ-ДД, часу не вигадуй"),
			"day":       text("лише для події, що попереду: її дата, РРРР-ММ-ДД"),
			"hour":      text("година цієї події за UTC, ГГ:ХХ — лише якщо джерело її називає; тоді й day — дата за UTC. Години не вигадуй"),
			"text":      text("що сталося або станеться, одним реченням, українською"),
			"source":    text("адреса головного джерела"),
			"outlet":    text("назва видання, якщо адреса веде через стрічку новин, а не на саме видання"),
			"other_sources": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "адреси інших джерел, що пишуть про той самий факт",
			},
			"unconfirmed": map[string]any{"type": "boolean", "description": "true, якщо першоджерела немає"},
			"approximate": map[string]any{"type": "boolean", "description": "true, якщо джерело не називає точної дати події і day — оцінка"},
		}, "symbol", "published", "text", "source"),
	}
}

func (t NewsTool) Call(_ context.Context, raw json.RawMessage, turn *agents.Turn) (string, error) {
	var args struct {
		Symbol      string   `json:"symbol"`
		Published   string   `json:"published"`
		Day         string   `json:"day"`
		Hour        string   `json:"hour"`
		Text        string   `json:"text"`
		Source      string   `json:"source"`
		Outlet      string   `json:"outlet"`
		Others      []string `json:"other_sources"`
		Unconfirmed bool     `json:"unconfirmed"`
		Approximate bool     `json:"approximate"`
	}

	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}

	if !research.Stamped(args.Published) {
		return "", errors.New("published має бути РРРР-ММ-ДД або РРРР-ММ-ДДTГГ:ХХ:ССZ")
	}

	if _, err := time.Parse(time.DateOnly, args.Day); args.Day != "" && err != nil {
		return "", errors.New("day має бути РРРР-ММ-ДД або порожнім")
	}

	if _, err := time.Parse(research.HourLayout, args.Hour); args.Hour != "" && (err != nil || args.Day == "") {
		return "", errors.New("hour має бути ГГ:ХХ за UTC разом із day або порожнім")
	}

	if strings.TrimSpace(args.Text) == "" {
		return "", errors.New("порожній text")
	}

	if !ukrainian(args.Text) {
		return "", errors.New("text має бути українською")
	}

	finding := research.Finding{
		ID:          newID(),
		Asset:       watchlist.Normalize(args.Symbol),
		Published:   args.Published,
		Day:         args.Day,
		Hour:        args.Hour,
		Text:        strings.TrimSpace(args.Text),
		Source:      args.Source,
		Outlet:      strings.TrimSpace(args.Outlet),
		Others:      args.Others,
		Unconfirmed: args.Unconfirmed,
		Approximate: args.Approximate,
	}

	if err := t.Keeper.Keep(finding); err != nil {
		return "", err
	}

	found, added := research.Report(turn.Findings, finding)

	turn.Findings = found

	if !added {
		return "цей факт уже передано: нові адреси додано до тієї новини", nil
	}

	return "передано", nil
}

func ukrainian(text string) bool {
	letters, cyrillic := 0, 0

	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}

		letters++

		if unicode.Is(unicode.Cyrillic, r) {
			cyrillic++
		}
	}

	return cyrillic*cyrillicShare >= letters
}
