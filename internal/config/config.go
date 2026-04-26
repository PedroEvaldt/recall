package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL       string
	Port        string
	StoragePath string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	config := &Config{
		DBURL:       getEnv("DB_URL", ""),
		Port:        getEnv("SERVER_PORT", "8080"),
		StoragePath: getEnv("STORAGE_PATH", "./storage"),
	}
	if config.DBURL == "" {
		return &Config{}, errors.New("failed to get DBULR, define it in your .env file")
	}
	return config, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	return value
}
