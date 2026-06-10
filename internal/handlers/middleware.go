package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

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
