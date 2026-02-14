package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// StubCreditBureauClient is a development/test adapter that returns a
// deterministic credit score derived from the applicant ID.
// It implements port.CreditBureauClient.
type StubCreditBureauClient struct{}

// NewStubCreditBureauClient creates a new stub adapter.
func NewStubCreditBureauClient() *StubCreditBureauClient {
	return &StubCreditBureauClient{}
}

// GetCreditScore returns a deterministic score between 300 and 850 based on
// a hash of the applicant ID. This allows repeatable test scenarios.
func (c *StubCreditBureauClient) GetCreditScore(_ context.Context, applicantID string) (string, error) {
	if applicantID == "" {
		return "", fmt.Errorf("applicant ID is required")
	}

	h := sha256.Sum256([]byte(applicantID))
	num := binary.BigEndian.Uint32(h[:4])
	score := 300 + int(num%551) // range [300, 850]

	return fmt.Sprintf("%d", score), nil
}
