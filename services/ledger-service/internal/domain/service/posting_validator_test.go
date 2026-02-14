package service_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/ledger-service/internal/domain/service"
	"github.com/bibbank/bib/services/ledger-service/internal/domain/valueobject"
)

func TestPostingValidator_ValidatePostings_Valid(t *testing.T) {
	validator := service.NewPostingValidator()

	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	pp, err := valueobject.NewPostingPair(debit, credit, decimal.NewFromInt(100), "USD", "test")
	require.NoError(t, err)

	err = validator.ValidatePostings([]valueobject.PostingPair{pp})
	assert.NoError(t, err)
}

func TestPostingValidator_ValidatePostings_MultiplePostings(t *testing.T) {
	validator := service.NewPostingValidator()

	debit1 := valueobject.MustAccountCode("1000")
	credit1 := valueobject.MustAccountCode("2000")
	pp1, err := valueobject.NewPostingPair(debit1, credit1, decimal.NewFromInt(100), "USD", "posting 1")
	require.NoError(t, err)

	debit2 := valueobject.MustAccountCode("3000")
	credit2 := valueobject.MustAccountCode("4000")
	pp2, err := valueobject.NewPostingPair(debit2, credit2, decimal.NewFromInt(200), "USD", "posting 2")
	require.NoError(t, err)

	err = validator.ValidatePostings([]valueobject.PostingPair{pp1, pp2})
	assert.NoError(t, err)
}

func TestPostingValidator_ValidatePostings_MultipleCurrencies(t *testing.T) {
	validator := service.NewPostingValidator()

	debit1 := valueobject.MustAccountCode("1000")
	credit1 := valueobject.MustAccountCode("2000")
	pp1, err := valueobject.NewPostingPair(debit1, credit1, decimal.NewFromInt(100), "USD", "USD posting")
	require.NoError(t, err)

	debit2 := valueobject.MustAccountCode("3000")
	credit2 := valueobject.MustAccountCode("4000")
	pp2, err := valueobject.NewPostingPair(debit2, credit2, decimal.NewFromInt(85), "EUR", "EUR posting")
	require.NoError(t, err)

	err = validator.ValidatePostings([]valueobject.PostingPair{pp1, pp2})
	assert.NoError(t, err)
}

func TestPostingValidator_ValidatePostings_EmptyPostings(t *testing.T) {
	validator := service.NewPostingValidator()

	err := validator.ValidatePostings(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one posting pair is required")

	err = validator.ValidatePostings([]valueobject.PostingPair{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one posting pair is required")
}

func TestPostingValidator_ValidateNotSelfPosting_Valid(t *testing.T) {
	validator := service.NewPostingValidator()

	debit := valueobject.MustAccountCode("1000")
	credit := valueobject.MustAccountCode("2000")
	pp, err := valueobject.NewPostingPair(debit, credit, decimal.NewFromInt(100), "USD", "test")
	require.NoError(t, err)

	err = validator.ValidateNotSelfPosting([]valueobject.PostingPair{pp})
	assert.NoError(t, err)
}

func TestPostingValidator_ValidateNotSelfPosting_EmptySlice(t *testing.T) {
	validator := service.NewPostingValidator()

	// Empty slice should pass (no self-postings to find)
	err := validator.ValidateNotSelfPosting([]valueobject.PostingPair{})
	assert.NoError(t, err)
}

func TestPostingValidator_ValidateNotSelfPosting_MultipleValid(t *testing.T) {
	validator := service.NewPostingValidator()

	pp1, err := valueobject.NewPostingPair(
		valueobject.MustAccountCode("1000"),
		valueobject.MustAccountCode("2000"),
		decimal.NewFromInt(100), "USD", "posting 1",
	)
	require.NoError(t, err)

	pp2, err := valueobject.NewPostingPair(
		valueobject.MustAccountCode("3000"),
		valueobject.MustAccountCode("4000"),
		decimal.NewFromInt(200), "USD", "posting 2",
	)
	require.NoError(t, err)

	err = validator.ValidateNotSelfPosting([]valueobject.PostingPair{pp1, pp2})
	assert.NoError(t, err)
}
