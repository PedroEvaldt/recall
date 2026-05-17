package handlers

import "net/http"

// Health returns a simple liveness response for probes and local checks.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, "Good Health")
}
