package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const validToken = "correct-token"

func newSpyHandler() (http.Handler, *bool) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	return next, &called
}

func TestBearerAuthMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		authHeader string
		wantCalled bool
		wantCode   int
	}{
		{
			name:       "Health bypasses auth",
			target:     "/health",
			authHeader: "",
			wantCalled: true,
			wantCode:   http.StatusOK,
		},
		{
			name:       "Missing header",
			target:     "/documents",
			authHeader: "",
			wantCalled: false,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "Wrong scheme",
			target:     "/documents",
			authHeader: "Token " + validToken,
			wantCalled: false,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "Bearer with wrong token",
			target:     "/documents",
			authHeader: "Bearer wrong-token",
			wantCalled: false,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "Bearer with shorter token",
			target:     "/documents",
			authHeader: "Bearer abc",
			wantCalled: false,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "Bearer with longer token",
			target:     "/documents",
			authHeader: "Bearer " + validToken + "XXX",
			wantCalled: false,
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "Bearer with valid token",
			target:     "/documents",
			authHeader: "Bearer " + validToken,
			wantCalled: true,
			wantCode:   http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, called := newSpyHandler()
			handler := bearerAuthMiddleware(validToken)(next)

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.wantCalled != *called {
				t.Errorf("expected next called %v; got %v", tt.wantCalled, *called)
			}
			if rec.Code != tt.wantCode {
				t.Errorf("expected %v; got %v", tt.wantCode, rec.Code)
			}

			if tt.wantCode == http.StatusUnauthorized {
				if got := rec.Header().Get("WWW-Authenticate"); got == "" {
					t.Errorf("expected WWW-Authenticate header on 401; got empty")
				}
				body, _ := io.ReadAll(rec.Body)
				var resp struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("unmarshal error body: %v", err)
				}
				if resp.Error != "unauthorized" {
					t.Errorf("expected unauthorized; got %v", resp.Error)
				}
			}
		})
	}
}
