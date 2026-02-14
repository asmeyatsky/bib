package adapters

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

var _ port.RailAdapter = (*ACHAdapter)(nil)

// ACHAdapter implements the RailAdapter for ACH payments.
type ACHAdapter struct {
	logger *slog.Logger
}

func NewACHAdapter(logger *slog.Logger) *ACHAdapter {
	return &ACHAdapter{logger: logger}
}

func (a *ACHAdapter) Submit(ctx context.Context, order model.PaymentOrder) error {
	a.logger.Info("ACH: submitting payment",
		"order_id", order.ID(),
		"amount", order.Amount(),
		"routing_number", order.RoutingInfo().RoutingNumber(),
	)
	// Stub: in production, this would call an ACH processor API (e.g., Moov, Dwolla, Column)
	return nil
}

func (a *ACHAdapter) GetStatus(ctx context.Context, orderID uuid.UUID) (valueobject.PaymentStatus, string, error) {
	// Stub: return settled
	return valueobject.PaymentStatusSettled, "", nil
}
