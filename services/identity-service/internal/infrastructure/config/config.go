package config

import (
	"os"
	"strconv"
)

// Config holds all service configuration loaded from environment variables.
type Config struct {
	Telemetry TelemetryConfig
	LogLevel  string
	LogFormat string
	Kafka     KafkaConfig
	DB        DBConfig
	HTTPPort  int
	GRPCPort  int
}

type DBConfig struct {
	Host     string
	User     string
	Password string
	Name     string
	SSLMode  string
	Port     int
	MaxConns int32
	MinConns int32
}

type KafkaConfig struct {
	Brokers []string
}

type TelemetryConfig struct {
	OTLPEndpoint string
	ServiceName  string
}

// Validate checks required configuration values.
func (c Config) Validate() {
	if c.DB.Password == "" {
		panic("DB_PASSWORD environment variable is required")
	}
}

// Load reads configuration from environment variables with defaults.
func Load() Config {
	return Config{
		HTTPPort: getEnvInt("HTTP_PORT", 8085),
		GRPCPort: getEnvInt("GRPC_PORT", 9085),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "bib"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "bib_identity"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
			MaxConns: int32(getEnvInt("DB_MAX_CONNS", 20)), //nolint:gosec // bounded by env config
			MinConns: int32(getEnvInt("DB_MIN_CONNS", 5)),  //nolint:gosec // bounded by env config
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		},
		Telemetry: TelemetryConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
			ServiceName:  "identity-service",
		},
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
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
