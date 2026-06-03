package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PedroEvaldt/recall/internal/handlers"
)

func TestHealth(t *testing.T) {
	h := &handlers.Handler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", http.StatusText(rec.Code))
	}

	header := rec.Header()
	if contentType := header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json; got %v", contentType)
	}

	got := rec.Body.String()
	want := `"` + handlers.HealthMsg + `"`
	if got != want {
		t.Errorf("expected response to be %s; got %s", want, got)
	}
}
