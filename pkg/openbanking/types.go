// Package openbanking provides data types and client interfaces for open
// banking integrations (Plaid, bank account linking, transaction syncing).
package openbanking

import "time"

// AccountType represents the type of external bank account.
type AccountType string

const (
	AccountTypeChecking   AccountType = "CHECKING"
	AccountTypeSavings    AccountType = "SAVINGS"
	AccountTypeCreditCard AccountType = "CREDIT_CARD"
	AccountTypeLoan       AccountType = "LOAN"
	AccountTypeInvestment AccountType = "INVESTMENT"
	AccountTypeOther      AccountType = "OTHER"
)

// AccountSubtype provides additional classification of the account.
type AccountSubtype string

const (
	SubtypePersonalChecking AccountSubtype = "PERSONAL_CHECKING"
	SubtypeBusinessChecking AccountSubtype = "BUSINESS_CHECKING"
	SubtypeCD               AccountSubtype = "CD"
	SubtypeMoneyMarket      AccountSubtype = "MONEY_MARKET"
	SubtypePaypal           AccountSubtype = "PAYPAL"
	SubtypePrepaid          AccountSubtype = "PREPAID"
)

// BankAccount represents an external bank account linked via open banking.
type BankAccount struct {
	Balances        AccountBalances
	AccountID       string
	InstitutionID   string
	InstitutionName string
	Name            string
	OfficialName    string
	Type            AccountType
	Subtype         AccountSubtype
	Mask            string
	Currency        string
}

// AccountBalances represents balance information for an external account.
type AccountBalances struct {
	LastUpdated time.Time
	Available   string
	Current     string
	Limit       string
	Currency    string
}

// Transaction represents a single transaction from an external account.
type Transaction struct {
	Date           time.Time
	AuthorizedDate *time.Time
	TransactionID  string
	AccountID      string
	Amount         string
	Currency       string
	Name           string
	MerchantName   string
	PaymentChannel string
	Category       []string
	Pending        bool
}

// TransactionSyncResult represents the result of a transaction sync operation.
type TransactionSyncResult struct {
	NextCursor string
	Added      []Transaction
	Modified   []Transaction
	Removed    []string
	HasMore    bool
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
