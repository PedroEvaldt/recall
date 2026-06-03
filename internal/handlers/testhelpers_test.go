//go:build integration

package handlers_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/PedroEvaldt/recall/internal/handlers"
	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/PedroEvaldt/recall/internal/storage/database"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestServer(t *testing.T) (*handlers.Handler, func(), string) {
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

	dir := t.TempDir()
	fs, err := storage.NewFileStore(dir)
	if err != nil {
		t.Fatalf("could not create temp filestore: %v", err)
	}
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("could not create database pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := handlers.New(pool, database.New(pool), fs, "test-token")

	reset := func() {
		if err := ctr.Restore(ctx); err != nil {
			t.Fatalf("restore: %v", err)
		}
		pool.Reset()
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("cleanup dir: %v", err)
		}
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
				t.Fatalf("remove dir %s: %v", e.Name(), err)
			}
		}
	}
	return h, reset, dir
}

func newMultipartReq(t *testing.T, hasFile bool, title, filename string) *http.Request {
	var buf bytes.Buffer
	multWriter := multipart.NewWriter(&buf)
	if err := multWriter.WriteField("title", title); err != nil {
		t.Fatalf("could not write title field in multipart writer: %v", err)
	}
	if hasFile {
		fileWriter, err := multWriter.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("could not create writer from multipart writer: %v", err)
		}
		if _, err := fmt.Fprint(fileWriter, "#Go struct file body"); err != nil {
			t.Fatalf("could not write body into file writer: %v", err)
		}
	}
	if err := multWriter.Close(); err != nil {
		t.Fatalf("could not close multipart writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/documents", &buf)
	req.Header.Set("Content-Type", multWriter.FormDataContentType())

	return req
}
