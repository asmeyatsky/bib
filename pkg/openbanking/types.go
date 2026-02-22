// Package openbanking provides data types and client interfaces for open
// banking integrations (Plaid, bank account linking, transaction syncing).
package openbanking

import "time"

// AccountType represents the type of external bank account.
type AccountType string

const (
	AccountTypeChecking    AccountType = "CHECKING"
	AccountTypeSavings     AccountType = "SAVINGS"
	AccountTypeCreditCard  AccountType = "CREDIT_CARD"
	AccountTypeLoan        AccountType = "LOAN"
	AccountTypeInvestment  AccountType = "INVESTMENT"
	AccountTypeOther       AccountType = "OTHER"
)

// AccountSubtype provides additional classification of the account.
type AccountSubtype string

const (
	SubtypePersonalChecking AccountSubtype = "PERSONAL_CHECKING"
	SubtypeBusinessChecking AccountSubtype = "BUSINESS_CHECKING"
	SubtypeCD              AccountSubtype = "CD"
	SubtypeMoneyMarket     AccountSubtype = "MONEY_MARKET"
	SubtypePaypal          AccountSubtype = "PAYPAL"
	SubtypePrepaid         AccountSubtype = "PREPAID"
)

// BankAccount represents an external bank account linked via open banking.
type BankAccount struct {
	// AccountID is the provider-assigned unique identifier.
	AccountID string
	// InstitutionID identifies the financial institution (e.g. Plaid institution ID).
	InstitutionID string
	// InstitutionName is the human-readable institution name.
	InstitutionName string
	// Name is the account name (e.g. "Plaid Checking").
	Name string
	// OfficialName is the official institution name for this account.
	OfficialName string
	// Type is the account type.
	Type AccountType
	// Subtype provides additional classification.
	Subtype AccountSubtype
	// Mask is the last 4 digits of the account number.
	Mask string
	// Currency is the ISO 4217 currency code.
	Currency string
	// Balances contains the account balance information.
	Balances AccountBalances
}

// AccountBalances represents balance information for an external account.
type AccountBalances struct {
	// Available is the amount available for spending/withdrawal.
	Available string // decimal string
	// Current is the current account balance.
	Current string // decimal string
	// Limit is the credit limit (for credit accounts).
	Limit string // decimal string
	// Currency is the ISO 4217 currency code.
	Currency string
	// LastUpdated is when the balance was last refreshed.
	LastUpdated time.Time
}

// Transaction represents a single transaction from an external account.
type Transaction struct {
	// TransactionID is the provider-assigned transaction identifier.
	TransactionID string
	// AccountID identifies the account this transaction belongs to.
	AccountID string
	// Amount is the transaction amount (positive = debit, negative = credit).
	Amount string // decimal string
	// Currency is the ISO 4217 currency code.
	Currency string
	// Date is the transaction date.
	Date time.Time
	// AuthorizedDate is when the transaction was authorized (may differ from Date).
	AuthorizedDate *time.Time
	// Name is the merchant or counterparty name.
	Name string
	// MerchantName is the cleaned-up merchant name.
	MerchantName string
	// Category holds the transaction categories (e.g. ["Food", "Restaurant"]).
	Category []string
	// Pending indicates if the transaction is still pending.
	Pending bool
	// PaymentChannel indicates how the transaction was made (online, in_store, etc.).
	PaymentChannel string
}

// TransactionSyncResult represents the result of a transaction sync operation.
type TransactionSyncResult struct {
	// Added contains newly discovered transactions.
	Added []Transaction
	// Modified contains transactions that have been updated.
	Modified []Transaction
	// Removed contains IDs of transactions that have been removed.
	Removed []string
	// NextCursor is the pagination cursor for the next sync call.
	NextCursor string
	// HasMore indicates if there are more transactions to sync.
	HasMore bool
}

// LinkTokenResponse is returned when creating a link token for account linking.
type LinkTokenResponse struct {
	// LinkToken is the token used to initialize the link flow.
	LinkToken string
	// Expiration is when the link token expires.
	Expiration time.Time
	// RequestID is the provider's request identifier.
	RequestID string
}

// ItemAccessResponse is returned after completing the link flow.
type ItemAccessResponse struct {
	// AccessToken is the persistent token for accessing the linked item.
	AccessToken string
	// ItemID identifies the linked item at the provider.
	ItemID string
	// InstitutionID identifies the financial institution.
	InstitutionID string
	// Accounts lists the accounts available under this item.
	Accounts []BankAccount
}
