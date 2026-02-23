package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/payment-service/internal/domain/model"
	"github.com/bibbank/bib/services/payment-service/internal/domain/port"
	"github.com/bibbank/bib/services/payment-service/internal/domain/valueobject"
)

// Compile-time interface check.
var _ port.PaymentOrderRepository = (*PaymentOrderRepo)(nil)

// PaymentOrderRepo implements PaymentOrderRepository using PostgreSQL.
type PaymentOrderRepo struct {
	pool *pgxpool.Pool
}

func NewPaymentOrderRepo(pool *pgxpool.Pool) *PaymentOrderRepo {
	return &PaymentOrderRepo{pool: pool}
}

func (r *PaymentOrderRepo) Save(ctx context.Context, order model.PaymentOrder) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var destAcctID *uuid.UUID
	if order.DestinationAccountID() != uuid.Nil {
		id := order.DestinationAccountID()
		destAcctID = &id
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO payment_orders (
			id, tenant_id, source_account_id, destination_account_id,
			amount, currency, rail, status,
			routing_number, external_account_number,
			reference, description, failure_reason,
			initiated_at, settled_at, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			failure_reason = EXCLUDED.failure_reason,
			settled_at = EXCLUDED.settled_at,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`,
		order.ID(), order.TenantID(), order.SourceAccountID(), destAcctID,
		order.Amount(), order.Currency(), order.Rail().String(), order.Status().String(),
		order.RoutingInfo().RoutingNumber(), order.RoutingInfo().ExternalAccountNumber(),
		order.Reference(), order.Description(), order.FailureReason(),
		order.InitiatedAt(), order.SettledAt(), order.Version(), order.CreatedAt(), order.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("upsert payment order: %w", err)
	}

	// Write domain events to outbox.
	for _, evt := range order.DomainEvents() {
		payload, merr := json.Marshal(evt)
		if merr != nil {
			return fmt.Errorf("marshal outbox event: %w", merr)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO outbox (id, aggregate_id, aggregate_type, event_type, payload, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, evt.EventID(), evt.AggregateID(), evt.AggregateType(), evt.EventType(), payload, evt.OccurredAt())
		if err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PaymentOrderRepo) FindByID(ctx context.Context, id uuid.UUID) (model.PaymentOrder, error) {
	var (
		orderID        uuid.UUID
		tenantID       uuid.UUID
		sourceAcctID   uuid.UUID
		destAcctID     *uuid.UUID
		amount         decimal.Decimal
		currency       string
		railStr        string
		statusStr      string
		routingNumber  string
		extAcctNumber  string
		reference      string
		description    string
		failureReason  string
		initiatedAt    time.Time
		settledAt      *time.Time
		version        int
		createdAt      time.Time
		updatedAt      time.Time
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, source_account_id, destination_account_id,
			amount, currency, rail, status,
			routing_number, external_account_number,
			reference, description, failure_reason,
			initiated_at, settled_at, version, created_at, updated_at
		FROM payment_orders WHERE id = $1
	`, id).Scan(
		&orderID, &tenantID, &sourceAcctID, &destAcctID,
		&amount, &currency, &railStr, &statusStr,
		&routingNumber, &extAcctNumber,
		&reference, &description, &failureReason,
		&initiatedAt, &settledAt, &version, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.PaymentOrder{}, fmt.Errorf("payment order %s not found", id)
		}
		return model.PaymentOrder{}, fmt.Errorf("query payment order: %w", err)
	}

	rail, _ := valueobject.NewPaymentRail(railStr)
	status, _ := valueobject.NewPaymentStatus(statusStr)
	routingInfo, _ := valueobject.NewRoutingInfo(routingNumber, extAcctNumber)

	var destinationAccountID uuid.UUID
	if destAcctID != nil {
		destinationAccountID = *destAcctID
	}

	return model.Reconstruct(
		orderID, tenantID, sourceAcctID, destinationAccountID,
		amount, currency, rail, status, routingInfo,
		reference, description, failureReason,
		initiatedAt, settledAt, version, createdAt, updatedAt,
	), nil
}

func (r *PaymentOrderRepo) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment_orders
		WHERE source_account_id = $1 OR destination_account_id = $1
	`, accountID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count payment orders: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id FROM payment_orders
		WHERE source_account_id = $1 OR destination_account_id = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query payment orders: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan payment order id: %w", err)
		}
		ids = append(ids, id)
	}

	var orders []model.PaymentOrder
	for _, id := range ids {
		order, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}

	return orders, total, nil
}

func (r *PaymentOrderRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.PaymentOrder, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM payment_orders WHERE tenant_id = $1
	`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count payment orders: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id FROM payment_orders
		WHERE tenant_id = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query payment orders: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan payment order id: %w", err)
		}
		ids = append(ids, id)
	}

	var orders []model.PaymentOrder
	for _, id := range ids {
		order, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}

	return orders, total, nil
}
