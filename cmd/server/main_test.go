//go:build integration

package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	healthCheckTimeout = 5 * time.Second
	healthCheckPoll    = 20 * time.Millisecond
	shutdownTimeout    = 15 * time.Second
)

// TestServerMain boots run end-to-end against a real postgres container and
// verifies that the HTTP server responds and shuts down gracefully on cancel.
func TestServerMain(t *testing.T) {
	dbURL := startPostgres(t)
	port := freePort(t)
	storageDir := t.TempDir()
	env := map[string]string{
		"DB_URL":       dbURL,
		"AUTH_TOKEN":   "test-token",
		"SERVER_HOST":  "127.0.0.1",
		"SERVER_PORT":  port,
		"STORAGE_PATH": storageDir,
	}
	getenv := func(k string) string { return env[k] }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stderr bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- run(ctx, getenv, &stderr) }()
	baseURL := "http://127.0.0.1:" + port
	waitForHealth(t, baseURL+"/health")
	respHealth, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("could not GET /health: %v", err)
	}
	defer respHealth.Body.Close()
	if respHealth.StatusCode != http.StatusOK {
		t.Errorf("expected %v; got %v", http.StatusOK, respHealth.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/documents", nil)
	if err != nil {
		t.Fatalf("could not build /documents request: %v", err)
	}
	respDocuments, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("could not GET /documents: %v", err)
	}
	defer respDocuments.Body.Close()
	if respDocuments.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected %v; got %v", http.StatusUnauthorized, respDocuments.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(shutdownTimeout):
		t.Fatalf("run did not shut down within 15s")
	}
}

// startPostgres boots a postgres testcontainer and returns its connection string.
func startPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	dbName := "documents"
	dbUser := "postgres"
	dbPassword := "postgres"

	ctr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithInitScripts(
			filepath.Join("..", "..", "internal", "handlers", "testdata", "schema.sql"),
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

	return dbURL
}

// freePort returns a TCP port available for binding on the loopback interface.
func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not get free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("could not close port listener: %v", err)
	}
	return strconv.Itoa(port)
}

// waitForHealth polls url until it responds 200 OK or the deadline expires.
func waitForHealth(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(healthCheckTimeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			time.Sleep(healthCheckPoll)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		resp.Body.Close()
	}
	t.Fatalf("server did not become healthy in %s", healthCheckTimeout)
}
