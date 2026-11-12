package httpapi

import (
	"encoding/json"
	"net/http"

	"newsbot/internal/agenda"
	"newsbot/internal/agents"
	"newsbot/internal/autopilot"
	"newsbot/internal/autotrade"
	"newsbot/internal/monitor"
	"newsbot/internal/radar"
	"newsbot/internal/research"
	"newsbot/internal/trading"
	"newsbot/internal/watchlist"
)

type HandlerDeps struct {
	Status    *monitor.Status
	Recent    *monitor.Recent
	Watchlist *watchlist.Service
	Live      Live
	Trader    *trading.Trader
	Agents    *agents.Team
	Research  *research.Keeper
	Autopilot *autopilot.Service
	Auto      *autotrade.Service
	Radar     *radar.Service
	Agenda    *agenda.Service
}

func NewHandler(deps HandlerDeps) http.Handler {
	mux := http.NewServeMux()

	routeMonitor(mux, deps)
	routeWatchlist(mux, deps)
	routeTrading(mux, deps)
	routeResearch(mux, deps)
	routeAutopilot(mux, deps)
	routeRadar(mux, deps)
	routeAgenda(mux, deps)
	routeAgents(mux, deps)

	return mux
}

type resultDTO struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		writeResult(w, err)
	}

	return err == nil
}

func writeResult(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		writeJSON(w, resultDTO{Error: err.Error()})

		return
	}

	writeJSON(w, resultDTO{OK: true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	_ = enc.Encode(v)
}
