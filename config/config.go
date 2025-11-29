package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisAddr   string
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
	}

	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
