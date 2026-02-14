package model

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

// Balance represents the balance of a ledger account at a point in time.
type Balance struct {
	accountCode valueobject.AccountCode
	amount      decimal.Decimal
	currency    string
	asOf        time.Time
}

func NewBalance(accountCode valueobject.AccountCode, amount decimal.Decimal, currency string, asOf time.Time) Balance {
	return Balance{
		accountCode: accountCode,
		amount:      amount,
		currency:    currency,
		asOf:        asOf,
	}
}

func (b Balance) AccountCode() valueobject.AccountCode { return b.accountCode }
func (b Balance) Amount() decimal.Decimal              { return b.amount }
func (b Balance) Currency() string                     { return b.currency }
func (b Balance) AsOf() time.Time                      { return b.asOf }
func (b Balance) IsZero() bool                         { return b.amount.IsZero() }
