package config

import (
	"os"
	"strconv"
)

// Config holds all service configuration loaded from environment variables.
type Config struct {
	HTTPPort  int
	GRPCPort  int
	DB        DBConfig
	Kafka     KafkaConfig
	Telemetry TelemetryConfig
	LogLevel  string
	LogFormat string
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
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
		HTTPPort: getEnvInt("HTTP_PORT", 8086),
		GRPCPort: getEnvInt("GRPC_PORT", 9086),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "bib"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "bib_payment"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
			MaxConns: int32(getEnvInt("DB_MAX_CONNS", 20)),
			MinConns: int32(getEnvInt("DB_MIN_CONNS", 5)),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		},
		Telemetry: TelemetryConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
			ServiceName:  "payment-service",
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
