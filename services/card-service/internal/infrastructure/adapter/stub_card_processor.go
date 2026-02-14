package adapter

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/card-service/internal/domain/model"
)

// StubCardProcessor is a stub implementation of the CardProcessorAdapter port.
// It simulates interactions with an external card processor (e.g., Marqeta, Adyen).
// In production, this would be replaced with a real HTTP/gRPC client.
type StubCardProcessor struct {
	logger *slog.Logger
}

// NewStubCardProcessor creates a new StubCardProcessor.
func NewStubCardProcessor(logger *slog.Logger) *StubCardProcessor {
	return &StubCardProcessor{
		logger: logger,
	}
}

// IssuePhysicalCard simulates requesting a physical card from the processor.
func (p *StubCardProcessor) IssuePhysicalCard(ctx context.Context, card model.Card) error {
	p.logger.Info("stub: issuing physical card",
		slog.String("card_id", card.ID().String()),
		slog.String("tenant_id", card.TenantID().String()),
		slog.String("card_type", card.CardType().String()),
		slog.String("last_four", card.CardNumber().LastFour()),
	)
	return nil
}

// GetCardDetails simulates retrieving card details from the processor.
func (p *StubCardProcessor) GetCardDetails(ctx context.Context, cardID uuid.UUID) error {
	p.logger.Info("stub: getting card details",
		slog.String("card_id", cardID.String()),
	)
	return nil
}
