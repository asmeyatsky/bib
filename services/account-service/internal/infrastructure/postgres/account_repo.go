package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

// AccountRepository implements port.AccountRepository using PostgreSQL.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository creates a new PostgreSQL-backed AccountRepository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

// Save persists a CustomerAccount using an upsert with optimistic concurrency control.
// It also writes domain events to the outbox table within the same transaction.
func (r *AccountRepository) Save(ctx context.Context, account model.CustomerAccount) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert customer_accounts with optimistic locking.
	const upsertAccountSQL = `
		INSERT INTO customer_accounts (
			id, tenant_id, account_number, account_type, status,
			currency, ledger_account_code, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			ledger_account_code = EXCLUDED.ledger_account_code,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
		WHERE customer_accounts.version = EXCLUDED.version - 1
	`

	result, err := tx.Exec(ctx, upsertAccountSQL,
		account.ID(),
		account.TenantID(),
		account.AccountNumber().String(),
		account.AccountType().String(),
		string(account.Status()),
		account.Currency(),
		account.LedgerAccountCode(),
		account.Version(),
		account.CreatedAt(),
		account.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert account: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("optimistic concurrency conflict: account %s has been modified", account.ID())
	}

	// Upsert account holder.
	const upsertHolderSQL = `
		INSERT INTO account_holders (
			id, account_id, first_name, last_name, email, identity_verification_id
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (account_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			email = EXCLUDED.email,
			identity_verification_id = EXCLUDED.identity_verification_id
	`

	holder := account.Holder()
	var identityVerificationID *uuid.UUID
	if holder.IdentityVerificationID() != uuid.Nil {
		id := holder.IdentityVerificationID()
		identityVerificationID = &id
	}

	_, err = tx.Exec(ctx, upsertHolderSQL,
		holder.ID(),
		account.ID(),
		holder.FirstName(),
		holder.LastName(),
		holder.Email(),
		identityVerificationID,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert account holder: %w", err)
	}

	// Write domain events to outbox.
	for _, evt := range account.DomainEvents() {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		const insertOutboxSQL = `
			INSERT INTO outbox (aggregate_id, aggregate_type, event_type, payload)
			VALUES ($1, $2, $3, $4)
		`

		_, err = tx.Exec(ctx, insertOutboxSQL,
			account.ID(),
			"CustomerAccount",
			evt.EventType(),
			payload,
		)
		if err != nil {
			return fmt.Errorf("failed to insert outbox event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FindByID retrieves a CustomerAccount by its unique identifier.
func (r *AccountRepository) FindByID(ctx context.Context, id uuid.UUID) (model.CustomerAccount, error) {
	const query = `
		SELECT
			ca.id, ca.tenant_id, ca.account_number, ca.account_type, ca.status,
			ca.currency, ca.ledger_account_code, ca.version, ca.created_at, ca.updated_at,
			ah.id, ah.first_name, ah.last_name, ah.email, ah.identity_verification_id
		FROM customer_accounts ca
		JOIN account_holders ah ON ah.account_id = ca.id
		WHERE ca.id = $1
	`

	return r.scanAccount(ctx, query, id)
}

// FindByAccountNumber retrieves a CustomerAccount by its account number.
func (r *AccountRepository) FindByAccountNumber(ctx context.Context, number valueobject.AccountNumber) (model.CustomerAccount, error) {
	const query = `
		SELECT
			ca.id, ca.tenant_id, ca.account_number, ca.account_type, ca.status,
			ca.currency, ca.ledger_account_code, ca.version, ca.created_at, ca.updated_at,
			ah.id, ah.first_name, ah.last_name, ah.email, ah.identity_verification_id
		FROM customer_accounts ca
		JOIN account_holders ah ON ah.account_id = ca.id
		WHERE ca.account_number = $1
	`

	return r.scanAccount(ctx, query, number.String())
}

// ListByTenant retrieves all accounts for a given tenant with pagination.
func (r *AccountRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	const countQuery = `SELECT COUNT(*) FROM customer_accounts WHERE tenant_id = $1`
	const listQuery = `
		SELECT
			ca.id, ca.tenant_id, ca.account_number, ca.account_type, ca.status,
			ca.currency, ca.ledger_account_code, ca.version, ca.created_at, ca.updated_at,
			ah.id, ah.first_name, ah.last_name, ah.email, ah.identity_verification_id
		FROM customer_accounts ca
		JOIN account_holders ah ON ah.account_id = ca.id
		WHERE ca.tenant_id = $1
		ORDER BY ca.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count accounts: %w", err)
	}

	accounts, err := r.scanAccounts(ctx, listQuery, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// ListByHolder retrieves all accounts for a given holder with pagination.
func (r *AccountRepository) ListByHolder(ctx context.Context, holderID uuid.UUID, limit, offset int) ([]model.CustomerAccount, int, error) {
	const countQuery = `
		SELECT COUNT(*)
		FROM customer_accounts ca
		JOIN account_holders ah ON ah.account_id = ca.id
		WHERE ah.id = $1
	`
	const listQuery = `
		SELECT
			ca.id, ca.tenant_id, ca.account_number, ca.account_type, ca.status,
			ca.currency, ca.ledger_account_code, ca.version, ca.created_at, ca.updated_at,
			ah.id, ah.first_name, ah.last_name, ah.email, ah.identity_verification_id
		FROM customer_accounts ca
		JOIN account_holders ah ON ah.account_id = ca.id
		WHERE ah.id = $1
		ORDER BY ca.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, holderID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count accounts: %w", err)
	}

	accounts, err := r.scanAccounts(ctx, listQuery, holderID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// scanAccount scans a single account row from a query result.
func (r *AccountRepository) scanAccount(ctx context.Context, query string, args ...interface{}) (model.CustomerAccount, error) {
	row := r.pool.QueryRow(ctx, query, args...)

	var (
		id                     uuid.UUID
		tenantID               uuid.UUID
		accountNumberStr       string
		accountTypeStr         string
		statusStr              string
		currency               string
		ledgerAccountCode      string
		version                int
		createdAt              time.Time
		updatedAt              time.Time
		holderID               uuid.UUID
		firstName              string
		lastName               string
		email                  string
		identityVerificationID *uuid.UUID
	)

	err := row.Scan(
		&id, &tenantID, &accountNumberStr, &accountTypeStr, &statusStr,
		&currency, &ledgerAccountCode, &version, &createdAt, &updatedAt,
		&holderID, &firstName, &lastName, &email, &identityVerificationID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.CustomerAccount{}, fmt.Errorf("account not found")
		}
		return model.CustomerAccount{}, fmt.Errorf("failed to scan account: %w", err)
	}

	return reconstructAccount(
		id, tenantID, accountNumberStr, accountTypeStr, statusStr,
		currency, ledgerAccountCode, version, createdAt, updatedAt,
		holderID, firstName, lastName, email, identityVerificationID,
	)
}

// scanAccounts scans multiple account rows from a query result.
func (r *AccountRepository) scanAccounts(ctx context.Context, query string, args ...interface{}) ([]model.CustomerAccount, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []model.CustomerAccount
	for rows.Next() {
		var (
			id                     uuid.UUID
			tenantID               uuid.UUID
			accountNumberStr       string
			accountTypeStr         string
			statusStr              string
			currency               string
			ledgerAccountCode      string
			version                int
			createdAt              time.Time
			updatedAt              time.Time
			holderID               uuid.UUID
			firstName              string
			lastName               string
			email                  string
			identityVerificationID *uuid.UUID
		)

		err := rows.Scan(
			&id, &tenantID, &accountNumberStr, &accountTypeStr, &statusStr,
			&currency, &ledgerAccountCode, &version, &createdAt, &updatedAt,
			&holderID, &firstName, &lastName, &email, &identityVerificationID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account row: %w", err)
		}

		account, err := reconstructAccount(
			id, tenantID, accountNumberStr, accountTypeStr, statusStr,
			currency, ledgerAccountCode, version, createdAt, updatedAt,
			holderID, firstName, lastName, email, identityVerificationID,
		)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows: %w", err)
	}

	return accounts, nil
}

// reconstructAccount rebuilds a CustomerAccount aggregate from scanned database values.
func reconstructAccount(
	id, tenantID uuid.UUID,
	accountNumberStr, accountTypeStr, statusStr string,
	currency, ledgerAccountCode string,
	version int,
	createdAt, updatedAt time.Time,
	holderID uuid.UUID,
	firstName, lastName, email string,
	identityVerificationID *uuid.UUID,
) (model.CustomerAccount, error) {
	accountNumber, err := valueobject.AccountNumberFromString(accountNumberStr)
	if err != nil {
		return model.CustomerAccount{}, fmt.Errorf("invalid stored account number: %w", err)
	}

	accountType, err := valueobject.NewAccountType(accountTypeStr)
	if err != nil {
		return model.CustomerAccount{}, fmt.Errorf("invalid stored account type: %w", err)
	}

	var verificationID uuid.UUID
	if identityVerificationID != nil {
		verificationID = *identityVerificationID
	}

	holder := model.ReconstructAccountHolder(
		holderID,
		firstName,
		lastName,
		email,
		verificationID,
	)

	return model.ReconstructCustomerAccount(
		id,
		tenantID,
		accountNumber,
		accountType,
		model.AccountStatus(statusStr),
		currency,
		holder,
		ledgerAccountCode,
		version,
		createdAt,
		updatedAt,
	), nil
}
