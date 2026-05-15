package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func bearerAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w)
				return
			}
			authToken, found := strings.CutPrefix(authHeader, "Bearer ")
			if !found {
				respondUnauthorized(w)
				return
			}
			equal := subtle.ConstantTimeCompare([]byte(authToken), []byte(token))
			if equal != 1 {
				respondUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
