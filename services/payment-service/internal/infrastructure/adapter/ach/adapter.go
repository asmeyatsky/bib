package ach

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// Compile-time interface check.
var _ port.RailAdapter = (*Adapter)(nil)

// Adapter is a stub ACH rail adapter that simulates ACH payment submission.
// In production, this would integrate with an ACH processor (e.g., Moov, Synapse, or direct Fed).
type Adapter struct {
	logger        *slog.Logger
	simulateDelay time.Duration
}

// NewAdapter creates a new stub ACH adapter.
func NewAdapter(logger *slog.Logger) *Adapter {
	return &Adapter{
		logger:        logger,
		simulateDelay: 100 * time.Millisecond,
	}
}

// Submit simulates submitting a payment to the ACH network.
// In this stub, it logs the submission and returns success after a short delay.
func (a *Adapter) Submit(ctx context.Context, order model.PaymentOrder) error {
	a.logger.Info("ACH: submitting payment",
		"payment_id", order.ID(),
		"amount", order.Amount().String(),
		"currency", order.Currency(),
		"routing_number", order.RoutingInfo().RoutingNumber(),
		"external_account", order.RoutingInfo().ExternalAccountNumber(),
	)

	// Simulate ACH processing delay.
	select {
	case <-time.After(a.simulateDelay):
		// Simulated success.
	case <-ctx.Done():
		return ctx.Err()
	}

	a.logger.Info("ACH: payment submitted successfully",
		"payment_id", order.ID(),
	)

	return nil
}

// GetStatus queries the current status of a payment in the ACH network.
// This stub always returns SETTLED for simplicity.
func (a *Adapter) GetStatus(_ context.Context, orderID uuid.UUID) (valueobject.PaymentStatus, string, error) {
	a.logger.Info("ACH: checking payment status",
		"payment_id", orderID,
	)
	return valueobject.PaymentStatusSettled, "", nil
}
