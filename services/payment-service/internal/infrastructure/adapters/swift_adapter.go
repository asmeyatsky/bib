package adapters

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

var _ port.RailAdapter = (*SWIFTAdapter)(nil)

// SWIFTAdapter implements the RailAdapter for SWIFT international payments.
type SWIFTAdapter struct {
	logger *slog.Logger
}

func NewSWIFTAdapter(logger *slog.Logger) *SWIFTAdapter {
	return &SWIFTAdapter{logger: logger}
}

func (a *SWIFTAdapter) Submit(ctx context.Context, order model.PaymentOrder) error {
	a.logger.Info("SWIFT: submitting international payment",
		"order_id", order.ID(),
		"amount", order.Amount(),
		"currency", order.Currency(),
	)
	// Stub: in production, this would construct an ISO 20022 pacs.008 message
	// and submit it via SWIFT Alliance Lite2 or Alliance Access API.
	//
	// ISO 20022 message construction placeholder:
	//   - Build CreditTransferTransactionInformation (CdtTrfTxInf)
	//   - Set InstructedAmount with currency
	//   - Set Debtor/Creditor agent BICs
	//   - Set RemittanceInformation
	//   - Submit via SWIFT gpi for tracking (UETR)
	return nil
}

func (a *SWIFTAdapter) GetStatus(ctx context.Context, orderID uuid.UUID) (valueobject.PaymentStatus, string, error) {
	// Stub: in production, this would query SWIFT gpi Tracker for payment status.
	// SWIFT payments typically settle in 1-2 business days.
	return valueobject.PaymentStatusProcessing, "awaiting correspondent bank processing", nil
}
