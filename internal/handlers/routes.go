package handlers

import (
	"net/http"

	"github.com/PedroEvaldt/recall/internal/metrics"
)

func addRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /health", h.Health)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.HandleFunc("POST /documents", h.CreateDocument)
	mux.HandleFunc("GET /documents", h.ListDocuments)
	mux.HandleFunc("GET /documents/{id}", h.GetDocumentMeta)
	mux.HandleFunc("GET /documents/{id}/content", h.GetDocument)
}
