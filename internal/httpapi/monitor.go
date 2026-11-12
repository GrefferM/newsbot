package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"newsbot/internal/monitor"
	"newsbot/internal/news"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

func routeMonitor(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, health(deps.Status))
	})

	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, events(deps.Recent.Latest(limit(r))))
	})

	mux.HandleFunc("DELETE /events", func(w http.ResponseWriter, r *http.Request) {
		writeResult(w, deps.Recent.Remove(r.URL.Query().Get("id")))
	})
}

type sourceDTO struct {
	LastOK    time.Time `json:"last_ok,omitzero"`
	LastError string    `json:"last_error,omitempty"`
	LastErrAt time.Time `json:"last_error_at,omitzero"`
	Items     int       `json:"items"`
	NewTotal  int       `json:"new_total"`
}

type healthDTO struct {
	Started time.Time            `json:"started"`
	Sources map[string]sourceDTO `json:"sources"`
}

type eventDTO struct {
	ID       string    `json:"id"`
	Source   string    `json:"source"`
	Category string    `json:"category"`
	Kind     string    `json:"kind"`
	Title    string    `json:"title"`
	URL      string    `json:"url,omitempty"`
	Time     time.Time `json:"time"`
	Symbols  []string  `json:"symbols,omitempty"`
	Details  []string  `json:"details,omitempty"`
}

func health(status *monitor.Status) healthDTO {
	started, sources := status.Snapshot()
	dto := healthDTO{Started: started, Sources: make(map[string]sourceDTO, len(sources))}

	for name, s := range sources {
		dto.Sources[name] = sourceDTO(s)
	}

	return dto
}

func events(list []news.Event) []eventDTO {
	out := make([]eventDTO, 0, len(list))

	for _, e := range list {
		out = append(out, eventDTO{
			ID:       e.ID,
			Source:   e.Source,
			Category: string(e.Category),
			Kind:     e.Kind,
			Title:    e.Title,
			URL:      e.URL,
			Time:     e.Time,
			Symbols:  symbols(e.Symbols),
			Details:  e.Details,
		})
	}

	return out
}

func symbols(list []news.Symbol) []string {
	out := make([]string, 0, len(list))

	for _, s := range list {
		out = append(out, string(s))
	}

	return out
}

func limit(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 || n > maxLimit {
		return defaultLimit
	}

	return n
}
