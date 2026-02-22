package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/account-service/internal/domain/model"
	"github.com/bibbank/bib/services/account-service/internal/domain/valueobject"
)

func newTestHolder(t *testing.T) model.AccountHolder {
	t.Helper()
	holder, err := model.NewAccountHolder(
		uuid.New(),
		"John",
		"Doe",
		"john.doe@example.com",
		uuid.New(),
	)
	require.NoError(t, err)
	return holder
}

func newTestAccount(t *testing.T) model.CustomerAccount {
	t.Helper()
	account, err := model.NewCustomerAccount(
		uuid.New(),
		valueobject.AccountTypeChecking,
		"USD",
		newTestHolder(t),
	)
	require.NoError(t, err)
	return account
}

func TestNewCustomerAccount(t *testing.T) {
	t.Run("creates account in PENDING status", func(t *testing.T) {
		tenantID := uuid.New()
		holder := newTestHolder(t)

		account, err := model.NewCustomerAccount(tenantID, valueobject.AccountTypeChecking, "USD", holder)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, account.ID())
		assert.Equal(t, tenantID, account.TenantID())
		assert.False(t, account.AccountNumber().IsZero())
		assert.True(t, account.AccountType().Equal(valueobject.AccountTypeChecking))
		assert.Equal(t, model.AccountStatusPending, account.Status())
		assert.Equal(t, "USD", account.Currency())
		assert.Equal(t, holder.FirstName(), account.Holder().FirstName())
		assert.Equal(t, 1, account.Version())
		assert.False(t, account.CreatedAt().IsZero())
		assert.False(t, account.UpdatedAt().IsZero())
	})

	t.Run("emits AccountOpened event", func(t *testing.T) {
		account := newTestAccount(t)
		events := account.DomainEvents()

		require.Len(t, events, 1)
		assert.Equal(t, "account.opened", events[0].EventType())
		assert.Equal(t, account.ID().String(), events[0].AggregateID())
		assert.Equal(t, "CustomerAccount", events[0].AggregateType())
	})

	t.Run("rejects nil tenant ID", func(t *testing.T) {
		_, err := model.NewCustomerAccount(uuid.Nil, valueobject.AccountTypeChecking, "USD", newTestHolder(t))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant ID")
	})

	t.Run("rejects zero account type", func(t *testing.T) {
		_, err := model.NewCustomerAccount(uuid.New(), valueobject.AccountType{}, "USD", newTestHolder(t))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account type")
	})

	t.Run("rejects empty currency", func(t *testing.T) {
		_, err := model.NewCustomerAccount(uuid.New(), valueobject.AccountTypeChecking, "", newTestHolder(t))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency")
	})

	t.Run("rejects invalid currency length", func(t *testing.T) {
		_, err := model.NewCustomerAccount(uuid.New(), valueobject.AccountTypeChecking, "US", newTestHolder(t))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "3-letter ISO code")
	})
}

func TestCustomerAccount_Activate(t *testing.T) {
	t.Run("activates PENDING account", func(t *testing.T) {
		account := newTestAccount(t)
		now := time.Now()

		activated, err := account.Activate(now)
		require.NoError(t, err)

		assert.Equal(t, model.AccountStatusActive, activated.Status())
		assert.Equal(t, now, activated.UpdatedAt())
		assert.Equal(t, account.Version()+1, activated.Version())
	})

	t.Run("emits AccountActivated event", func(t *testing.T) {
		account := newTestAccount(t).ClearDomainEvents()
		now := time.Now()

		activated, err := account.Activate(now)
		require.NoError(t, err)

		events := activated.DomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "account.activated", events[0].EventType())
	})

	t.Run("original account is unchanged (immutability)", func(t *testing.T) {
		account := newTestAccount(t)
		originalStatus := account.Status()
		originalVersion := account.Version()

		_, err := account.Activate(time.Now())
		require.NoError(t, err)

		assert.Equal(t, originalStatus, account.Status())
		assert.Equal(t, originalVersion, account.Version())
	})

	t.Run("rejects activation from ACTIVE status", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())

		_, err := activated.Activate(time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ACTIVE")
	})

	t.Run("rejects activation from FROZEN status", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		frozen, _ := activated.Freeze("test reason", time.Now())

		_, err := frozen.Activate(time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FROZEN")
	})

	t.Run("rejects activation from CLOSED status", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		closed, _ := activated.Close("test reason", time.Now())

		_, err := closed.Activate(time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CLOSED")
	})
}

func TestCustomerAccount_Freeze(t *testing.T) {
	t.Run("freezes ACTIVE account", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		now := time.Now()

		frozen, err := activated.Freeze("suspicious activity", now)
		require.NoError(t, err)

		assert.Equal(t, model.AccountStatusFrozen, frozen.Status())
		assert.Equal(t, now, frozen.UpdatedAt())
		assert.Equal(t, activated.Version()+1, frozen.Version())
	})

	t.Run("emits AccountFrozen event", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		activated = activated.ClearDomainEvents()

		frozen, err := activated.Freeze("fraud detected", time.Now())
		require.NoError(t, err)

		events := frozen.DomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "account.frozen", events[0].EventType())
	})

	t.Run("rejects freeze from PENDING status", func(t *testing.T) {
		account := newTestAccount(t)
		_, err := account.Freeze("test", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PENDING")
	})

	t.Run("rejects freeze from FROZEN status", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		frozen, _ := activated.Freeze("reason", time.Now())

		_, err := frozen.Freeze("another reason", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FROZEN")
	})

	t.Run("rejects freeze without reason", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())

		_, err := activated.Freeze("", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reason")
	})
}

func TestCustomerAccount_Unfreeze(t *testing.T) {
	t.Run("unfreezes FROZEN account", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		frozen, _ := activated.Freeze("test reason", time.Now())
		now := time.Now()

		unfrozen, err := frozen.Unfreeze(now)
		require.NoError(t, err)

		assert.Equal(t, model.AccountStatusActive, unfrozen.Status())
		assert.Equal(t, now, unfrozen.UpdatedAt())
		assert.Equal(t, frozen.Version()+1, unfrozen.Version())
	})

	t.Run("emits AccountUnfrozen event", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		frozen, _ := activated.Freeze("test", time.Now())
		frozen = frozen.ClearDomainEvents()

		unfrozen, err := frozen.Unfreeze(time.Now())
		require.NoError(t, err)

		events := unfrozen.DomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "account.unfrozen", events[0].EventType())
	})

	t.Run("rejects unfreeze from ACTIVE status", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())

		_, err := activated.Unfreeze(time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ACTIVE")
	})

	t.Run("rejects unfreeze from PENDING status", func(t *testing.T) {
		account := newTestAccount(t)
		_, err := account.Unfreeze(time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PENDING")
	})
}

func TestCustomerAccount_Close(t *testing.T) {
	t.Run("closes ACTIVE account", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		now := time.Now()

		closed, err := activated.Close("customer request", now)
		require.NoError(t, err)

		assert.Equal(t, model.AccountStatusClosed, closed.Status())
		assert.Equal(t, now, closed.UpdatedAt())
		assert.Equal(t, activated.Version()+1, closed.Version())
	})

	t.Run("closes FROZEN account", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		frozen, _ := activated.Freeze("fraud", time.Now())
		now := time.Now()

		closed, err := frozen.Close("compliance decision", now)
		require.NoError(t, err)

		assert.Equal(t, model.AccountStatusClosed, closed.Status())
	})

	t.Run("emits AccountClosed event", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		activated = activated.ClearDomainEvents()

		closed, err := activated.Close("closing reason", time.Now())
		require.NoError(t, err)

		events := closed.DomainEvents()
		require.Len(t, events, 1)
		assert.Equal(t, "account.closed", events[0].EventType())
	})

	t.Run("rejects close from PENDING status", func(t *testing.T) {
		account := newTestAccount(t)
		_, err := account.Close("reason", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PENDING")
	})

	t.Run("rejects close from CLOSED status", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())
		closed, _ := activated.Close("reason", time.Now())

		_, err := closed.Close("another reason", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CLOSED")
	})

	t.Run("rejects close without reason", func(t *testing.T) {
		account := newTestAccount(t)
		activated, _ := account.Activate(time.Now())

		_, err := activated.Close("", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reason")
	})
}

func TestCustomerAccount_AssignLedgerCode(t *testing.T) {
	t.Run("assigns ledger code", func(t *testing.T) {
		account := newTestAccount(t)

		updated, err := account.AssignLedgerCode("2000-123", time.Now())
		require.NoError(t, err)
		assert.Equal(t, "2000-123", updated.LedgerAccountCode())
	})

	t.Run("rejects empty ledger code", func(t *testing.T) {
		account := newTestAccount(t)
		_, err := account.AssignLedgerCode("", time.Now())
		assert.Error(t, err)
	})

	t.Run("rejects reassignment", func(t *testing.T) {
		account := newTestAccount(t)
		updated, _ := account.AssignLedgerCode("2000-123", time.Now())

		_, err := updated.AssignLedgerCode("2000-456", time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned")
	})
}

func TestCustomerAccount_DomainEvents(t *testing.T) {
	t.Run("returns defensive copy", func(t *testing.T) {
		account := newTestAccount(t)
		events1 := account.DomainEvents()
		events2 := account.DomainEvents()

		require.Len(t, events1, 1)
		require.Len(t, events2, 1)

		// Modifying one slice should not affect the other.
		events1[0] = nil
		assert.NotNil(t, events2[0])
	})

	t.Run("ClearDomainEvents returns account with no events", func(t *testing.T) {
		account := newTestAccount(t)
		require.Len(t, account.DomainEvents(), 1)

		cleared := account.ClearDomainEvents()
		assert.Len(t, cleared.DomainEvents(), 0)

		// Original should still have events.
		assert.Len(t, account.DomainEvents(), 1)
	})

	t.Run("events accumulate across transitions", func(t *testing.T) {
		account := newTestAccount(t) // 1 event: AccountOpened
		activated, _ := account.Activate(time.Now())

		// The activated account should have its own events (not accumulated from account)
		// because clone creates a copy of events. But Activate appends to the cloned events.
		events := activated.DomainEvents()
		require.Len(t, events, 2) // AccountOpened + AccountActivated
	})
}

func TestCustomerAccount_FullLifecycle(t *testing.T) {
	t.Run("PENDING -> ACTIVE -> FROZEN -> ACTIVE -> CLOSED", func(t *testing.T) {
		// Open account (PENDING).
		account := newTestAccount(t)
		assert.Equal(t, model.AccountStatusPending, account.Status())

		// Activate.
		activated, err := account.Activate(time.Now())
		require.NoError(t, err)
		assert.Equal(t, model.AccountStatusActive, activated.Status())

		// Freeze.
		frozen, err := activated.Freeze("suspicious activity", time.Now())
		require.NoError(t, err)
		assert.Equal(t, model.AccountStatusFrozen, frozen.Status())

		// Unfreeze.
		unfrozen, err := frozen.Unfreeze(time.Now())
		require.NoError(t, err)
		assert.Equal(t, model.AccountStatusActive, unfrozen.Status())

		// Close.
		closed, err := unfrozen.Close("customer request", time.Now())
		require.NoError(t, err)
		assert.Equal(t, model.AccountStatusClosed, closed.Status())

		// Verify version incremented correctly.
		assert.Equal(t, 5, closed.Version())
	})
}
