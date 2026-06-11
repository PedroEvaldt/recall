package handlers

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/PedroEvaldt/recall/internal/metrics"
	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"
)

type ctxKey string

const (
	requestIDKey     ctxKey = "request_id"
	maxRequestIDSize int    = 128
)

// RequestID returns the request_id propagated through the context by the
// logging middleware, or "" if none is set.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func (h *Handler) loggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if id == "" || len(id) > maxRequestIDSize {
				id = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", id)
			r = r.WithContext(context.WithValue(r.Context(), requestIDKey, id))
			m := httpsnoop.CaptureMetrics(next, w, r)
			h.logger.LogAttrs(r.Context(), slog.LevelInfo, "request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", m.Code),
				slog.Duration("duration", m.Duration),
				slog.Int64("bytes_written", m.Written),
				slog.String("request_id", id),
			)
		})
	}
}

func (h *Handler) bearerAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
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

func (h *Handler) metricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := httpsnoop.CaptureMetrics(next, w, r)
			method := r.Method
			route := r.Pattern
			if route == "" {
				route = "unknown"
			}
			status := strconv.Itoa(m.Code)
			metrics.HTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(method, route).Observe(m.Duration.Seconds())
		})
	}
}
