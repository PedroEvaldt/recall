//go:build integration

package handlers_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PedroEvaldt/recall/internal/handlers"
	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestServer(t *testing.T) (*handlers.Handler, *postgres.PostgresContainer) {
	t.Helper()
	ctx := context.Background()
	dbName := "documents"
	dbUser := "postgres"
	dbPassword := "postgres"

	ctr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithInitScripts(
			filepath.Join("testdata", "schema.sql"),
		),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.WithSQLDriver("pgx"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("could not start postgres container: %v", err)
	}
	testcontainers.CleanupContainer(t, ctr)

	dbURL, err := ctr.ConnectionString(ctx, "sslmode=disable", "application_name=test")
	if err != nil {
		t.Fatalf("could not get container connection string: %v", err)
	}

	if err := ctr.Snapshot(ctx); err != nil {
		t.Fatalf("could not make database snapshot: %v", err)
	}

	fs, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("could not create temp filestore: %v", err)
	}
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("could not create database pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := handlers.New(pool, database.New(pool), fs, "test-token")

	return h, ctr
}
