package ml

import (
	"context"
	"log/slog"
)

// StubModelClient implements port.MLModelClient as a stub for development.
// In production, this would call an external ML model service (e.g., SageMaker, Vertex AI).
type StubModelClient struct {
	logger *slog.Logger
}

// NewStubModelClient creates a new stub ML model client.
func NewStubModelClient(logger *slog.Logger) *StubModelClient {
	return &StubModelClient{logger: logger}
}

// Predict returns a default risk score. This is a stub implementation.
// In production, this would send features to an ML model and receive a prediction.
func (c *StubModelClient) Predict(ctx context.Context, features map[string]interface{}) (float64, error) {
	c.logger.Debug("stub ML model prediction requested",
		slog.Int("feature_count", len(features)),
	)

	// Return a neutral score; the rule-based RiskScorer handles actual scoring.
	return 0.5, nil
}
