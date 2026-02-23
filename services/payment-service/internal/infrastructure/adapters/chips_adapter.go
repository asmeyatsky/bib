package adapters

import (
	"context"
	"log/slog"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
	"github.com/google/uuid"
)

var _ port.RailAdapter = (*CHIPSAdapter)(nil)

// CHIPSAdapter implements the RailAdapter for CHIPS (Clearing House Interbank Payments System).
// CHIPS is a large-value USD payment system operated by The Clearing House,
// primarily used for wholesale and interbank transfers.
type CHIPSAdapter struct {
	logger *slog.Logger
}

func NewCHIPSAdapter(logger *slog.Logger) *CHIPSAdapter {
	return &CHIPSAdapter{logger: logger}
}

func (a *CHIPSAdapter) Submit(_ context.Context, order model.PaymentOrder) error {
	a.logger.Info("CHIPS: submitting large-value USD payment",
		"order_id", order.ID(),
		"amount", order.Amount(),
		"currency", order.Currency(),
	)
	// Stub: in production, this would submit a payment message to CHIPS
	// via the participant's connection to The Clearing House.
	//
	// CHIPS processes ~$1.8 trillion in payments daily with same-day settlement.
	// Messages use a proprietary format but are migrating to ISO 20022.
	return nil
}

func (a *CHIPSAdapter) GetStatus(_ context.Context, _ uuid.UUID) (valueobject.PaymentStatus, string, error) {
	// Stub: CHIPS provides same-day settlement with final netting at end of day.
	return valueobject.PaymentStatusProcessing, "pending CHIPS netting cycle", nil
}
