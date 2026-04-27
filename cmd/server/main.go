package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/PedroEvaldt/recall/internal/config"
	"github.com/PedroEvaldt/recall/internal/handlers"
	"github.com/PedroEvaldt/recall/internal/storage"
	"github.com/PedroEvaldt/recall/internal/storage/database"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := database.NewPool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	queries := database.New(pool)

	fileStore, err := storage.NewFileStore(cfg.StoragePath)
	if err != nil {
		log.Fatalf("file store: %v", err)
	}

	handler := handlers.NewHandler(pool, queries, fileStore)

	mux := http.NewServeMux()
	handler.Register(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server: %v", err)
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown: %v", err)
	}
}
