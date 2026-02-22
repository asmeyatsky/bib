package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the account service.
type Config struct {
	// gRPC server port
	GRPCPort int
	// HTTP metrics/health port
	HTTPPort int
	// Database configuration
	Database DatabaseConfig
	// Kafka configuration
	Kafka KafkaConfig
	// Service name for observability
	ServiceName string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// DSN returns the PostgreSQL data source name.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// KafkaConfig holds Kafka connection settings.
type KafkaConfig struct {
	Brokers []string
}

// Validate checks required configuration values.
func (c Config) Validate() {
	if c.Database.Password == "" {
		panic("DB_PASSWORD environment variable is required")
	}
}

// Load reads configuration from environment variables with defaults.
func Load() Config {
	return Config{
		GRPCPort:    getEnvInt("GRPC_PORT", 8082),
		HTTPPort:    getEnvInt("HTTP_PORT", 9082),
		ServiceName: getEnv("SERVICE_NAME", "account-service"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "bib"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "bib_account"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
