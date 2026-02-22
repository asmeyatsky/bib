package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/card-service/internal/domain/model"
	"github.com/bibbank/bib/services/card-service/internal/domain/valueobject"
)

// PostgresCardRepository implements the CardRepository port using PostgreSQL.
type PostgresCardRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCardRepository creates a new PostgresCardRepository.
func NewPostgresCardRepository(pool *pgxpool.Pool) *PostgresCardRepository {
	return &PostgresCardRepository{pool: pool}
}

// Save persists a new card aggregate.
func (r *PostgresCardRepository) Save(ctx context.Context, card model.Card) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO cards (
			id, tenant_id, account_id, card_type, status,
			last_four, expiry_month, expiry_year, currency,
			daily_limit, monthly_limit, daily_spent, monthly_spent,
			version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err = tx.Exec(ctx, query,
		card.ID(),
		card.TenantID(),
		card.AccountID(),
		card.CardType().String(),
		card.Status().String(),
		card.CardNumber().LastFour(),
		card.CardNumber().ExpiryMonth(),
		card.CardNumber().ExpiryYear(),
		card.Currency(),
		card.DailyLimit(),
		card.MonthlyLimit(),
		card.DailySpent(),
		card.MonthlySpent(),
		card.Version(),
		card.CreatedAt(),
		card.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert card: %w", err)
	}

	// Write domain events to the outbox within the same transaction.
	if err := r.writeOutbox(ctx, tx, card); err != nil {
		return fmt.Errorf("failed to write outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Update persists changes to an existing card aggregate with optimistic locking.
func (r *PostgresCardRepository) Update(ctx context.Context, card model.Card) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE cards SET
			status = $1,
			daily_spent = $2,
			monthly_spent = $3,
			version = $4,
			updated_at = $5
		WHERE id = $6 AND version = $7
	`

	result, err := tx.Exec(ctx, query,
		card.Status().String(),
		card.DailySpent(),
		card.MonthlySpent(),
		card.Version(),
		card.UpdatedAt(),
		card.ID(),
		card.Version()-1, // Optimistic concurrency: expect previous version.
	)
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("optimistic locking failure: card %s has been modified by another process", card.ID())
	}

	// Write domain events to the outbox within the same transaction.
	if err := r.writeOutbox(ctx, tx, card); err != nil {
		return fmt.Errorf("failed to write outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID retrieves a card by its unique identifier.
func (r *PostgresCardRepository) FindByID(ctx context.Context, id uuid.UUID) (model.Card, error) {
	query := `
		SELECT id, tenant_id, account_id, card_type, status,
			   last_four, expiry_month, expiry_year, currency,
			   daily_limit, monthly_limit, daily_spent, monthly_spent,
			   version, created_at, updated_at
		FROM cards WHERE id = $1
	`

	return r.scanCard(r.pool.QueryRow(ctx, query, id))
}

// FindByAccountID retrieves all cards belonging to an account.
func (r *PostgresCardRepository) FindByAccountID(ctx context.Context, accountID uuid.UUID) ([]model.Card, error) {
	query := `
		SELECT id, tenant_id, account_id, card_type, status,
			   last_four, expiry_month, expiry_year, currency,
			   daily_limit, monthly_limit, daily_spent, monthly_spent,
			   version, created_at, updated_at
		FROM cards WHERE account_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards by account: %w", err)
	}
	defer rows.Close()

	return r.scanCards(rows)
}

// FindByTenantID retrieves all cards belonging to a tenant.
func (r *PostgresCardRepository) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.Card, error) {
	query := `
		SELECT id, tenant_id, account_id, card_type, status,
			   last_four, expiry_month, expiry_year, currency,
			   daily_limit, monthly_limit, daily_spent, monthly_spent,
			   version, created_at, updated_at
		FROM cards WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards by tenant: %w", err)
	}
	defer rows.Close()

	return r.scanCards(rows)
}

// SaveTransaction records a card transaction.
func (r *PostgresCardRepository) SaveTransaction(
	ctx context.Context,
	cardID uuid.UUID,
	amount decimal.Decimal,
	currency, merchantName, merchantCategory, authCode, status string,
) error {
	query := `
		INSERT INTO card_transactions (card_id, amount, currency, merchant_name, merchant_category, auth_code, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query, cardID, amount, currency, merchantName, merchantCategory, authCode, status)
	if err != nil {
		return fmt.Errorf("failed to insert card transaction: %w", err)
	}

	return nil
}

// scanCard scans a single row into a Card aggregate.
func (r *PostgresCardRepository) scanCard(row pgx.Row) (model.Card, error) {
	var (
		id           uuid.UUID
		tenantID     uuid.UUID
		accountID    uuid.UUID
		cardTypeStr  string
		statusStr    string
		lastFour     string
		expiryMonth  string
		expiryYear   string
		currency     string
		dailyLimit   decimal.Decimal
		monthlyLimit decimal.Decimal
		dailySpent   decimal.Decimal
		monthlySpent decimal.Decimal
		version      int
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(
		&id, &tenantID, &accountID, &cardTypeStr, &statusStr,
		&lastFour, &expiryMonth, &expiryYear, &currency,
		&dailyLimit, &monthlyLimit, &dailySpent, &monthlySpent,
		&version, &createdAt, &updatedAt,
	)
	if err != nil {
		return model.Card{}, fmt.Errorf("failed to scan card: %w", err)
	}

	cardType, err := valueobject.NewCardType(cardTypeStr)
	if err != nil {
		return model.Card{}, fmt.Errorf("invalid card type in DB: %w", err)
	}

	status, err := valueobject.NewCardStatus(statusStr)
	if err != nil {
		return model.Card{}, fmt.Errorf("invalid card status in DB: %w", err)
	}

	cardNumber, err := valueobject.NewCardNumber(lastFour, expiryMonth, expiryYear)
	if err != nil {
		return model.Card{}, fmt.Errorf("invalid card number in DB: %w", err)
	}

	return model.Reconstruct(
		id, tenantID, accountID,
		cardType, status, cardNumber,
		currency, dailyLimit, monthlyLimit,
		dailySpent, monthlySpent,
		version, createdAt, updatedAt,
	), nil
}

// scanCards scans multiple rows into a slice of Card aggregates.
func (r *PostgresCardRepository) scanCards(rows pgx.Rows) ([]model.Card, error) {
	var cards []model.Card
	for rows.Next() {
		card, err := r.scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return cards, nil
}

// writeOutbox writes domain events to the transactional outbox table within the given transaction.
func (r *PostgresCardRepository) writeOutbox(ctx context.Context, tx pgx.Tx, card model.Card) error {
	for _, evt := range card.DomainEvents() {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		query := `
			INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload)
			VALUES ($1, $2, $3, $4)
		`

		_, err = tx.Exec(ctx, query, card.ID(), "Card", evt.EventType(), payload)
		if err != nil {
			return fmt.Errorf("failed to insert outbox event: %w", err)
		}
	}
	return nil
}
