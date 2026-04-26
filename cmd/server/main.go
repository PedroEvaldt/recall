package main

import (
	"context"
	"log"

	"github.com/PedroEvaldt/recall/internal/config"
	"github.com/PedroEvaldt/recall/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config %v", err)
	}

	pool, err := storage.NewPool(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()
}
