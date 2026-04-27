package handlers

import "net/http"

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, "Good Health")
}
