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

func NewServer(pool *pgxpool.Pool, queries *database.Queries, fileStore *storage.FileStore, authToken string) http.Handler {
	h := &Handler{
		pool:      pool,
		queries:   queries,
		fileStore: fileStore,
		authToken: authToken,
	}
	mux := http.NewServeMux()
	addRoutes(mux, h)
	var handler http.Handler = mux
	handler = bearerAuthMiddleware(authToken)(handler)
	return handler
}
