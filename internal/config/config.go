package config

import (
	"github.com/PedroEvaldt/recall/internal/storage/database"
)

type Config struct {
	db *db.Queries
}

func GetConfig(db *db.Queries) *Config {
	return &Config{
		db: db,
	}
}
