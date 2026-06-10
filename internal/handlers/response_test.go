package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestHandler() *Handler {
	return &Handler{
		authToken: validToken,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestRespondWithJSON(t *testing.T) {
	type requestBody struct {
		Msg string `json:"msg"`
	}
	tests := []struct {
		name    string
		header  string
		body    any
		code    int
		wantErr bool
	}{
		{
			name:    "payload valido",
			header:  "application/json",
			body:    requestBody{Msg: "simple request"},
			code:    http.StatusOK,
			wantErr: false,
		},
		{
			name:    "payload invalido",
			header:  "",
			body:    make(chan int),
			code:    http.StatusInternalServerError,
			wantErr: true,
		},
		{
			name:    "payload nil",
			header:  "application/json",
			body:    nil,
			code:    http.StatusOK,
			wantErr: false,
		},
		{
			name:    "different code",
			header:  "application/json",
			body:    requestBody{Msg: "simple request"},
			code:    http.StatusCreated,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h := newTestHandler()
			h.respondWithJSON(rec, tt.code, tt.body)
			if rec.Code != tt.code {
				t.Errorf("expected code %v; got %v", tt.code, rec.Code)
			}
			if !tt.wantErr {
				if header := rec.Header().Get("Content-Type"); header != tt.header {
					t.Errorf("expected header %v; got %v", tt.header, header)
				}
				if body, ok := tt.body.(requestBody); ok {
					var result requestBody
					if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
						t.Fatalf("unmarshaling: %v", err)
					}
					if result.Msg != body.Msg {
						t.Errorf("expected msg %s; got %v", body.Msg, result.Msg)
					}
				}
				if tt.body == nil {
					if rec.Body.String() != "null" {
						t.Errorf("expected body to be null; got %s", rec.Body.String())
					}
				}

			}
		})
	}
}

func TestRespondWithError(t *testing.T) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	var (
		errMsg = "this is an error"
		result errorResponse
	)
	rec := httptest.NewRecorder()
	h := newTestHandler()

	h.respondWithError(rec, http.StatusNotFound, errMsg)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected code %v; got %v", http.StatusNotFound, rec.Code)
	}
	if header := rec.Header().Get("Content-Type"); header != "application/json" {
		t.Errorf("expected header application/json; got %v", header)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if result.Error != errMsg {
		t.Errorf("expected body %s; got %s", errMsg, result.Error)
	}
}

func TestRespondUnauthorized(t *testing.T) {
	type unauthorizedMsg struct {
		Error string `json:"error"`
	}
	var result unauthorizedMsg

	rec := httptest.NewRecorder()
	h := newTestHandler()
	h.respondUnauthorized(rec)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected code %v; got %v", http.StatusUnauthorized, rec.Code)
	}
	if header := rec.Header().Get("Content-Type"); header != "application/json" {
		t.Errorf("expected header application/json; got %v", header)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if result.Error != "unauthorized" {
		t.Errorf("expected body unauthorized; got %s", result.Error)
	}
	if header := rec.Header().Get("WWW-Authenticate"); header != `Bearer realm="restricted", charset="UTF-8"` {
		t.Errorf("expected header Bearer realm=restricted; got %v", rec.Header().Get("WWW-Authenticate"))
	}
}
