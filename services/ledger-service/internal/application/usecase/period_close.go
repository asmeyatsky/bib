package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/event"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/port"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// PeriodClose closes a fiscal period, preventing further postings.
type PeriodClose struct {
	periodRepo port.FiscalPeriodRepository
	publisher  port.EventPublisher
}

func NewPeriodClose(periodRepo port.FiscalPeriodRepository, publisher port.EventPublisher) *PeriodClose {
	return &PeriodClose{
		periodRepo: periodRepo,
		publisher:  publisher,
	}
}

func (uc *PeriodClose) Execute(ctx context.Context, req dto.PeriodCloseRequest) error {
	period, err := valueobject.NewFiscalPeriod(req.Year, time.Month(req.Month))
	if err != nil {
		return fmt.Errorf("invalid fiscal period: %w", err)
	}

	// Check current status
	status, err := uc.periodRepo.GetPeriodStatus(ctx, req.TenantID, period)
	if err != nil {
		return fmt.Errorf("failed to get period status: %w", err)
	}
	if status == valueobject.PeriodStatusClosed {
		return fmt.Errorf("period %s is already closed", period)
	}

	// Close the period
	if err := uc.periodRepo.ClosePeriod(ctx, req.TenantID, period); err != nil {
		return fmt.Errorf("failed to close period: %w", err)
	}

	// Publish event
	evt := event.NewPeriodClosed(req.TenantID, period.String())
	if err := uc.publisher.Publish(ctx, TopicLedgerEntries, evt); err != nil {
		return fmt.Errorf("failed to publish period closed event: %w", err)
	}

	return nil
}
