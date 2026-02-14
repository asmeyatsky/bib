package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// Compile-time interface check
var _ port.VerificationRepository = (*VerificationRepo)(nil)

// VerificationRepo implements VerificationRepository using PostgreSQL.
type VerificationRepo struct {
	pool *pgxpool.Pool
}

func NewVerificationRepo(pool *pgxpool.Pool) *VerificationRepo {
	return &VerificationRepo{pool: pool}
}

func (r *VerificationRepo) Save(ctx context.Context, v model.IdentityVerification) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert identity verification
	_, err = tx.Exec(ctx, `
		INSERT INTO identity_verifications (id, tenant_id, applicant_first_name, applicant_last_name,
			applicant_email, applicant_dob, applicant_country, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`, v.ID(), v.TenantID(), v.ApplicantFirstName(), v.ApplicantLastName(),
		v.ApplicantEmail(), v.ApplicantDOB(), v.ApplicantCountry(),
		v.Status().String(), v.Version(), v.CreatedAt(), v.UpdatedAt())
	if err != nil {
		return fmt.Errorf("upsert identity verification: %w", err)
	}

	// Delete existing checks (for upsert scenario)
	_, err = tx.Exec(ctx, `DELETE FROM verification_checks WHERE verification_id = $1`, v.ID())
	if err != nil {
		return fmt.Errorf("delete existing checks: %w", err)
	}

	// Insert checks
	for _, c := range v.Checks() {
		_, err = tx.Exec(ctx, `
			INSERT INTO verification_checks (id, verification_id, check_type, status, provider,
				provider_reference, completed_at, failure_reason)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, c.ID(), v.ID(), c.CheckType().String(), c.Status().String(),
			c.Provider(), c.ProviderReference(), c.CompletedAt(), c.FailureReason())
		if err != nil {
			return fmt.Errorf("insert check %s: %w", c.ID(), err)
		}
	}

	// Write domain events to outbox
	for _, evt := range v.DomainEvents() {
		_, err = tx.Exec(ctx, `
			INSERT INTO outbox (id, aggregate_id, aggregate_type, event_type, payload, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, evt.EventID(), evt.AggregateID(), evt.AggregateType(), evt.EventType(), evt.Payload(), evt.OccurredAt())
		if err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *VerificationRepo) FindByID(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
	var (
		vID       uuid.UUID
		tenantID  uuid.UUID
		firstName string
		lastName  string
		email     string
		dob       string
		country   string
		status    string
		version   int
		createdAt time.Time
		updatedAt time.Time
	)

	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, applicant_first_name, applicant_last_name,
			applicant_email, applicant_dob, applicant_country,
			status, version, created_at, updated_at
		FROM identity_verifications WHERE id = $1
	`, id).Scan(&vID, &tenantID, &firstName, &lastName, &email, &dob, &country,
		&status, &version, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.IdentityVerification{}, fmt.Errorf("verification %s not found", id)
		}
		return model.IdentityVerification{}, fmt.Errorf("query verification: %w", err)
	}

	// Query checks
	checks, err := r.findChecksByVerificationID(ctx, id)
	if err != nil {
		return model.IdentityVerification{}, err
	}

	verificationStatus, err := valueobject.NewVerificationStatus(status)
	if err != nil {
		return model.IdentityVerification{}, fmt.Errorf("invalid verification status in DB: %w", err)
	}

	return model.Reconstruct(
		vID, tenantID,
		firstName, lastName, email, dob, country,
		verificationStatus, checks,
		version, createdAt, updatedAt,
	), nil
}

func (r *VerificationRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.IdentityVerification, int, error) {
	// Count total
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM identity_verifications WHERE tenant_id = $1
	`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count verifications: %w", err)
	}

	// Query IDs
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM identity_verifications
		WHERE tenant_id = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query verifications: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan verification id: %w", err)
		}
		ids = append(ids, id)
	}

	var verifications []model.IdentityVerification
	for _, id := range ids {
		v, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		verifications = append(verifications, v)
	}

	return verifications, total, nil
}

func (r *VerificationRepo) findChecksByVerificationID(ctx context.Context, verificationID uuid.UUID) ([]model.VerificationCheck, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, check_type, status, provider, provider_reference, completed_at, failure_reason
		FROM verification_checks WHERE verification_id = $1
		ORDER BY id
	`, verificationID)
	if err != nil {
		return nil, fmt.Errorf("query checks: %w", err)
	}
	defer rows.Close()

	var checks []model.VerificationCheck
	for rows.Next() {
		var (
			id            uuid.UUID
			checkTypeStr  string
			statusStr     string
			provider      string
			providerRef   string
			completedAt   *time.Time
			failureReason string
		)
		if err := rows.Scan(&id, &checkTypeStr, &statusStr, &provider, &providerRef, &completedAt, &failureReason); err != nil {
			return nil, fmt.Errorf("scan check: %w", err)
		}

		checkType, err := valueobject.NewCheckType(checkTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid check type in DB: %w", err)
		}
		status, err := valueobject.NewVerificationStatus(statusStr)
		if err != nil {
			return nil, fmt.Errorf("invalid check status in DB: %w", err)
		}

		checks = append(checks, model.ReconstructCheck(id, checkType, status, provider, providerRef, completedAt, failureReason))
	}

	return checks, nil
}
