package agents

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"newsbot/internal/research"
)

type scriptedModel struct {
	replies []ChatMessage
	seen    [][]ChatMessage
}

func (m *scriptedModel) Complete(_ context.Context, _ string, messages []ChatMessage, _ []ToolSpec) (ChatMessage, error) {
	m.seen = append(m.seen, messages)

	if len(m.replies) == 0 {
		return ChatMessage{}, errors.New("script ended")
	}

	reply := m.replies[0]
	m.replies = m.replies[1:]

	return reply, nil
}

func call(name, args string) ChatMessage {
	return ChatMessage{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Name: name, Arguments: args}}}
}

type factTool struct{}

func (factTool) Spec() ToolSpec {
	return ToolSpec{Name: "report_fact"}
}

func (factTool) Call(_ context.Context, raw json.RawMessage, turn *Turn) (string, error) {
	var f research.Finding

	if err := json.Unmarshal(raw, &f); err != nil {
		return "", err
	}

	if f.Text == "" {
		return "", errors.New("порожній факт")
	}

	turn.Findings = append(turn.Findings, f)

	return "передано", nil
}

type lookTool struct{}

func (lookTool) Spec() ToolSpec {
	return ToolSpec{Name: "look"}
}

func (lookTool) Call(context.Context, json.RawMessage, *Turn) (string, error) {
	return "нічого", nil
}

type keptRecords struct {
	mu      sync.Mutex
	records []Record
}

func (k *keptRecords) Keep(r Record) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.records = append(k.records, r)
}

func (k *keptRecords) Last(agent, topic string, _ int) ([]Record, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	var out []Record

	for _, r := range k.records {
		if r.Topic != topic || r.Agent != agent {
			continue
		}

		if r.Cleared {
			out = nil

			continue
		}

		out = append(out, r)
	}

	return out, nil
}

func team(model ChatModel, agent Agent) *Team {
	return NewTeam(Deps{Model: model, Agents: []Agent{agent}})
}

func TestAnAgentUsesItsToolsAndAnswersWithoutMemory(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{
		call("report_fact", `{"asset":"BLUR","text":"Закриття сайту"}`),
		{Role: "assistant", Content: "BLUR закриває сайт."},
		{Role: "assistant", Content: "Друга відповідь."},
	}}
	agents := team(model, Agent{Name: "osinter", Prompt: "Роль.", Tools: []Tool{factTool{}}})

	answer, err := agents.Ask(context.Background(), "osinter", "", "Що з BLUR?")
	if err != nil || answer.Text != "BLUR закриває сайт." || len(answer.Findings) != 1 {
		t.Fatalf("answer %+v, %v", answer, err)
	}

	if _, err := agents.Ask(context.Background(), "osinter", "", "Інше питання"); err != nil {
		t.Fatal(err)
	}

	if len(model.seen[2]) != 2 || model.seen[2][1].Content != "Інше питання" {
		t.Errorf("a new question carried history: %+v", model.seen[2])
	}

	if _, err := agents.Ask(context.Background(), "nobody", "", "?"); !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("unknown agent: %v", err)
	}
}

func TestAToolErrorGoesBackToTheModel(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{
		call("report_fact", `{"asset":"X"}`),
		{Role: "assistant", Content: "Не знайшов."},
	}}

	if _, err := team(model, Agent{Name: "osinter", Tools: []Tool{factTool{}}}).Ask(context.Background(), "osinter", "", "?"); err != nil {
		t.Fatal(err)
	}

	toolReply := model.seen[1][len(model.seen[1])-1]

	if toolReply.Role != "tool" || !strings.HasPrefix(toolReply.Content, "помилка") {
		t.Errorf("tool reply %+v", toolReply)
	}
}

func TestAnAgentThatOnlyGathersIsMadeToReport(t *testing.T) {
	replies := make([]ChatMessage, 0, agentSteps+2)

	for range agentSteps {
		replies = append(replies, call("look", `{}`))
	}

	replies = append(replies,
		call("report_fact", `{"asset":"BLUR","text":"Факт"}`),
		ChatMessage{Role: "assistant", Content: "Підсумок."},
	)

	model := &scriptedModel{replies: replies}
	agent := Agent{Name: "osinter", Tools: []Tool{lookTool{}}, Reports: []Tool{factTool{}}}

	answer, err := team(model, agent).Ask(context.Background(), "osinter", "", "?")
	if err != nil || answer.Text != "Підсумок." || len(answer.Findings) != 1 {
		t.Fatalf("answer %+v, %v", answer, err)
	}

	asked := model.seen[agentSteps]

	if asked[len(asked)-1].Content != wrapUp {
		t.Errorf("no wrap-up request: %+v", asked[len(asked)-1])
	}
}

func TestEveryQuestionIsKeptWithItsStepsAndAnswer(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{
		call("report_fact", `{"asset":"BLUR","text":"Закриття сайту"}`),
		{Role: "assistant", Content: "Готово."},
	}}
	kept := &keptRecords{}
	agents := NewTeam(Deps{Model: model, Agents: []Agent{{Name: "osinter", Tools: []Tool{factTool{}}}}, Log: kept})

	if _, err := agents.Ask(context.Background(), "osinter", "", "Що з BLUR?"); err != nil {
		t.Fatal(err)
	}

	if len(kept.records) != 1 {
		t.Fatalf("kept %d records", len(kept.records))
	}

	r := kept.records[0]

	if r.Agent != "osinter" || r.Question != "Що з BLUR?" || r.Answer != "Готово." || len(r.Findings) != 1 {
		t.Errorf("record %+v", r)
	}

	if len(r.Steps) != 1 || r.Steps[0].Tool != "report_fact" || r.Steps[0].Result != "передано" {
		t.Errorf("steps %+v", r.Steps)
	}

	if history, err := agents.History("osinter", "", 10); err != nil || len(history) != 1 {
		t.Errorf("history %+v, %v", history, err)
	}

	if _, err := agents.History("nobody", "", 10); !errors.Is(err, ErrUnknownAgent) {
		t.Errorf("history of an unknown agent: %v", err)
	}
}

func TestAnEmptyReplyIsAskedAgainForAnAnswerInWords(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{
		{Role: "assistant", Content: ""},
		{Role: "assistant", Content: " "},
		{Role: "assistant", Content: "Відповідь."},
	}}

	answer, err := team(model, Agent{Name: "binance"}).Ask(context.Background(), "binance", "", "?")
	if err != nil || answer.Text != "Відповідь." {
		t.Fatalf("answer %+v, %v", answer, err)
	}

	last := model.seen[2]

	if last[len(last)-1].Content != closing {
		t.Errorf("the last request did not ask for words: %+v", last[len(last)-1])
	}
}

func TestAnAgentWithMemoryIsShownItsLastExchangesOnTheSameTopic(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{
		{Role: "assistant", Content: "Перша."},
		{Role: "assistant", Content: "Інша монета."},
		{Role: "assistant", Content: "Друга."},
	}}
	kept := &keptRecords{}
	agents := NewTeam(Deps{Model: model, Agents: []Agent{{Name: "osinter", Memory: 5}}, Log: kept})

	for _, ask := range [][2]string{{"BLUR", "Що нового?"}, {"MOVE", "А тут?"}, {"BLUR", "А тепер?"}} {
		if _, err := agents.Ask(context.Background(), "osinter", ask[0], ask[1]); err != nil {
			t.Fatal(err)
		}
	}

	last := model.seen[2]

	if len(last) != 4 || last[1].Content != "Інструмент BLUR. Що нового?" || last[2].Content != "Перша." || last[3].Content != "Інструмент BLUR. А тепер?" {
		t.Errorf("the third question saw %+v", last)
	}

	if kept.records[0].Question != "Що нового?" {
		t.Errorf("the journal keeps the framed question: %q", kept.records[0].Question)
	}

	if err := agents.Forget("osinter", "BLUR"); err != nil {
		t.Fatal(err)
	}

	model.replies = []ChatMessage{{Role: "assistant", Content: "З чистого аркуша."}}

	if _, err := agents.Ask(context.Background(), "osinter", "BLUR", "Знову?"); err != nil {
		t.Fatal(err)
	}

	if fresh := model.seen[3]; len(fresh) != 2 {
		t.Errorf("after forgetting the agent still saw %+v", fresh)
	}
}

type pickTool struct {
	assets []string
}

func (pickTool) Spec() ToolSpec {
	return ToolSpec{Name: "pick"}
}

func (p pickTool) Call(_ context.Context, _ json.RawMessage, turn *Turn) (string, error) {
	for _, asset := range p.assets {
		turn.Picks = append(turn.Picks, Pick{Asset: asset, Direction: "шорт", Why: "слабка"})
	}

	return "прийнято", nil
}

type roleModel struct {
	mu    sync.Mutex
	asked map[string][]string
}

func (m *roleModel) Complete(_ context.Context, _ string, messages []ChatMessage, tools []ToolSpec) (ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	role := messages[0].Content
	last := messages[len(messages)-1]

	m.asked[role] = append(m.asked[role], last.Content)

	if strings.HasPrefix(role, "Відбір") && last.Role == "user" {
		return call("pick", `{}`), nil
	}

	switch {
	case strings.HasPrefix(role, "Відбір"):
		return ChatMessage{Role: "assistant", Content: "Ринок падає, фіналісти названі."}, nil
	case strings.HasPrefix(role, "Розвідка"):
		return ChatMessage{Role: "assistant", Content: "Знайдене передав."}, nil
	case strings.HasPrefix(role, "Огляд"):
		return ChatMessage{Role: "assistant", Content: "Ринок оглянув."}, nil
	case strings.HasPrefix(role, "Копання"):
		return ChatMessage{Role: "assistant", Content: "Досьє: " + strings.Fields(last.Content)[1]}, nil
	default:
		return ChatMessage{Role: "assistant", Content: "Підсумок."}, nil
	}
}

func TestTheFunnelSurveysSelectsDigsEachFinalistApartAndRanks(t *testing.T) {
	model := &roleModel{asked: map[string][]string{}}
	kept := &keptRecords{}
	agents := NewTeam(Deps{
		Model:  model,
		Log:    kept,
		Agents: []Agent{{Name: "binance"}},
		Funnel: Funnel{
			Survey:   Agent{Name: "osinter", Prompt: "Огляд"},
			Select:   Agent{Name: "binance", Prompt: "Відбір", Tools: []Tool{pickTool{assets: []string{"WLD", "ZRO"}}}},
			Research: Agent{Name: "osinter", Prompt: "Розвідка"},
			Dig:      Agent{Name: "binance", Prompt: "Копання"},
			Rank:     Agent{Name: "binance", Prompt: "Зведення"},
		},
	})

	answer, err := agents.Scout(context.Background(), "Кандидати на шорт?")
	if err != nil || answer.Text != "Підсумок." {
		t.Fatalf("answer %+v, %v", answer, err)
	}

	var ranked, researched, surveyed, selected string

	for role, asked := range model.asked {
		if strings.HasPrefix(role, "Зведення") {
			ranked = asked[0]
		}

		if strings.HasPrefix(role, "Огляд") {
			surveyed = asked[0]
		}

		if strings.HasPrefix(role, "Відбір") {
			selected = strings.Join(asked, "\n")
		}

		if strings.HasPrefix(role, "Розвідка") {
			researched = strings.Join(asked, "\n")
		}
	}

	for _, want := range []string{"Кандидати на шорт?", "Ринок падає", "Досьє WLD", "Досьє ZRO"} {
		if !strings.Contains(ranked, want) {
			t.Errorf("the ranking was not shown %q: %s", want, ranked)
		}
	}

	topics := map[string]int{}

	for _, r := range kept.records {
		topics[r.Agent+" "+r.Topic]++
	}

	if topics["binance WLD"] != 1 || topics["binance ZRO"] != 1 || topics["binance "] != 1 {
		t.Errorf("records by topic %v", topics)
	}

	if topics["osinter WLD"] != 1 || topics["osinter ZRO"] != 1 {
		t.Errorf("the researcher is asked about each finalist before its dossier: %v", topics)
	}

	if topics["osinter "] != 1 || !strings.Contains(surveyed, "Кандидати на шорт?") || kept.records[0].Agent != "osinter" {
		t.Errorf("the researcher surveys the world before the selection, with the owner's question: %q, %v", surveyed, topics)
	}

	if strings.Contains(selected, "Ринок оглянув") || strings.Contains(ranked, "Ринок оглянув") {
		t.Errorf("the survey's answer goes to no other agent: %q", selected)
	}

	if !strings.Contains(researched, "Інструмент WLD.") || !strings.Contains(researched, "напрям: шорт") || strings.Contains(ranked, "Знайдене передав") {
		t.Errorf("the researcher is told the finalist and its direction, and its answer goes to no other agent: %q", researched)
	}

	if history, _ := agents.History("binance", "", 10); len(history) != 1 || history[0].Answer != "Підсумок." {
		t.Errorf("the chat history holds %+v", history)
	}
}

func TestTheFunnelWithoutFinalistsAnswersWithItsSelection(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{{Role: "assistant", Content: "Нікого не відібрав."}}}
	agents := NewTeam(Deps{Model: model, Funnel: Funnel{Select: Agent{Name: "binance", Prompt: "Відбір"}}})

	answer, err := agents.Scout(context.Background(), "Кандидати?")
	if err != nil || answer.Text != "Нікого не відібрав." {
		t.Errorf("answer %+v, %v", answer, err)
	}

	if _, err := NewTeam(Deps{Model: model}).Scout(context.Background(), "?"); !errors.Is(err, ErrNoFunnel) {
		t.Errorf("without a funnel: %v", err)
	}
}

type ideaTool struct{}

func (ideaTool) Spec() ToolSpec {
	return ToolSpec{Name: "trade_idea"}
}

func (ideaTool) Call(_ context.Context, _ json.RawMessage, turn *Turn) (string, error) {
	turn.Ideas = append(turn.Ideas, Idea{Asset: "ZEC", Direction: "шорт", Why: "падає на обігу", Cancel: "вище 1100"})

	return "ідею передано", nil
}

func TestQuickIdeasComeBackWithTheAnswerAndAreKeptWithIt(t *testing.T) {
	model := &scriptedModel{replies: []ChatMessage{
		call("trade_idea", `{}`),
		{Role: "assistant", Content: "Ринок падає разом, окремо слабша ZEC."},
	}}
	kept := &keptRecords{}
	agents := NewTeam(Deps{
		Model: model,
		Log:   kept,
		Quick: Agent{Name: "binance", Prompt: "Роль.", Reports: []Tool{ideaTool{}}, Rounds: 3, Memory: 5},
	})

	answer, err := agents.Quick(context.Background(), "Швидкі ідеї")
	if err != nil {
		t.Fatal(err)
	}

	if len(answer.Ideas) != 1 || answer.Ideas[0].Asset != "ZEC" || answer.Text == "" {
		t.Errorf("answer %+v", answer)
	}

	if len(kept.records) != 1 || len(kept.records[0].Ideas) != 1 || kept.records[0].Agent != "binance" {
		t.Errorf("kept %+v", kept.records)
	}

	if _, err := NewTeam(Deps{Model: model}).Quick(context.Background(), "?"); !errors.Is(err, ErrNoQuick) {
		t.Errorf("without the quick agent: %v", err)
	}
}
