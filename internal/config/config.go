package config

import (
	"errors"

	"github.com/joho/godotenv"
)

// Config concentra todas as variáveis de ambiente lidas no startup.
type Config struct {
	DBURL       string
	AuthToken   string
	Host        string
	Port        string
	StoragePath string
}

func LoadFrom(getenv func(string) string) (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DBURL:       getEnvDefault(getenv, "DB_URL", ""),
		AuthToken:   getEnvDefault(getenv, "AUTH_TOKEN", ""),
		Host:        getEnvDefault(getenv, "SERVER_HOST", "localhost"),
		Port:        getEnvDefault(getenv, "SERVER_PORT", "8080"),
		StoragePath: getEnvDefault(getenv, "STORAGE_PATH", "./storage"),
	}

	if cfg.AuthToken == "" {
		return nil, errors.New("AUTH_TOKEN is required, set it in your .env file")
	}

	if cfg.DBURL == "" {
		return nil, errors.New("DB_URL is required, set it in your .env file")
	}
	return cfg, nil
}

func getEnvDefault(getenv func(string) string, key, fallback string) string {
	if value := getenv(key); value != "" {
		return value
	}
	return fallback
}
