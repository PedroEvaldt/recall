package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL string
	Port  string
}

func GetConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		DBURL: getEnv("DB_URL", ""),
		Port:  getEnv("SERVER_PORT", ""),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		value = fallback
	}
	return value
}
