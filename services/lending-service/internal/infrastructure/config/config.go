package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the lending service.
type Config struct {
	GRPCPort    int
	HTTPPort    int
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
		GRPCPort:    getIntEnv("GRPC_PORT", 8087),
		HTTPPort:    getIntEnv("HTTP_PORT", 9087),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		ServiceName: "lending-service",
	}
}

// GRPCAddr returns the formatted gRPC listen address.
func (c Config) GRPCAddr() string {
	return fmt.Sprintf(":%d", c.GRPCPort)
}

// HTTPAddr returns the formatted HTTP listen address.
func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
