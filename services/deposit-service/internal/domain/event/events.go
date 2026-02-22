package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/pkg/events"
)

const AggregateTypeDepositPosition = "DepositPosition"

// DepositOpened is emitted when a new deposit position is opened.
type DepositOpened struct {
	events.BaseEvent
	PositionID uuid.UUID `json:"position_id"`
	AccountID  uuid.UUID `json:"account_id"`
	ProductID  uuid.UUID `json:"product_id"`
	Principal  string    `json:"principal"`
	Currency   string    `json:"currency"`
}

func NewDepositOpened(positionID, tenantID, accountID, productID uuid.UUID, principal decimal.Decimal, currency string) DepositOpened {
	return DepositOpened{
		BaseEvent:  events.NewBaseEvent("deposit.position.opened", positionID.String(), AggregateTypeDepositPosition, tenantID.String()),
		PositionID: positionID,
		AccountID:  accountID,
		ProductID:  productID,
		Principal:  principal.String(),
		Currency:   currency,
	}
}

// InterestAccrued is emitted when interest is accrued on a deposit position.
type InterestAccrued struct {
	events.BaseEvent
	PositionID uuid.UUID `json:"position_id"`
	AccountID  uuid.UUID `json:"account_id"`
	Amount     string    `json:"amount"`
	Currency   string    `json:"currency"`
	AsOf       time.Time `json:"as_of"`
}

func NewInterestAccrued(positionID, tenantID, accountID uuid.UUID, amount decimal.Decimal, currency string, asOf time.Time) InterestAccrued {
	return InterestAccrued{
		BaseEvent:  events.NewBaseEvent("deposit.interest.accrued", positionID.String(), AggregateTypeDepositPosition, tenantID.String()),
		PositionID: positionID,
		AccountID:  accountID,
		Amount:     amount.String(),
		Currency:   currency,
		AsOf:       asOf,
	}
}

// DepositMatured is emitted when a term deposit reaches maturity.
type DepositMatured struct {
	events.BaseEvent
	PositionID uuid.UUID `json:"position_id"`
	AccountID  uuid.UUID `json:"account_id"`
}

func NewDepositMatured(positionID, tenantID, accountID uuid.UUID) DepositMatured {
	return DepositMatured{
		BaseEvent:  events.NewBaseEvent("deposit.position.matured", positionID.String(), AggregateTypeDepositPosition, tenantID.String()),
		PositionID: positionID,
		AccountID:  accountID,
	}
}

// DepositClosed is emitted when a deposit position is closed.
type DepositClosed struct {
	events.BaseEvent
	PositionID uuid.UUID `json:"position_id"`
	AccountID  uuid.UUID `json:"account_id"`
}

func NewDepositClosed(positionID, tenantID, accountID uuid.UUID) DepositClosed {
	return DepositClosed{
		BaseEvent:  events.NewBaseEvent("deposit.position.closed", positionID.String(), AggregateTypeDepositPosition, tenantID.String()),
		PositionID: positionID,
		AccountID:  accountID,
	}
}
