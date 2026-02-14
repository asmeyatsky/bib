package observability

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{name: "debug level", input: "debug", expected: slog.LevelDebug},
		{name: "info level", input: "info", expected: slog.LevelInfo},
		{name: "warn level", input: "warn", expected: slog.LevelWarn},
		{name: "warning level", input: "warning", expected: slog.LevelWarn},
		{name: "error level", input: "error", expected: slog.LevelError},
		{name: "uppercase DEBUG", input: "DEBUG", expected: slog.LevelDebug},
		{name: "uppercase INFO", input: "INFO", expected: slog.LevelInfo},
		{name: "uppercase WARN", input: "WARN", expected: slog.LevelWarn},
		{name: "uppercase ERROR", input: "ERROR", expected: slog.LevelError},
		{name: "mixed case Info", input: "Info", expected: slog.LevelInfo},
		{name: "empty string defaults to info", input: "", expected: slog.LevelInfo},
		{name: "unknown level defaults to info", input: "unknown", expected: slog.LevelInfo},
		{name: "gibberish defaults to info", input: "xyzzy", expected: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInitLoggerJSON(t *testing.T) {
	cfg := LogConfig{
		Level:  "debug",
		Format: "json",
	}

	logger := InitLogger(cfg)
	if logger == nil {
		t.Fatal("InitLogger returned nil")
	}

	// Verify the logger is functional by calling it (should not panic).
	logger.Info("test message", "key", "value")
}

func TestInitLoggerText(t *testing.T) {
	cfg := LogConfig{
		Level:  "info",
		Format: "text",
	}

	logger := InitLogger(cfg)
	if logger == nil {
		t.Fatal("InitLogger returned nil")
	}

	logger.Info("test message", "key", "value")
}

func TestInitLoggerDefaultFormat(t *testing.T) {
	cfg := LogConfig{
		Level:  "warn",
		Format: "",
	}

	logger := InitLogger(cfg)
	if logger == nil {
		t.Fatal("InitLogger returned nil")
	}

	// With empty format, should default to text handler and still work.
	logger.Warn("warning message")
}

func TestInitLoggerAllLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := LogConfig{
				Level:  level,
				Format: "json",
			}

			logger := InitLogger(cfg)
			if logger == nil {
				t.Fatalf("InitLogger returned nil for level %q", level)
			}

			// Verify the logger handles messages without panicking.
			logger.Debug("debug msg")
			logger.Info("info msg")
			logger.Warn("warn msg")
			logger.Error("error msg")
		})
	}
}

func TestInitLoggerSetsDefault(t *testing.T) {
	cfg := LogConfig{
		Level:  "info",
		Format: "json",
	}

	logger := InitLogger(cfg)

	// Verify that slog.Default() was updated.
	defaultLogger := slog.Default()
	if defaultLogger == nil {
		t.Fatal("slog.Default() returned nil after InitLogger")
	}

	// The returned logger and default should be the same instance.
	if logger.Handler() != defaultLogger.Handler() {
		t.Error("InitLogger did not set the default logger")
	}
}
