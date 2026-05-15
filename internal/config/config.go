package config

import (
	"errors"
	"os"

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

// Load lê o arquivo .env (se existir) e o ambiente, retornando erro se algum
// valor obrigatório estiver ausente.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DBURL:       getEnv("DB_URL", ""),
		AuthToken:   getEnv("AUTH_TOKEN", ""),
		Host:        getEnv("SERVER_HOST", "localhost"),
		Port:        getEnv("SERVER_PORT", "8080"),
		StoragePath: getEnv("STORAGE_PATH", "./storage"),
	}

	if cfg.DBURL == "" {
		return nil, errors.New("DB_URL is required, set it in your .env file")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
