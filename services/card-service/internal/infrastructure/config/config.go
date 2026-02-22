package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	GRPCPort    string
	HTTPPort    string
	DatabaseURL string
	KafkaBroker string
	ServiceName string
}

// Validate checks required configuration values.
func (c Config) Validate() {
	if c.DatabaseURL == "" {
		panic("DATABASE_URL environment variable is required")
	}
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		GRPCPort:    getEnv("GRPC_PORT", "8089"),
		HTTPPort:    getEnv("HTTP_PORT", "9089"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		ServiceName: "card-service",
	}
}

// GRPCAddr returns the full gRPC listen address.
func (c Config) GRPCAddr() string {
	return fmt.Sprintf(":%s", c.GRPCPort)
}

// HTTPAddr returns the full HTTP listen address.
func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
