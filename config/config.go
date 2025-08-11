package config

import (
	"log/slog"
	"os"
)

type Config struct {
	// MongoDB configuration
	MongoURI     string
	DatabaseName string

	// Webhook configuration
	VerifyToken string

	// Server configuration
	Port string
}

func LoadConfig() *Config {
	cfg := &Config{
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName: getEnv("MONGO_DB_NAME", "facebook_bot"),
		VerifyToken:  getEnv("WEBHOOK_VERIFY_TOKEN", "webhook_verify_token"),
		Port:         getEnv("PORT", "8080"),
	}

	// Validate required configuration
	if cfg.MongoURI == "" {
		slog.Error("MONGO_URI not set")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
