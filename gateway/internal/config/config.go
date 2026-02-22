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
	RateLimit     int // requests per second per client
	LogLevel      string
	LogFormat     string
}

// Validate checks required configuration values.
func (c Config) Validate() {
	if c.JWTSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}
}

// Load reads configuration from environment variables with sensible defaults.
// Service addresses accept both LEDGER_ADDR and LEDGER_SERVICE_ADDR patterns
// to match docker-compose conventions.
func Load() Config {
	return Config{
		HTTPPort:      getEnvInt("HTTP_PORT", 8080),
		LedgerAddr:    getEnvWithAlt("LEDGER_ADDR", "LEDGER_SERVICE_ADDR", "localhost:9081"),
		AccountAddr:   getEnvWithAlt("ACCOUNT_ADDR", "ACCOUNT_SERVICE_ADDR", "localhost:9082"),
		FXAddr:        getEnvWithAlt("FX_ADDR", "FX_SERVICE_ADDR", "localhost:9083"),
		DepositAddr:   getEnvWithAlt("DEPOSIT_ADDR", "DEPOSIT_SERVICE_ADDR", "localhost:9084"),
		IdentityAddr:  getEnvWithAlt("IDENTITY_ADDR", "IDENTITY_SERVICE_ADDR", "localhost:9085"),
		PaymentAddr:   getEnvWithAlt("PAYMENT_ADDR", "PAYMENT_SERVICE_ADDR", "localhost:9086"),
		LendingAddr:   getEnvWithAlt("LENDING_ADDR", "LENDING_SERVICE_ADDR", "localhost:9087"),
		FraudAddr:     getEnvWithAlt("FRAUD_ADDR", "FRAUD_SERVICE_ADDR", "localhost:9088"),
		CardAddr:      getEnvWithAlt("CARD_ADDR", "CARD_SERVICE_ADDR", "localhost:9089"),
		ReportingAddr: getEnvWithAlt("REPORTING_ADDR", "REPORTING_SERVICE_ADDR", "localhost:9090"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
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

// getEnvWithAlt returns the value of the primary env var, falling back to
// the alternate env var, then the default value.
func getEnvWithAlt(primary, alt, defaultVal string) string {
	if val, ok := os.LookupEnv(primary); ok {
		return val
	}
	if val, ok := os.LookupEnv(alt); ok {
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
