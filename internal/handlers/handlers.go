package handlers

import (
	"log/slog"
	"net/http"

	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler groups the dependencies used by the application's HTTP handlers.
type Handler struct {
	pool      *pgxpool.Pool
	queries   *database.Queries
	fileStore *storage.FileStore
	authToken string
	logger    *slog.Logger
}

// New build a new Handler struct
func New(pool *pgxpool.Pool, queries *database.Queries, fileStore *storage.FileStore, authToken string, logger *slog.Logger) *Handler {
	return &Handler{
		pool:      pool,
		queries:   queries,
		fileStore: fileStore,
		authToken: authToken,
		logger:    logger,
	}
}

// NewServer wires routes and middleware into a single HTTP handler.
func NewServer(pool *pgxpool.Pool, queries *database.Queries, fileStore *storage.FileStore, authToken string, logger *slog.Logger) http.Handler {
	h := New(pool, queries, fileStore, authToken, logger)
	mux := http.NewServeMux()
	addRoutes(mux, h)
	var handler http.Handler = mux
	handler = h.bearerAuthMiddleware()(handler)
	handler = h.metricsMiddleware()(handler)
	handler = h.loggingMiddleware()(handler)
	return handler
}
