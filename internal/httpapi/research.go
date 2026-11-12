package httpapi

import (
	"net/http"

	"newsbot/internal/research"
)

func routeResearch(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /coins/{asset}/news", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, deps.Research.Of(r.PathValue("asset")))
	})

	mux.HandleFunc("DELETE /news/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeResult(w, deps.Research.Remove(r.PathValue("id")))
	})

	mux.HandleFunc("GET /coins/{asset}/plan", func(w http.ResponseWriter, r *http.Request) {
		plan, kept := deps.Research.Plan(r.PathValue("asset"))
		if !kept {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, resultDTO{Error: research.ErrNoPlan.Error()})

			return
		}

		writeJSON(w, plan)
	})

	mux.HandleFunc("PUT /coins/{asset}/plan", func(w http.ResponseWriter, r *http.Request) {
		var plan research.Plan

		if !readJSON(w, r, &plan) {
			return
		}

		writeResult(w, deps.Research.FixPlan(r.PathValue("asset"), plan))
	})
}

func routeAutopilot(mux *http.ServeMux, deps HandlerDeps) {
	mux.HandleFunc("GET /watches", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, deps.Autopilot.Armed())
	})

	mux.HandleFunc("GET /coins/{asset}/watches", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, deps.Autopilot.Of(r.PathValue("asset")))
	})

	mux.HandleFunc("POST /coins/{asset}/watches", func(w http.ResponseWriter, r *http.Request) {
		var plan research.Scenario

		if !readJSON(w, r, &plan) {
			return
		}

		watch, err := deps.Autopilot.Arm(r.PathValue("asset"), plan)
		if err != nil {
			writeResult(w, err)

			return
		}

		writeJSON(w, watch)
	})

	mux.HandleFunc("PUT /watches/{id}", func(w http.ResponseWriter, r *http.Request) {
		var numbers research.Scenario

		if !readJSON(w, r, &numbers) {
			return
		}

		watch, err := deps.Autopilot.Amend(r.PathValue("id"), numbers)
		if err != nil {
			writeResult(w, err)

			return
		}

		writeJSON(w, watch)
	})

	mux.HandleFunc("DELETE /watches/{id}", func(w http.ResponseWriter, r *http.Request) {
		writeResult(w, deps.Autopilot.Cancel(r.PathValue("id")))
	})
}
