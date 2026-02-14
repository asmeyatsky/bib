package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the API gateway.
type Config struct {
	HTTPPort      int
	LedgerAddr    string
	AccountAddr   string
	FXAddr        string
	DepositAddr   string
	IdentityAddr  string
	PaymentAddr   string
	LendingAddr   string
	FraudAddr     string
	CardAddr      string
	ReportingAddr string
	JWTSecret     string
	RateLimit     int // requests per second
	LogLevel      string
	LogFormat     string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		HTTPPort:      getEnvInt("HTTP_PORT", 8080),
		LedgerAddr:    getEnv("LEDGER_ADDR", "localhost:9081"),
		AccountAddr:   getEnv("ACCOUNT_ADDR", "localhost:9082"),
		FXAddr:        getEnv("FX_ADDR", "localhost:9083"),
		DepositAddr:   getEnv("DEPOSIT_ADDR", "localhost:9084"),
		IdentityAddr:  getEnv("IDENTITY_ADDR", "localhost:9085"),
		PaymentAddr:   getEnv("PAYMENT_ADDR", "localhost:9086"),
		LendingAddr:   getEnv("LENDING_ADDR", "localhost:9087"),
		FraudAddr:     getEnv("FRAUD_ADDR", "localhost:9088"),
		CardAddr:      getEnv("CARD_ADDR", "localhost:9089"),
		ReportingAddr: getEnv("REPORTING_ADDR", "localhost:9090"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-in-prod"),
		RateLimit:     getEnvInt("RATE_LIMIT", 100),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LogFormat:     getEnv("LOG_FORMAT", "json"),
	}
}

// getEnv returns the value of an environment variable or a default.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// getEnvInt returns the integer value of an environment variable or a default.
func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
