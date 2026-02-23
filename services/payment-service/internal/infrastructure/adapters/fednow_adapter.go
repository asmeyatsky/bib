package adapters

import (
	"context"
	"log/slog"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
	"github.com/google/uuid"
)

var _ port.RailAdapter = (*FedNowAdapter)(nil)

// FedNowAdapter implements the RailAdapter for FedNow/RTP instant payments.
type FedNowAdapter struct {
	logger *slog.Logger
}

func NewFedNowAdapter(logger *slog.Logger) *FedNowAdapter {
	return &FedNowAdapter{logger: logger}
}

func (a *FedNowAdapter) Submit(_ context.Context, order model.PaymentOrder) error {
	a.logger.Info("FedNow: submitting instant payment",
		"order_id", order.ID(),
		"amount", order.Amount(),
		"currency", order.Currency(),
	)
	// Stub: in production, this would connect to the Federal Reserve's FedNow Service
	// via ISO 20022 messaging (pacs.008 for credit transfers).
	return nil
}

func (a *FedNowAdapter) GetStatus(_ context.Context, _ uuid.UUID) (valueobject.PaymentStatus, string, error) {
	// Stub: FedNow provides near-instant confirmation, return settled.
	return valueobject.PaymentStatusSettled, "", nil
}
