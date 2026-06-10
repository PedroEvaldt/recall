package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("respondWithJSON marshal", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(body); err != nil {
		h.logger.Error("respondWithJSON write", slog.String("error", err.Error()))
	}
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, msg string) {
	h.respondWithJSON(w, code, map[string]string{"error": msg})
}

func (h *Handler) respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="restricted", charset="UTF-8"`)
	h.respondWithError(w, http.StatusUnauthorized, "unauthorized")
}
