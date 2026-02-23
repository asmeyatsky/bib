package adapters

import (
	"context"
	"log/slog"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
	"github.com/google/uuid"
)

var _ port.RailAdapter = (*SEPAAdapter)(nil)

// SEPAAdapter implements the RailAdapter for SEPA (Single Euro Payments Area) transfers.
type SEPAAdapter struct {
	logger *slog.Logger
}

func NewSEPAAdapter(logger *slog.Logger) *SEPAAdapter {
	return &SEPAAdapter{logger: logger}
}

func (a *SEPAAdapter) Submit(_ context.Context, order model.PaymentOrder) error {
	a.logger.Info("SEPA: submitting euro payment",
		"order_id", order.ID(),
		"amount", order.Amount(),
		"currency", order.Currency(),
	)
	// Stub: in production, this would construct a SEPA Credit Transfer (SCT) or
	// SEPA Instant Credit Transfer (SCT Inst) message in pain.001 XML format
	// and submit it via the bank's SEPA clearing connection (e.g., EBA STEP2, TARGET2).
	//
	// For SEPA Instant (SCT Inst):
	//   - Maximum amount: EUR 100,000 (may vary by scheme rules)
	//   - Settlement within 10 seconds
	//   - 24/7/365 availability
	return nil
}

func (a *SEPAAdapter) GetStatus(_ context.Context, _ uuid.UUID) (valueobject.PaymentStatus, string, error) {
	// Stub: SEPA Instant provides near-real-time confirmation.
	// Standard SEPA credit transfers settle by next business day.
	return valueobject.PaymentStatusSettled, "", nil
}
