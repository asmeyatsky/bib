package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the fraud service.
type Config struct {
	GRPCPort    string
	HTTPPort    string
	DatabaseURL string
	KafkaBroker string
	Environment string
	LogLevel    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		GRPCPort:    getEnv("GRPC_PORT", "8088"),
		HTTPPort:    getEnv("HTTP_PORT", "9088"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://bib:bib@localhost:5432/bib_fraud?sslmode=disable"),
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

// GRPCAddress returns the full gRPC listen address.
func (c *Config) GRPCAddress() string {
	return fmt.Sprintf(":%s", c.GRPCPort)
}

// HTTPAddress returns the full HTTP listen address.
func (c *Config) HTTPAddress() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
