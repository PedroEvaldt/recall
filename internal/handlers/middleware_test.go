package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
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
			h := newTestHandler()
			handler := h.bearerAuthMiddleware()(next)

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

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		target         string
		handler        http.HandlerFunc
		wantStatus     float64
		wantBytesMin   float64
		wantBytesExact *float64
	}{
		{
			name:   "200 with body",
			method: http.MethodGet,
			target: "/documents",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			},
			wantStatus:   200,
			wantBytesMin: 1,
		},
		{
			name:   "custom status",
			method: http.MethodPost,
			target: "/document",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`created`))
			},
			wantStatus:   201,
			wantBytesMin: 1,
		},
		{
			name:   "no write defaults to 200",
			method: http.MethodGet,
			target: "/health",
			handler: func(_ http.ResponseWriter, _ *http.Request) {
			},
			wantStatus:     200,
			wantBytesExact: floatPtr(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := &Handler{
				authToken: validToken,
				logger:    slog.New(slog.NewJSONHandler(&buf, nil)),
			}
			handler := h.loggingMiddleware()(tt.handler)

			req := httptest.NewRequest(tt.method, tt.target, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("unmarshal log entry: %v; raw=%q", err, buf.String())
			}

			if entry["msg"] != "request completed" {
				t.Errorf("expected msg %q; got %v", "request completed", entry["msg"])
			}
			if entry["method"] != tt.method {
				t.Errorf("expected method %q; got %v", tt.method, entry["method"])
			}
			if entry["path"] != tt.target {
				t.Errorf("expected path %q; got %v", tt.target, entry["path"])
			}
			if entry["status"] != tt.wantStatus {
				t.Errorf("expected status %v; got %v", tt.wantStatus, entry["status"])
			}
			if dur, ok := entry["duration"].(float64); !ok || dur <= 0 {
				t.Errorf("expected duration > 0; got %v", entry["duration"])
			}
			bytesWritten, ok := entry["bytes_written"].(float64)
			if !ok {
				t.Fatalf("expected bytes_written to be number; got %T", entry["bytes_written"])
			}
			if tt.wantBytesExact != nil {
				if bytesWritten != *tt.wantBytesExact {
					t.Errorf("expected bytes_written %v; got %v", *tt.wantBytesExact, bytesWritten)
				}
			} else if bytesWritten < tt.wantBytesMin {
				t.Errorf("expected bytes_written >= %v; got %v", tt.wantBytesMin, bytesWritten)
			}
		})
	}
}

func floatPtr(f float64) *float64 { return &f }
