package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// AccountHolder is an entity that represents the owner of a customer account.
// It is part of the CustomerAccount aggregate.
type AccountHolder struct {
	firstName              string
	lastName               string
	email                  string
	id                     uuid.UUID
	identityVerificationID uuid.UUID
}

// NewAccountHolder creates a new AccountHolder with validation.
func NewAccountHolder(
	id uuid.UUID,
	firstName string,
	lastName string,
	email string,
	identityVerificationID uuid.UUID,
) (AccountHolder, error) {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	email = strings.TrimSpace(email)

	if firstName == "" {
		return AccountHolder{}, fmt.Errorf("first name is required")
	}
	if lastName == "" {
		return AccountHolder{}, fmt.Errorf("last name is required")
	}
	if email == "" {
		return AccountHolder{}, fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return AccountHolder{}, fmt.Errorf("invalid email format: %q", email)
	}

	if id == uuid.Nil {
		id = uuid.New()
	}

	return AccountHolder{
		id:                     id,
		firstName:              firstName,
		lastName:               lastName,
		email:                  email,
		identityVerificationID: identityVerificationID,
	}, nil
}

// ReconstructAccountHolder recreates an AccountHolder from persisted data without validation.
func ReconstructAccountHolder(
	id uuid.UUID,
	firstName string,
	lastName string,
	email string,
	identityVerificationID uuid.UUID,
) AccountHolder {
	return AccountHolder{
		id:                     id,
		firstName:              firstName,
		lastName:               lastName,
		email:                  email,
		identityVerificationID: identityVerificationID,
	}
}

// ID returns the holder's unique identifier.
func (h AccountHolder) ID() uuid.UUID {
	return h.id
}

// FirstName returns the holder's first name.
func (h AccountHolder) FirstName() string {
	return h.firstName
}

// LastName returns the holder's last name.
func (h AccountHolder) LastName() string {
	return h.lastName
}

// FullName returns the holder's full name.
func (h AccountHolder) FullName() string {
	return h.firstName + " " + h.lastName
}

// Email returns the holder's email address.
func (h AccountHolder) Email() string {
	return h.email
}

// IdentityVerificationID returns the ID of the identity verification record.
func (h AccountHolder) IdentityVerificationID() uuid.UUID {
	return h.identityVerificationID
}
