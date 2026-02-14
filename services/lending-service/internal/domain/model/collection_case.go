package model

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/lending-service/internal/domain/valueobject"
)

// ---------------------------------------------------------------------------
// CollectionCase entity
// ---------------------------------------------------------------------------

// CollectionCase tracks collections activity for a delinquent or defaulted loan.
type CollectionCase struct {
	id         string
	loanID     string
	tenantID   string
	status     valueobject.CollectionCaseStatus
	assignedTo string
	notes      []string
	createdAt  time.Time
	updatedAt  time.Time
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewCollectionCase creates a new case in OPEN status.
func NewCollectionCase(loanID, tenantID string, now time.Time) (CollectionCase, error) {
	if loanID == "" {
		return CollectionCase{}, errors.New("loan ID is required")
	}
	if tenantID == "" {
		return CollectionCase{}, errors.New("tenant ID is required")
	}
	return CollectionCase{
		id:        uuid.New().String(),
		loanID:    loanID,
		tenantID:  tenantID,
		status:    valueobject.CollectionCaseStatusOpen,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ReconstructCollectionCase rebuilds from persistence.
func ReconstructCollectionCase(
	id, loanID, tenantID string,
	status valueobject.CollectionCaseStatus,
	assignedTo string,
	notes []string,
	createdAt, updatedAt time.Time,
) CollectionCase {
	return CollectionCase{
		id:         id,
		loanID:     loanID,
		tenantID:   tenantID,
		status:     status,
		assignedTo: assignedTo,
		notes:      notes,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}
}

// ---------------------------------------------------------------------------
// Mutations (return new copies)
// ---------------------------------------------------------------------------

// AddNote appends a note to the case.
func (c CollectionCase) AddNote(note string, now time.Time) CollectionCase {
	next := c
	next.notes = make([]string, len(c.notes)+1)
	copy(next.notes, c.notes)
	next.notes[len(c.notes)] = note
	next.updatedAt = now
	return next
}

// Assign sets an agent and transitions to IN_PROGRESS if currently OPEN.
func (c CollectionCase) Assign(agentID string, now time.Time) (CollectionCase, error) {
	if agentID == "" {
		return c, errors.New("agent ID is required")
	}
	next := c
	next.assignedTo = agentID
	next.updatedAt = now
	if c.status.Equal(valueobject.CollectionCaseStatusOpen) {
		next.status = valueobject.CollectionCaseStatusInProgress
	}
	return next, nil
}

// Resolve transitions to RESOLVED.
func (c CollectionCase) Resolve(now time.Time) (CollectionCase, error) {
	if !c.status.Equal(valueobject.CollectionCaseStatusOpen) && !c.status.Equal(valueobject.CollectionCaseStatusInProgress) {
		return c, valueobject.ErrInvalidStatusTransition
	}
	next := c
	next.status = valueobject.CollectionCaseStatusResolved
	next.updatedAt = now
	return next, nil
}

// Close transitions to CLOSED.
func (c CollectionCase) Close(now time.Time) (CollectionCase, error) {
	if !c.status.Equal(valueobject.CollectionCaseStatusResolved) {
		return c, valueobject.ErrInvalidStatusTransition
	}
	next := c
	next.status = valueobject.CollectionCaseStatusClosed
	next.updatedAt = now
	return next, nil
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (c CollectionCase) ID() string                                { return c.id }
func (c CollectionCase) LoanID() string                            { return c.loanID }
func (c CollectionCase) TenantID() string                          { return c.tenantID }
func (c CollectionCase) Status() valueobject.CollectionCaseStatus  { return c.status }
func (c CollectionCase) AssignedTo() string                        { return c.assignedTo }
func (c CollectionCase) CreatedAt() time.Time                      { return c.createdAt }
func (c CollectionCase) UpdatedAt() time.Time                      { return c.updatedAt }

// Notes returns a defensive copy.
func (c CollectionCase) Notes() []string {
	if c.notes == nil {
		return nil
	}
	out := make([]string, len(c.notes))
	copy(out, c.notes)
	return out
}
