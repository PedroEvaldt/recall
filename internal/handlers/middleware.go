package handlers

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
)

func (h *Handler) loggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := httpsnoop.CaptureMetrics(next, w, r)
			h.logger.LogAttrs(r.Context(), slog.LevelInfo, "request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", m.Code),
				slog.Duration("duration", m.Duration),
				slog.Int64("bytes_written", m.Written),
			)
		})
	}
}

func (h *Handler) bearerAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				h.respondUnauthorized(w)
				return
			}
			authToken, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				h.respondUnauthorized(w)
				return
			}
			equal := subtle.ConstantTimeCompare([]byte(authToken), []byte(h.authToken))
			if equal != 1 {
				h.respondUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
