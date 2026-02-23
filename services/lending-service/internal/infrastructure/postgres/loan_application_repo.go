package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/lending-service/internal/domain/model"
	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

// LoanApplicationRepo implements port.LoanApplicationRepository.
type LoanApplicationRepo struct {
	pool *pgxpool.Pool
}

// NewLoanApplicationRepo creates a new repository backed by PostgreSQL.
func NewLoanApplicationRepo(pool *pgxpool.Pool) *LoanApplicationRepo {
	return &LoanApplicationRepo{pool: pool}
}

// Save persists a loan application (upsert by ID with optimistic locking).
func (r *LoanApplicationRepo) Save(ctx context.Context, app model.LoanApplication) error {
	query := `
		INSERT INTO loan_applications (
			id, tenant_id, applicant_id, requested_amount, currency,
			term_months, purpose, status, decision_reason, credit_score,
			version, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			status          = EXCLUDED.status,
			decision_reason = EXCLUDED.decision_reason,
			credit_score    = EXCLUDED.credit_score,
			version         = loan_applications.version + 1,
			updated_at      = EXCLUDED.updated_at
		WHERE loan_applications.version = $11
	`
	tag, err := r.pool.Exec(ctx, query,
		app.ID(), app.TenantID(), app.ApplicantID(),
		app.RequestedAmount(), app.Currency(),
		app.TermMonths(), app.Purpose(),
		app.Status().String(), app.DecisionReason(), app.CreditScore(),
		app.Version(), app.CreatedAt(), app.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("save loan application: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("optimistic locking conflict on loan application")
	}
	return nil
}

// FindByID retrieves a single loan application.
func (r *LoanApplicationRepo) FindByID(ctx context.Context, tenantID, id string) (model.LoanApplication, error) {
	query := `
		SELECT id, tenant_id, applicant_id, requested_amount, currency,
		       term_months, purpose, status, decision_reason, credit_score,
		       version, created_at, updated_at
		FROM loan_applications
		WHERE tenant_id = $1 AND id = $2
	`
	return r.scanOne(ctx, query, tenantID, id)
}

// FindByApplicantID retrieves all applications for a given applicant.
func (r *LoanApplicationRepo) FindByApplicantID(ctx context.Context, tenantID, applicantID string) ([]model.LoanApplication, error) {
	query := `
		SELECT id, tenant_id, applicant_id, requested_amount, currency,
		       term_months, purpose, status, decision_reason, credit_score,
		       version, created_at, updated_at
		FROM loan_applications
		WHERE tenant_id = $1 AND applicant_id = $2
		ORDER BY created_at DESC
	`
	return r.scanMany(ctx, query, tenantID, applicantID)
}

// ---------------------------------------------------------------------------
// scan helpers
// ---------------------------------------------------------------------------

type scannable interface {
	Scan(dest ...any) error
}

func (r *LoanApplicationRepo) scanOne(ctx context.Context, query string, args ...any) (model.LoanApplication, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return scanApplication(row)
}

func (r *LoanApplicationRepo) scanMany(ctx context.Context, query string, args ...any) ([]model.LoanApplication, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query loan applications: %w", err)
	}
	defer rows.Close()

	var result []model.LoanApplication
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, app)
	}
	return result, rows.Err()
}

func scanApplication(s scannable) (model.LoanApplication, error) {
	var (
		id, tenantID, applicantID string
		requestedAmount           decimal.Decimal
		currency                  string
		termMonths                int
		purpose, statusStr        string
		decisionReason            string
		creditScore               string
		version                   int
		createdAt, updatedAt      time.Time
	)

	err := s.Scan(
		&id, &tenantID, &applicantID,
		&requestedAmount, &currency,
		&termMonths, &purpose,
		&statusStr, &decisionReason, &creditScore,
		&version, &createdAt, &updatedAt,
	)
	if err != nil {
		return model.LoanApplication{}, fmt.Errorf("scan loan application: %w", err)
	}

	status, err := valueobject.NewLoanApplicationStatus(statusStr)
	if err != nil {
		return model.LoanApplication{}, fmt.Errorf("parse status: %w", err)
	}

	return model.ReconstructLoanApplication(
		id, tenantID, applicantID,
		requestedAmount, currency,
		termMonths, purpose,
		status, decisionReason, creditScore,
		version, createdAt, updatedAt,
	), nil
}
