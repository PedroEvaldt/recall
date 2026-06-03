package handlers

import "net/http"

const HealthMsg = "Good Health"

// Health returns a simple liveness response for probes and local checks.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, HealthMsg)
}
