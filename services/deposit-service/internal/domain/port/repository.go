package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/deposit-service/internal/domain/model"
)

// DepositProductRepository defines persistence operations for deposit products.
type DepositProductRepository interface {
	// Save persists a deposit product (insert or update).
	Save(ctx context.Context, product model.DepositProduct) error
	// FindByID retrieves a deposit product by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.DepositProduct, error)
	// ListByTenant returns all deposit products for a given tenant.
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.DepositProduct, error)
}

// DepositPositionRepository defines persistence operations for deposit positions.
type DepositPositionRepository interface {
	// Save persists a deposit position (insert or update).
	Save(ctx context.Context, position model.DepositPosition) error
	// FindByID retrieves a deposit position by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.DepositPosition, error)
	// FindActiveByTenant returns all active deposit positions for a given tenant.
	FindActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.DepositPosition, error)
	// FindByAccount returns all deposit positions for a given account.
	FindByAccount(ctx context.Context, accountID uuid.UUID) ([]model.DepositPosition, error)
}

// CampaignRepository defines persistence operations for deposit campaigns.
type CampaignRepository interface {
	// Save persists a campaign (insert or update).
	Save(ctx context.Context, campaign model.Campaign) error
	// FindByID retrieves a campaign by its unique identifier.
	FindByID(ctx context.Context, id uuid.UUID) (model.Campaign, error)
	// FindActiveByProduct returns active campaigns for a given product.
	FindActiveByProduct(ctx context.Context, productID uuid.UUID) ([]model.Campaign, error)
	// ListByTenant returns all campaigns for a given tenant.
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.Campaign, error)
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, events ...events.DomainEvent) error
}
