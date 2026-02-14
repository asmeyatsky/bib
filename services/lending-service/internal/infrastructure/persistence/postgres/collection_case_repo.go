package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

// CollectionCaseRepo implements port.CollectionCaseRepository.
type CollectionCaseRepo struct {
	pool *pgxpool.Pool
}

// NewCollectionCaseRepo creates a new PostgreSQL-backed collection case repository.
func NewCollectionCaseRepo(pool *pgxpool.Pool) *CollectionCaseRepo {
	return &CollectionCaseRepo{pool: pool}
}

// Save persists a collection case (upsert).
func (r *CollectionCaseRepo) Save(ctx context.Context, c model.CollectionCase) error {
	notesJSON, err := json.Marshal(c.Notes())
	if err != nil {
		return fmt.Errorf("marshal notes: %w", err)
	}

	query := `
		INSERT INTO collection_cases (id, loan_id, tenant_id, status, assigned_to, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status      = EXCLUDED.status,
			assigned_to = EXCLUDED.assigned_to,
			notes       = EXCLUDED.notes,
			updated_at  = EXCLUDED.updated_at
	`
	tag, err := r.pool.Exec(ctx, query,
		c.ID(), c.LoanID(), c.TenantID(),
		c.Status().String(), c.AssignedTo(),
		notesJSON, c.CreatedAt(), c.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("save collection case: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("failed to save collection case")
	}
	return nil
}

// FindByID retrieves a collection case by ID.
func (r *CollectionCaseRepo) FindByID(ctx context.Context, tenantID, id string) (model.CollectionCase, error) {
	query := `
		SELECT id, loan_id, tenant_id, status, assigned_to, notes, created_at, updated_at
		FROM collection_cases
		WHERE tenant_id = $1 AND id = $2
	`
	row := r.pool.QueryRow(ctx, query, tenantID, id)
	return scanCollectionCase(row)
}

// FindByLoanID retrieves all collection cases for a loan.
func (r *CollectionCaseRepo) FindByLoanID(ctx context.Context, tenantID, loanID string) ([]model.CollectionCase, error) {
	query := `
		SELECT id, loan_id, tenant_id, status, assigned_to, notes, created_at, updated_at
		FROM collection_cases
		WHERE tenant_id = $1 AND loan_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, loanID)
	if err != nil {
		return nil, fmt.Errorf("query collection cases: %w", err)
	}
	defer rows.Close()

	var result []model.CollectionCase
	for rows.Next() {
		c, err := scanCollectionCase(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func scanCollectionCase(s scannable) (model.CollectionCase, error) {
	var (
		id, loanID, tenantID string
		statusStr, assignedTo string
		notesJSON             []byte
		createdAt, updatedAt  time.Time
	)

	err := s.Scan(&id, &loanID, &tenantID, &statusStr, &assignedTo, &notesJSON, &createdAt, &updatedAt)
	if err != nil {
		return model.CollectionCase{}, fmt.Errorf("scan collection case: %w", err)
	}

	status, err := valueobject.NewCollectionCaseStatus(statusStr)
	if err != nil {
		return model.CollectionCase{}, fmt.Errorf("parse collection case status: %w", err)
	}

	var notes []string
	if err := json.Unmarshal(notesJSON, &notes); err != nil {
		return model.CollectionCase{}, fmt.Errorf("unmarshal notes: %w", err)
	}

	return model.ReconstructCollectionCase(
		id, loanID, tenantID, status, assignedTo, notes, createdAt, updatedAt,
	), nil
}
