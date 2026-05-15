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
	authToken string
}

func NewHandler(pool *pgxpool.Pool, queries *database.Queries, fileStore *storage.FileStore, authToken string) *Handler {
	return &Handler{
		pool:      pool,
		queries:   queries,
		fileStore: fileStore,
		authToken: authToken,
	}
}

// Register registra todas as rotas HTTP no mux.
func (h *Handler) Register() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /documents", h.CreateDocument)
	mux.HandleFunc("GET /documents", h.ListDocuments)
	mux.HandleFunc("GET /documents/{id}", h.GetDocumentMeta)
	mux.HandleFunc("GET /documents/{id}/content", h.GetDocument)

	return bearerAuthMiddleware(h.authToken)(mux)
}
