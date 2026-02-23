package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fraud-service/internal/domain/model"
	"github.com/bibbank/bib/services/fraud-service/internal/domain/valueobject"
)

// AssessmentRepository implements port.AssessmentRepository using PostgreSQL.
type AssessmentRepository struct {
	pool *pgxpool.Pool
}

// NewAssessmentRepository creates a new PostgreSQL-backed assessment repository.
func NewAssessmentRepository(pool *pgxpool.Pool) *AssessmentRepository {
	return &AssessmentRepository{pool: pool}
}

// Save persists a transaction assessment and its risk signals.
func (r *AssessmentRepository) Save(ctx context.Context, assessment *model.TransactionAssessment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Upsert the assessment.
	query := `
		INSERT INTO transaction_assessments (
			id, tenant_id, transaction_id, account_id,
			amount, currency, transaction_type,
			risk_level, risk_score, decision,
			assessed_at, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id, transaction_id) DO UPDATE SET
			risk_level = EXCLUDED.risk_level,
			risk_score = EXCLUDED.risk_score,
			decision = EXCLUDED.decision,
			assessed_at = EXCLUDED.assessed_at,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`

	_, err = tx.Exec(ctx, query,
		assessment.ID(),
		assessment.TenantID(),
		assessment.TransactionID(),
		assessment.AccountID(),
		assessment.Amount(),
		assessment.Currency(),
		assessment.TransactionType(),
		assessment.RiskLevel().String(),
		assessment.RiskScore(),
		assessment.Decision().String(),
		assessment.AssessedAt(),
		assessment.Version(),
		assessment.CreatedAt(),
		assessment.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save assessment: %w", err)
	}

	// Delete existing signals and insert fresh ones.
	_, err = tx.Exec(ctx, `DELETE FROM risk_signals WHERE assessment_id = $1`, assessment.ID())
	if err != nil {
		return fmt.Errorf("failed to delete old risk signals: %w", err)
	}

	for _, signal := range assessment.RiskSignals() {
		_, err = tx.Exec(ctx,
			`INSERT INTO risk_signals (assessment_id, tenant_id, signal) VALUES ($1, $2, $3)`,
			assessment.ID(), assessment.TenantID(), signal,
		)
		if err != nil {
			return fmt.Errorf("failed to save risk signal: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID retrieves an assessment by its unique identifier.
func (r *AssessmentRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.TransactionAssessment, error) {
	query := `
		SELECT id, tenant_id, transaction_id, account_id,
			amount, currency, transaction_type,
			risk_level, risk_score, decision,
			assessed_at, version, created_at, updated_at
		FROM transaction_assessments
		WHERE tenant_id = $1 AND id = $2
	`

	assessment, err := r.scanAssessment(ctx, r.pool.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return nil, err
	}

	return assessment, nil
}

// FindByTransactionID retrieves an assessment by the original transaction ID.
func (r *AssessmentRepository) FindByTransactionID(ctx context.Context, tenantID, transactionID uuid.UUID) (*model.TransactionAssessment, error) {
	query := `
		SELECT id, tenant_id, transaction_id, account_id,
			amount, currency, transaction_type,
			risk_level, risk_score, decision,
			assessed_at, version, created_at, updated_at
		FROM transaction_assessments
		WHERE tenant_id = $1 AND transaction_id = $2
	`

	assessment, err := r.scanAssessment(ctx, r.pool.QueryRow(ctx, query, tenantID, transactionID))
	if err != nil {
		return nil, err
	}

	return assessment, nil
}

// FindByAccountID retrieves all assessments for a given account.
func (r *AssessmentRepository) FindByAccountID(ctx context.Context, tenantID, accountID uuid.UUID, limit, offset int) ([]*model.TransactionAssessment, error) {
	query := `
		SELECT id, tenant_id, transaction_id, account_id,
			amount, currency, transaction_type,
			risk_level, risk_score, decision,
			assessed_at, version, created_at, updated_at
		FROM transaction_assessments
		WHERE tenant_id = $1 AND account_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, tenantID, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query assessments: %w", err)
	}
	defer rows.Close()

	var assessments []*model.TransactionAssessment
	for rows.Next() {
		assessment, err := r.scanAssessmentFromRows(ctx, rows)
		if err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}

	return assessments, nil
}

func (r *AssessmentRepository) scanAssessment(ctx context.Context, row pgx.Row) (*model.TransactionAssessment, error) {
	var (
		id              uuid.UUID
		tenantID        uuid.UUID
		transactionID   uuid.UUID
		accountID       uuid.UUID
		amount          decimal.Decimal
		currency        string
		transactionType string
		riskLevelStr    string
		riskScore       int
		decisionStr     string
		assessedAt      *time.Time
		version         int
		createdAt       time.Time
		updatedAt       time.Time
	)

	err := row.Scan(
		&id, &tenantID, &transactionID, &accountID,
		&amount, &currency, &transactionType,
		&riskLevelStr, &riskScore, &decisionStr,
		&assessedAt, &version, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan assessment: %w", err)
	}

	riskLevel, err := valueobject.RiskLevelFromString(riskLevelStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse risk level: %w", err)
	}

	decision, err := valueobject.AssessmentDecisionFromString(decisionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decision: %w", err)
	}

	// Load risk signals.
	signals, err := r.loadSignals(ctx, id)
	if err != nil {
		return nil, err
	}

	var assessedAtVal time.Time
	if assessedAt != nil {
		assessedAtVal = *assessedAt
	}

	return model.Reconstruct(
		id, tenantID, transactionID, accountID,
		amount, currency, transactionType,
		riskLevel, riskScore, decision, signals,
		assessedAtVal, version, createdAt, updatedAt,
	), nil
}

func (r *AssessmentRepository) scanAssessmentFromRows(ctx context.Context, rows pgx.Rows) (*model.TransactionAssessment, error) {
	var (
		id              uuid.UUID
		tenantID        uuid.UUID
		transactionID   uuid.UUID
		accountID       uuid.UUID
		amount          decimal.Decimal
		currency        string
		transactionType string
		riskLevelStr    string
		riskScore       int
		decisionStr     string
		assessedAt      *time.Time
		version         int
		createdAt       time.Time
		updatedAt       time.Time
	)

	err := rows.Scan(
		&id, &tenantID, &transactionID, &accountID,
		&amount, &currency, &transactionType,
		&riskLevelStr, &riskScore, &decisionStr,
		&assessedAt, &version, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan assessment row: %w", err)
	}

	riskLevel, err := valueobject.RiskLevelFromString(riskLevelStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse risk level: %w", err)
	}

	decision, err := valueobject.AssessmentDecisionFromString(decisionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decision: %w", err)
	}

	signals, err := r.loadSignals(ctx, id)
	if err != nil {
		return nil, err
	}

	var assessedAtVal time.Time
	if assessedAt != nil {
		assessedAtVal = *assessedAt
	}

	return model.Reconstruct(
		id, tenantID, transactionID, accountID,
		amount, currency, transactionType,
		riskLevel, riskScore, decision, signals,
		assessedAtVal, version, createdAt, updatedAt,
	), nil
}

func (r *AssessmentRepository) loadSignals(ctx context.Context, assessmentID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT signal FROM risk_signals WHERE assessment_id = $1 ORDER BY created_at`,
		assessmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk signals: %w", err)
	}
	defer rows.Close()

	var signals []string
	for rows.Next() {
		var signal string
		if err := rows.Scan(&signal); err != nil {
			return nil, fmt.Errorf("failed to scan risk signal: %w", err)
		}
		signals = append(signals, signal)
	}

	if signals == nil {
		signals = make([]string, 0)
	}

	return signals, nil
}
