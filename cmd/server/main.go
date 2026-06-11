// Package main is the entrypoint for the recall HTTP server binary.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PedroEvaldt/recall/internal/config"
	"github.com/PedroEvaldt/recall/internal/handlers"
	"github.com/PedroEvaldt/recall/internal/logging"
	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/PedroEvaldt/recall/internal/storage/database"
)

func run(ctx context.Context, getenv func(string) string, stderr io.Writer) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := logging.New(stderr, getenv("LOG_LEVEL"), getenv("LOG_FORMAT"))

	cfg, err := config.LoadFrom(getenv)
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		return fmt.Errorf("config: %w", err)
	}

	pool, err := database.NewPool(ctx, cfg.DBURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	queries := database.New(pool)

	fileStore, err := storage.NewFileStore(cfg.StoragePath)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	handler := handlers.NewServer(pool, queries, fileStore, cfg.AuthToken, logger)

	srv := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Getenv, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
