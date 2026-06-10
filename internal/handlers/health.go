package handlers

import "net/http"

// HealthMsg is the payload returned by the health endpoint when the service is alive.
const HealthMsg = "Good Health"

// Health returns a simple liveness response for probes and local checks.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	h.respondWithJSON(w, http.StatusOK, HealthMsg)
}
