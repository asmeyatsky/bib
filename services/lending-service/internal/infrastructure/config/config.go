package config

import (
	"fmt"
	"os"
	"strconv"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type KafkaConfig struct {
	Brokers []string
}

type Config struct {
	GRPCPort    int
	HTTPPort    int
	DB          DatabaseConfig
	Kafka       KafkaConfig
	ServiceName string
}

func (c Config) Validate() {
	if c.DB.Password == "" {
		panic("DB_PASSWORD environment variable is required")
	}
}

func Load() Config {
	return Config{
		GRPCPort: getEnvInt("GRPC_PORT", 9087),
		HTTPPort: getEnvInt("HTTP_PORT", 8087),
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "bib"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "bib_lending"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		},
		ServiceName: "lending-service",
	}
}

func (c Config) GRPCAddr() string {
	return fmt.Sprintf(":%d", c.GRPCPort)
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
