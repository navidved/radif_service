// Package config loads application configuration from environment variables.
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the service.
type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
	AppEnv      string

	// Object storage (S3-compatible: MinIO locally, ArvanCloud in production)
	StorageEndpoint   string
	StorageAccessKey  string
	StorageSecretKey  string
	StorageBucket     string
	StorageUseSSL     bool
	StoragePublicBase string // browser-accessible base URL, e.g. "http://localhost:9000/avatars"
}

// Load reads configuration from a .env file (if present) and environment variables.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://radif:radif@postgres:5432/radif?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "change_me_in_production"),
		Port:        getEnv("PORT", "8080"),
		AppEnv:      getEnv("APP_ENV", "development"),

		StorageEndpoint:   getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:  getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:  getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:     getEnv("STORAGE_BUCKET", "avatars"),
		StorageUseSSL:     getEnv("STORAGE_USE_SSL", "false") == "true",
		StoragePublicBase: getEnv("STORAGE_PUBLIC_BASE", "http://localhost:9000/avatars"),
	}
}

// IsProduction returns true when the app is running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
