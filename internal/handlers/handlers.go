package handlers

import (
	"net/http"

	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler agrupa as dependências usadas pelos HTTP handlers da aplicação.
type Handler struct {
	pool      *pgxpool.Pool
	queries   *database.Queries
	fileStore *storage.FileStore
}

func NewHandler(pool *pgxpool.Pool, queries *database.Queries, fileStore *storage.FileStore) *Handler {
	return &Handler{
		pool:      pool,
		queries:   queries,
		fileStore: fileStore,
	}
}

// Register registra todas as rotas HTTP no mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /documents", h.CreateDocument)
	// mux.HandleFunc("GET /documents", h.ListDocuments)
}
