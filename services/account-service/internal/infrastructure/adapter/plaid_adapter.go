package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bibbank/bib/pkg/openbanking"
)

// PlaidAdapter implements the OpenBanking port for the account service.
// It wraps the Plaid client and translates between the open banking types
// and the account service domain.
//
// In production, this adapter would use a real PlaidClient implementation.
// For development/testing, it uses a simulated stub.
type PlaidAdapter struct {
	client openbanking.PlaidClient
	config openbanking.PlaidConfig
}

// NewPlaidAdapter creates a new Plaid adapter. If client is nil, a simulated
// stub client is used for development and testing.
func NewPlaidAdapter(config openbanking.PlaidConfig, client openbanking.PlaidClient) *PlaidAdapter {
	if client == nil {
		client = &stubPlaidClient{config: config}
	}
	return &PlaidAdapter{
		client: client,
		config: config,
	}
}

// CreateLinkToken generates a Plaid Link token for the given user.
func (a *PlaidAdapter) CreateLinkToken(ctx context.Context, userID string) (openbanking.LinkTokenResponse, error) {
	if userID == "" {
		return openbanking.LinkTokenResponse{}, fmt.Errorf("user ID is required")
	}
	return a.client.CreateLinkToken(ctx, userID, a.config.Products)
}

// LinkAccount exchanges the public token from the Link flow and retrieves
// the linked accounts.
func (a *PlaidAdapter) LinkAccount(ctx context.Context, publicToken string) (openbanking.ItemAccessResponse, error) {
	if publicToken == "" {
		return openbanking.ItemAccessResponse{}, fmt.Errorf("public token is required")
	}
	return a.client.ExchangePublicToken(ctx, publicToken)
}

// GetAccounts retrieves the accounts for a linked item.
func (a *PlaidAdapter) GetAccounts(ctx context.Context, accessToken string) ([]openbanking.BankAccount, error) {
	return a.client.GetAccounts(ctx, accessToken)
}

// GetBalances retrieves the current balances for linked accounts.
func (a *PlaidAdapter) GetBalances(ctx context.Context, accessToken string) ([]openbanking.BankAccount, error) {
	return a.client.GetBalances(ctx, accessToken)
}

// SyncTransactions performs incremental transaction synchronization.
func (a *PlaidAdapter) SyncTransactions(ctx context.Context, accessToken, cursor string) (openbanking.TransactionSyncResult, error) {
	return a.client.SyncTransactions(ctx, accessToken, cursor)
}

// UnlinkAccount removes a linked item.
func (a *PlaidAdapter) UnlinkAccount(ctx context.Context, accessToken string) error {
	return a.client.RemoveItem(ctx, accessToken)
}

// ---------------------------------------------------------------------------
// Stub Plaid client for development/testing
// ---------------------------------------------------------------------------

type stubPlaidClient struct {
	config openbanking.PlaidConfig
}

func (s *stubPlaidClient) CreateLinkToken(_ context.Context, userID string, _ []string) (openbanking.LinkTokenResponse, error) {
	token := fmt.Sprintf("link-%s-%s", userID, hashShort(userID))
	return openbanking.LinkTokenResponse{
		LinkToken:  token,
		Expiration: time.Now().Add(30 * time.Minute),
		RequestID:  fmt.Sprintf("req-%s", hashShort(token)),
	}, nil
}

func (s *stubPlaidClient) ExchangePublicToken(_ context.Context, publicToken string) (openbanking.ItemAccessResponse, error) {
	accessToken := fmt.Sprintf("access-%s", hashShort(publicToken))
	return openbanking.ItemAccessResponse{
		AccessToken:   accessToken,
		ItemID:        fmt.Sprintf("item-%s", hashShort(publicToken)),
		InstitutionID: "ins_1",
		Accounts: []openbanking.BankAccount{
			{
				AccountID:       fmt.Sprintf("acct-%s-1", hashShort(publicToken)),
				InstitutionID:   "ins_1",
				InstitutionName: "First National Bank (Sandbox)",
				Name:            "Plaid Checking",
				Type:            openbanking.AccountTypeChecking,
				Subtype:         openbanking.SubtypePersonalChecking,
				Mask:            "0000",
				Currency:        "USD",
				Balances: openbanking.AccountBalances{
					Available:   "1000.00",
					Current:     "1100.00",
					Currency:    "USD",
					LastUpdated: time.Now().UTC(),
				},
			},
		},
	}, nil
}

func (s *stubPlaidClient) GetAccounts(_ context.Context, accessToken string) ([]openbanking.BankAccount, error) {
	return []openbanking.BankAccount{
		{
			AccountID:       fmt.Sprintf("acct-%s-1", hashShort(accessToken)),
			InstitutionID:   "ins_1",
			InstitutionName: "First National Bank (Sandbox)",
			Name:            "Plaid Checking",
			Type:            openbanking.AccountTypeChecking,
			Mask:            "0000",
			Currency:        "USD",
			Balances: openbanking.AccountBalances{
				Available:   "1000.00",
				Current:     "1100.00",
				Currency:    "USD",
				LastUpdated: time.Now().UTC(),
			},
		},
	}, nil
}

func (s *stubPlaidClient) GetBalances(_ context.Context, accessToken string) ([]openbanking.BankAccount, error) {
	return s.GetAccounts(context.TODO(), accessToken)
}

func (s *stubPlaidClient) SyncTransactions(_ context.Context, _ string, _ string) (openbanking.TransactionSyncResult, error) {
	now := time.Now().UTC()
	return openbanking.TransactionSyncResult{
		Added: []openbanking.Transaction{
			{
				TransactionID:  "txn-stub-001",
				Amount:         "12.50",
				Currency:       "USD",
				Date:           now.AddDate(0, 0, -1),
				Name:           "Coffee Shop",
				MerchantName:   "Blue Bottle Coffee",
				Category:       []string{"Food and Drink", "Coffee Shop"},
				Pending:        false,
				PaymentChannel: "in_store",
			},
		},
		NextCursor: "cursor-next",
		HasMore:    false,
	}, nil
}

func (s *stubPlaidClient) GetTransactions(_ context.Context, _ string, _, _ string) ([]openbanking.Transaction, error) {
	now := time.Now().UTC()
	return []openbanking.Transaction{
		{
			TransactionID:  "txn-stub-001",
			Amount:         "12.50",
			Currency:       "USD",
			Date:           now.AddDate(0, 0, -1),
			Name:           "Coffee Shop",
			MerchantName:   "Blue Bottle Coffee",
			Category:       []string{"Food and Drink", "Coffee Shop"},
			PaymentChannel: "in_store",
		},
	}, nil
}

func (s *stubPlaidClient) RemoveItem(_ context.Context, _ string) error {
	return nil
}

// hashShort returns the first 8 hex characters of a SHA-256 hash.
func hashShort(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:4])
}
