package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/model"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

// ReportSubmissionRepo is the PostgreSQL implementation of ReportSubmissionRepository.
type ReportSubmissionRepo struct {
	pool *pgxpool.Pool
}

// NewReportSubmissionRepo creates a new ReportSubmissionRepo.
func NewReportSubmissionRepo(pool *pgxpool.Pool) *ReportSubmissionRepo {
	return &ReportSubmissionRepo{pool: pool}
}

// Save persists a report submission. It uses upsert to handle both create and update.
func (r *ReportSubmissionRepo) Save(ctx context.Context, submission model.ReportSubmission) error {
	validationErrorsJSON, err := json.Marshal(submission.ValidationErrors())
	if err != nil {
		return fmt.Errorf("failed to marshal validation errors: %w", err)
	}

	query := `
		INSERT INTO report_submissions (
			id, tenant_id, report_type, reporting_period, status,
			xbrl_content, generated_at, submitted_at, validation_errors,
			version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			xbrl_content = EXCLUDED.xbrl_content,
			generated_at = EXCLUDED.generated_at,
			submitted_at = EXCLUDED.submitted_at,
			validation_errors = EXCLUDED.validation_errors,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.pool.Exec(ctx, query,
		submission.ID(),
		submission.TenantID(),
		submission.ReportType().String(),
		submission.ReportingPeriod(),
		submission.Status().String(),
		submission.XBRLContent(),
		submission.GeneratedAt(),
		submission.SubmittedAt(),
		validationErrorsJSON,
		submission.Version(),
		submission.CreatedAt(),
		submission.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to save report submission: %w", err)
	}

	return nil
}

// FindByID retrieves a report submission by its ID.
func (r *ReportSubmissionRepo) FindByID(ctx context.Context, id uuid.UUID) (model.ReportSubmission, error) {
	query := `
		SELECT id, tenant_id, report_type, reporting_period, status,
			xbrl_content, generated_at, submitted_at, validation_errors,
			version, created_at, updated_at
		FROM report_submissions
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	return scanReportSubmission(row)
}

// FindByTenantAndPeriod retrieves report submissions for a given tenant and period.
func (r *ReportSubmissionRepo) FindByTenantAndPeriod(ctx context.Context, tenantID uuid.UUID, period string) ([]model.ReportSubmission, error) {
	query := `
		SELECT id, tenant_id, report_type, reporting_period, status,
			xbrl_content, generated_at, submitted_at, validation_errors,
			version, created_at, updated_at
		FROM report_submissions
		WHERE tenant_id = $1 AND reporting_period = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to query report submissions: %w", err)
	}
	defer rows.Close()

	return scanReportSubmissions(rows)
}

// FindByTenantAndType retrieves report submissions for a given tenant and type.
func (r *ReportSubmissionRepo) FindByTenantAndType(ctx context.Context, tenantID uuid.UUID, reportType string) ([]model.ReportSubmission, error) {
	query := `
		SELECT id, tenant_id, report_type, reporting_period, status,
			xbrl_content, generated_at, submitted_at, validation_errors,
			version, created_at, updated_at
		FROM report_submissions
		WHERE tenant_id = $1 AND report_type = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, reportType)
	if err != nil {
		return nil, fmt.Errorf("failed to query report submissions: %w", err)
	}
	defer rows.Close()

	return scanReportSubmissions(rows)
}

func scanReportSubmission(row pgx.Row) (model.ReportSubmission, error) {
	var (
		id              uuid.UUID
		tenantID        uuid.UUID
		reportTypeStr   string
		reportingPeriod string
		statusStr       string
		xbrlContent     string
		generatedAt     *time.Time
		submittedAt     *time.Time
		validationJSON  []byte
		version         int
		createdAt       time.Time
		updatedAt       time.Time
	)

	err := row.Scan(
		&id, &tenantID, &reportTypeStr, &reportingPeriod, &statusStr,
		&xbrlContent, &generatedAt, &submittedAt, &validationJSON,
		&version, &createdAt, &updatedAt,
	)
	if err != nil {
		return model.ReportSubmission{}, fmt.Errorf("failed to scan report submission: %w", err)
	}

	reportType, err := valueobject.NewReportType(reportTypeStr)
	if err != nil {
		return model.ReportSubmission{}, fmt.Errorf("invalid report type in database: %w", err)
	}

	status, err := valueobject.NewSubmissionStatus(statusStr)
	if err != nil {
		return model.ReportSubmission{}, fmt.Errorf("invalid status in database: %w", err)
	}

	var validationErrors []string
	if err := json.Unmarshal(validationJSON, &validationErrors); err != nil {
		return model.ReportSubmission{}, fmt.Errorf("failed to unmarshal validation errors: %w", err)
	}

	return model.Reconstruct(
		id, tenantID, reportType, reportingPeriod, status,
		xbrlContent, generatedAt, submittedAt, validationErrors,
		version, createdAt, updatedAt,
	), nil
}

func scanReportSubmissions(rows pgx.Rows) ([]model.ReportSubmission, error) {
	var submissions []model.ReportSubmission
	for rows.Next() {
		var (
			id              uuid.UUID
			tenantID        uuid.UUID
			reportTypeStr   string
			reportingPeriod string
			statusStr       string
			xbrlContent     string
			generatedAt     *time.Time
			submittedAt     *time.Time
			validationJSON  []byte
			version         int
			createdAt       time.Time
			updatedAt       time.Time
		)

		err := rows.Scan(
			&id, &tenantID, &reportTypeStr, &reportingPeriod, &statusStr,
			&xbrlContent, &generatedAt, &submittedAt, &validationJSON,
			&version, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report submission row: %w", err)
		}

		reportType, err := valueobject.NewReportType(reportTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid report type in database: %w", err)
		}

		status, err := valueobject.NewSubmissionStatus(statusStr)
		if err != nil {
			return nil, fmt.Errorf("invalid status in database: %w", err)
		}

		var validationErrors []string
		if err := json.Unmarshal(validationJSON, &validationErrors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal validation errors: %w", err)
		}

		submission := model.Reconstruct(
			id, tenantID, reportType, reportingPeriod, status,
			xbrlContent, generatedAt, submittedAt, validationErrors,
			version, createdAt, updatedAt,
		)
		submissions = append(submissions, submission)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return submissions, nil
}
