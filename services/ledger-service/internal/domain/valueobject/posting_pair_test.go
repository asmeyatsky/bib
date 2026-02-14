package valueobject_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func TestNewPostingPair_Valid(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(100)

	pp, err := valueobject.NewPostingPair(debit, credit, amount, "USD", "test posting")
	require.NoError(t, err)

	assert.True(t, pp.DebitAccount().Equal(debit))
	assert.True(t, pp.CreditAccount().Equal(credit))
	assert.True(t, pp.Amount().Equal(amount))
	assert.Equal(t, "USD", pp.Currency())
	assert.Equal(t, "test posting", pp.Description())
}

func TestNewPostingPair_String(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(500)

	pp, err := valueobject.NewPostingPair(debit, credit, amount, "EUR", "payment")
	require.NoError(t, err)

	str := pp.String()
	assert.Contains(t, str, "DR 1000")
	assert.Contains(t, str, "CR 2000")
	assert.Contains(t, str, "500")
	assert.Contains(t, str, "EUR")
}

func TestNewPostingPair_ZeroDebitAccount(t *testing.T) {
	var zeroAccount valueobject.AccountCode
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(100)

	_, err := valueobject.NewPostingPair(zeroAccount, credit, amount, "USD", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "debit account code is required")
}

func TestNewPostingPair_ZeroCreditAccount(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	var zeroAccount valueobject.AccountCode
	amount := decimal.NewFromInt(100)

	_, err := valueobject.NewPostingPair(debit, zeroAccount, amount, "USD", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credit account code is required")
}

func TestNewPostingPair_SameAccounts(t *testing.T) {
	account := valueobject.MustAccountCode("1000")
	amount := decimal.NewFromInt(100)

	_, err := valueobject.NewPostingPair(account, account, amount, "USD", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "debit and credit accounts must be different")
}

func TestNewPostingPair_ZeroAmount(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")

	_, err := valueobject.NewPostingPair(debit, credit, decimal.Zero, "USD", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "posting amount must be positive")
}

func TestNewPostingPair_NegativeAmount(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(-50)

	_, err := valueobject.NewPostingPair(debit, credit, amount, "USD", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "posting amount must be positive")
}

func TestNewPostingPair_EmptyCurrency(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(100)

	_, err := valueobject.NewPostingPair(debit, credit, amount, "", "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currency is required")
}

func TestNewPostingPair_EmptyDescription(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount := decimal.NewFromInt(100)

	// Empty description is allowed
	pp, err := valueobject.NewPostingPair(debit, credit, amount, "USD", "")
	require.NoError(t, err)
	assert.Equal(t, "", pp.Description())
}

func TestNewPostingPair_DecimalAmount(t *testing.T) {
	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	amount, _ := decimal.NewFromString("99.99")

	pp, err := valueobject.NewPostingPair(debit, credit, amount, "USD", "fractional")
	require.NoError(t, err)
	assert.True(t, pp.Amount().Equal(amount))
}
