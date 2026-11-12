package httpapi

import (
	"net/http"
	"time"
)

const (
	calendarAhead  = 90 * 24 * time.Hour
	calendarBehind = 7 * 24 * time.Hour
)

func routeAgenda(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /calendar", func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now().UTC()

		writeJSON(w, deps.Agenda.Between(now.Add(-calendarBehind), now.Add(calendarAhead)))
	})
}
