package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	surveyQuestion = "Власник запустив пошук кандидатів і питає: «%s». Оглянь, що відбувається поза Binance, і " +
		"передай у новини інструментів події, яких відбір за рухом не побачить."
	finalistQuestion = "Відбір кандидатів назвав цей інструмент фіналістом, напрям: %s. Чому відібраний: %s. " +
		"Знайди й передай у новини події, що підтверджують або ламають цей напрям."
)

var ErrNoFunnel = errors.New("відбір кандидатів не налаштований")

type Pick struct {
	Asset     string `json:"asset"`
	Direction string `json:"direction"`
	Why       string `json:"why"`
}

type Funnel struct {
	Survey   Agent
	Select   Agent
	Research Agent
	Dig      Agent
	Rank     Agent
}

type dossier struct {
	pick  Pick
	text  string
	turns []*Turn
}

func (a *Team) Scout(ctx context.Context, question string) (Answer, error) {
	funnel := a.deps.Funnel

	if funnel.Select.Name == "" {
		return Answer{}, ErrNoFunnel
	}

	if a.deps.Model == nil {
		return Answer{}, ErrNoModel
	}

	started := time.Now()
	whole := &Turn{}

	whole.absorb(a.survey(ctx, funnel.Survey, question))

	selection, err := a.converse(ctx, funnel.Select, a.remembered(funnel.Select, ""), question, whole)
	if err != nil || len(whole.Picks) == 0 {
		a.record(started, funnel.Select.Name, "", question, selection, whole, err)

		return a.answer(selection, whole), err
	}

	dossiers := a.dig(ctx, funnel, whole.Picks)

	for _, d := range dossiers {
		whole.absorb(d.turns)
	}

	verdict, err := a.converse(ctx, funnel.Rank, nil, brief(question, selection, dossiers), whole)

	a.record(started, funnel.Select.Name, "", question, verdict, whole, err)

	return a.answer(verdict, whole), err
}

func (t *Turn) absorb(turns []*Turn) {
	for _, turn := range turns {
		t.Calls = append(t.Calls, turn.Calls...)
		t.Steps = append(t.Steps, turn.Steps...)
		t.Tokens = t.Tokens.plus(turn.Tokens)
		t.Findings = append(t.Findings, turn.Findings...)
	}
}

func (a *Team) survey(ctx context.Context, researcher Agent, question string) []*Turn {
	if researcher.Name == "" {
		return nil
	}

	started := time.Now()
	turn := &Turn{}
	asked := fmt.Sprintf(surveyQuestion, question)

	text, err := a.converse(ctx, researcher, nil, asked, turn)

	a.record(started, researcher.Name, "", asked, text, turn, err)

	return []*Turn{turn}
}

func (a *Team) dig(ctx context.Context, funnel Funnel, picks []Pick) []dossier {
	dossiers := make([]dossier, len(picks))

	var wg sync.WaitGroup

	for i, pick := range picks {
		wg.Go(func() {
			found := a.research(ctx, funnel.Research, pick)
			dossiers[i] = a.study(ctx, funnel.Dig, pick)
			dossiers[i].turns = append(found, dossiers[i].turns...)
		})
	}

	wg.Wait()

	return dossiers
}

func (a *Team) research(ctx context.Context, researcher Agent, pick Pick) []*Turn {
	if researcher.Name == "" {
		return nil
	}

	started := time.Now()
	turn := &Turn{}
	question := fmt.Sprintf(finalistQuestion, pick.Direction, pick.Why)

	text, err := a.converse(ctx, researcher, a.remembered(researcher, pick.Asset), about(pick.Asset, question), turn)

	a.record(started, researcher.Name, pick.Asset, question, text, turn, err)

	return []*Turn{turn}
}

func (a *Team) study(ctx context.Context, digger Agent, pick Pick) dossier {
	started := time.Now()
	turn := &Turn{}
	question := fmt.Sprintf("Монета %s, напрям: %s. Чому відібрана: %s", pick.Asset, pick.Direction, pick.Why)

	text, err := a.converse(ctx, digger, a.remembered(digger, pick.Asset), question, turn)

	a.record(started, digger.Name, pick.Asset, question, text, turn, err)

	if err != nil {
		text = "досьє не складено: " + err.Error()
	}

	return dossier{pick: pick, text: text, turns: []*Turn{turn}}
}

func brief(question, selection string, dossiers []dossier) string {
	var b strings.Builder

	b.WriteString("Питання власника: " + question + "\n\nВідбір:\n" + selection)

	for _, d := range dossiers {
		fmt.Fprintf(&b, "\n\nДосьє %s (%s):\n%s", d.pick.Asset, d.pick.Direction, d.text)
	}

	return b.String()
}
