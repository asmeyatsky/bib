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

// LoanRepo implements port.LoanRepository.
type LoanRepo struct {
	pool *pgxpool.Pool
}

// NewLoanRepo creates a new PostgreSQL-backed loan repository.
func NewLoanRepo(pool *pgxpool.Pool) *LoanRepo {
	return &LoanRepo{pool: pool}
}

// Save persists a loan and its amortization schedule.
func (r *LoanRepo) Save(ctx context.Context, loan model.Loan) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	loanQuery := `
		INSERT INTO loans (
			id, tenant_id, application_id, borrower_account_id,
			principal, currency, interest_rate_bps, term_months,
			status, outstanding_balance, next_payment_due,
			version, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (id) DO UPDATE SET
			status              = EXCLUDED.status,
			outstanding_balance = EXCLUDED.outstanding_balance,
			next_payment_due    = EXCLUDED.next_payment_due,
			version             = loans.version + 1,
			updated_at          = EXCLUDED.updated_at
		WHERE loans.version = $12
	`
	tag, err := tx.Exec(ctx, loanQuery,
		loan.ID(), loan.TenantID(), loan.ApplicationID(), loan.BorrowerAccountID(),
		loan.Principal(), loan.Currency(), loan.InterestRateBps(), loan.TermMonths(),
		loan.Status().String(), loan.OutstandingBalance(), loan.NextPaymentDue(),
		loan.Version(), loan.CreatedAt(), loan.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("save loan: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("optimistic locking conflict on loan")
	}

	// Save amortization schedule (only on first insert).
	if loan.Version() == 1 {
		for _, entry := range loan.Schedule() {
			entryQuery := `
				INSERT INTO amortization_entries (loan_id, period, due_date, principal, interest, total, remaining_balance)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (loan_id, period) DO NOTHING
			`
			_, err := tx.Exec(ctx, entryQuery,
				loan.ID(), entry.Period, entry.DueDate,
				entry.Principal, entry.Interest, entry.Total, entry.RemainingBalance,
			)
			if err != nil {
				return fmt.Errorf("save amortization entry %d: %w", entry.Period, err)
			}
		}
	}

	return tx.Commit(ctx)
}

// FindByID retrieves a loan and its amortization schedule by ID.
func (r *LoanRepo) FindByID(ctx context.Context, tenantID, id string) (model.Loan, error) {
	query := `
		SELECT id, tenant_id, application_id, borrower_account_id,
		       principal, currency, interest_rate_bps, term_months,
		       status, outstanding_balance, next_payment_due,
		       version, created_at, updated_at
		FROM loans
		WHERE tenant_id = $1 AND id = $2
	`
	loan, err := r.scanOneLoan(ctx, query, tenantID, id)
	if err != nil {
		return model.Loan{}, err
	}

	schedule, err := r.loadSchedule(ctx, id)
	if err != nil {
		return model.Loan{}, err
	}

	return model.ReconstructLoan(
		loan.ID(), loan.TenantID(), loan.ApplicationID(), loan.BorrowerAccountID(),
		loan.Principal(), loan.Currency(), loan.InterestRateBps(), loan.TermMonths(),
		loan.Status(), schedule, loan.OutstandingBalance(), loan.NextPaymentDue(),
		loan.Version(), loan.CreatedAt(), loan.UpdatedAt(),
	), nil
}

// FindByApplicationID retrieves a loan by its originating application.
func (r *LoanRepo) FindByApplicationID(ctx context.Context, tenantID, applicationID string) (model.Loan, error) {
	query := `
		SELECT id, tenant_id, application_id, borrower_account_id,
		       principal, currency, interest_rate_bps, term_months,
		       status, outstanding_balance, next_payment_due,
		       version, created_at, updated_at
		FROM loans
		WHERE tenant_id = $1 AND application_id = $2
	`
	loan, err := r.scanOneLoan(ctx, query, tenantID, applicationID)
	if err != nil {
		return model.Loan{}, err
	}

	schedule, err := r.loadSchedule(ctx, loan.ID())
	if err != nil {
		return model.Loan{}, err
	}

	return model.ReconstructLoan(
		loan.ID(), loan.TenantID(), loan.ApplicationID(), loan.BorrowerAccountID(),
		loan.Principal(), loan.Currency(), loan.InterestRateBps(), loan.TermMonths(),
		loan.Status(), schedule, loan.OutstandingBalance(), loan.NextPaymentDue(),
		loan.Version(), loan.CreatedAt(), loan.UpdatedAt(),
	), nil
}

// FindByBorrowerAccountID retrieves all loans for a borrower account.
func (r *LoanRepo) FindByBorrowerAccountID(ctx context.Context, tenantID, borrowerAccountID string) ([]model.Loan, error) {
	query := `
		SELECT id, tenant_id, application_id, borrower_account_id,
		       principal, currency, interest_rate_bps, term_months,
		       status, outstanding_balance, next_payment_due,
		       version, created_at, updated_at
		FROM loans
		WHERE tenant_id = $1 AND borrower_account_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, borrowerAccountID)
	if err != nil {
		return nil, fmt.Errorf("query loans: %w", err)
	}
	defer rows.Close()

	var loans []model.Loan
	for rows.Next() {
		loan, err := scanLoanRow(rows)
		if err != nil {
			return nil, err
		}
		schedule, err := r.loadSchedule(ctx, loan.ID())
		if err != nil {
			return nil, err
		}
		loans = append(loans, model.ReconstructLoan(
			loan.ID(), loan.TenantID(), loan.ApplicationID(), loan.BorrowerAccountID(),
			loan.Principal(), loan.Currency(), loan.InterestRateBps(), loan.TermMonths(),
			loan.Status(), schedule, loan.OutstandingBalance(), loan.NextPaymentDue(),
			loan.Version(), loan.CreatedAt(), loan.UpdatedAt(),
		))
	}
	return loans, rows.Err()
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

func (r *LoanRepo) scanOneLoan(ctx context.Context, query string, args ...any) (model.Loan, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return scanLoanRow(row)
}

func scanLoanRow(s scannable) (model.Loan, error) {
	var (
		id, tenantID, applicationID, borrowerAccountID string
		principal                                      decimal.Decimal
		currency                                       string
		interestRateBps, termMonths                    int
		statusStr                                      string
		outstandingBalance                             decimal.Decimal
		nextPaymentDue                                 time.Time
		version                                        int
		createdAt, updatedAt                           time.Time
	)

	err := s.Scan(
		&id, &tenantID, &applicationID, &borrowerAccountID,
		&principal, &currency, &interestRateBps, &termMonths,
		&statusStr, &outstandingBalance, &nextPaymentDue,
		&version, &createdAt, &updatedAt,
	)
	if err != nil {
		return model.Loan{}, fmt.Errorf("scan loan: %w", err)
	}

	status, err := valueobject.NewLoanStatus(statusStr)
	if err != nil {
		return model.Loan{}, fmt.Errorf("parse loan status: %w", err)
	}

	return model.ReconstructLoan(
		id, tenantID, applicationID, borrowerAccountID,
		principal, currency, interestRateBps, termMonths,
		status, nil, outstandingBalance, nextPaymentDue,
		version, createdAt, updatedAt,
	), nil
}

func (r *LoanRepo) loadSchedule(ctx context.Context, loanID string) ([]model.AmortizationEntry, error) {
	query := `
		SELECT period, due_date, principal, interest, total, remaining_balance
		FROM amortization_entries
		WHERE loan_id = $1
		ORDER BY period
	`
	rows, err := r.pool.Query(ctx, query, loanID)
	if err != nil {
		return nil, fmt.Errorf("query amortization: %w", err)
	}
	defer rows.Close()

	var schedule []model.AmortizationEntry
	for rows.Next() {
		var e model.AmortizationEntry
		if err := rows.Scan(&e.Period, &e.DueDate, &e.Principal, &e.Interest, &e.Total, &e.RemainingBalance); err != nil {
			return nil, fmt.Errorf("scan amortization entry: %w", err)
		}
		schedule = append(schedule, e)
	}
	return schedule, rows.Err()
}
