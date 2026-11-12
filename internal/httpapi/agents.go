package httpapi

import (
	"net/http"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/research"
)

func routeAgents(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("POST /agents/{name}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Question string `json:"question"`
			Topic    string `json:"topic"`
		}

		if !readJSON(w, r, &req) {
			return
		}

		answer, err := deps.Agents.Ask(r.Context(), r.PathValue("name"), req.Topic, req.Question)

		writeAnswer(w, answer, err)
	})

	mux.HandleFunc("POST /scout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Question string `json:"question"`
		}

		if !readJSON(w, r, &req) {
			return
		}

		answer, err := deps.Agents.Scout(r.Context(), req.Question)

		writeAnswer(w, answer, err)
	})

	mux.HandleFunc("POST /ideas", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Question string `json:"question"`
		}

		if !readJSON(w, r, &req) {
			return
		}

		answer, err := deps.Agents.Quick(r.Context(), req.Question)

		writeAnswer(w, answer, err)
	})

	mux.HandleFunc("GET /agents/{name}/history", func(w http.ResponseWriter, r *http.Request) {
		records, err := deps.Agents.History(r.PathValue("name"), r.URL.Query().Get("topic"), limit(r))
		if err != nil {
			writeResult(w, err)

			return
		}

		writeJSON(w, history(records))
	})

	mux.HandleFunc("DELETE /agents/{name}/history", func(w http.ResponseWriter, r *http.Request) {
		writeResult(w, deps.Agents.Forget(r.PathValue("name"), r.URL.Query().Get("topic")))
	})
}

func writeAnswer(w http.ResponseWriter, answer agents.Answer, err error) {
	if err != nil {
		writeResult(w, err)

		return
	}

	writeJSON(w, answerOf(answer))
}

type exchangeDTO struct {
	At       time.Time     `json:"at"`
	Question string        `json:"question"`
	Answer   string        `json:"answer"`
	Calls    []string      `json:"calls"`
	Ideas    []agents.Idea `json:"ideas,omitempty"`
	Failed   bool          `json:"failed,omitempty"`
}

func history(records []agents.Record) []exchangeDTO {
	out := make([]exchangeDTO, 0, len(records))

	for _, r := range records {
		calls := make([]string, 0, len(r.Steps))

		for _, step := range r.Steps {
			calls = append(calls, step.Tool)
		}

		exchange := exchangeDTO{At: r.At, Question: r.Question, Answer: r.Answer, Calls: calls, Ideas: r.Ideas}

		if r.Error != "" {
			exchange.Answer, exchange.Failed = r.Error, true
		}

		out = append(out, exchange)
	}

	return out
}

type answerDTO struct {
	Text      string              `json:"text"`
	Findings  []research.Finding  `json:"findings"`
	Scenarios []research.Scenario `json:"scenarios"`
	Calls     []string            `json:"calls"`
	Ideas     []agents.Idea       `json:"ideas"`
}

func answerOf(a agents.Answer) answerDTO {
	dto := answerDTO{
		Text:      a.Text,
		Findings:  a.Findings,
		Scenarios: a.Scenarios,
		Calls:     a.Calls,
		Ideas:     a.Ideas,
	}

	if dto.Ideas == nil {
		dto.Ideas = []agents.Idea{}
	}

	if dto.Findings == nil {
		dto.Findings = []research.Finding{}
	}

	if dto.Scenarios == nil {
		dto.Scenarios = []research.Scenario{}
	}

	return dto
}
