package openbanking

import "context"

// PlaidClient defines the interface for Plaid API operations.
// Implementations may be real HTTP clients or stubs for testing.
type PlaidClient interface {
	// CreateLinkToken generates a link token for initializing the Plaid Link
	// flow. The user ID is used to associate the link session with a customer.
	CreateLinkToken(ctx context.Context, userID string, products []string) (LinkTokenResponse, error)

	// ExchangePublicToken exchanges a public token (from the link flow) for
	// a persistent access token.
	ExchangePublicToken(ctx context.Context, publicToken string) (ItemAccessResponse, error)

	// GetAccounts retrieves the accounts associated with an access token.
	GetAccounts(ctx context.Context, accessToken string) ([]BankAccount, error)

	// GetBalances retrieves the current balances for accounts under an access token.
	GetBalances(ctx context.Context, accessToken string) ([]BankAccount, error)

	// SyncTransactions performs incremental transaction synchronization.
	// Pass an empty cursor for the initial sync.
	SyncTransactions(ctx context.Context, accessToken string, cursor string) (TransactionSyncResult, error)

	// GetTransactions retrieves transactions for a date range.
	GetTransactions(ctx context.Context, accessToken string, startDate, endDate string) ([]Transaction, error)

	// RemoveItem removes a linked item and invalidates its access token.
	RemoveItem(ctx context.Context, accessToken string) error
}

// PlaidConfig holds configuration for the Plaid client.
type PlaidConfig struct {
	ClientID     string
	Secret       string
	Environment  string
	BaseURL      string
	WebhookURL   string
	Language     string
	Products     []string
	CountryCodes []string
}

// DefaultPlaidConfig returns configuration defaults for the Plaid sandbox.
func DefaultPlaidConfig() PlaidConfig {
	return PlaidConfig{
		ClientID:     "",
		Secret:       "",
		Environment:  "sandbox",
		BaseURL:      "https://sandbox.plaid.com",
		Products:     []string{"transactions", "auth"},
		CountryCodes: []string{"US"},
		Language:     "en",
	}
}
