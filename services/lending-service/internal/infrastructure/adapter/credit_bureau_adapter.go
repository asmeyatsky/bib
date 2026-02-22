package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// Credit Bureau Adapter – structured for real integration
// ---------------------------------------------------------------------------

// Bureau identifies a credit bureau provider.
type Bureau string

const (
	BureauExperian   Bureau = "EXPERIAN"
	BureauTransUnion Bureau = "TRANSUNION"
	BureauEquifax    Bureau = "EQUIFAX"
)

// CreditBureauConfig holds configuration for the credit bureau adapter.
type CreditBureauConfig struct {
	// PrimaryBureau is the preferred bureau for credit pulls.
	PrimaryBureau Bureau
	// BaseURL is the base URL for the credit bureau API.
	BaseURL string
	// APIKey is the authentication credential for the bureau API.
	APIKey string
	// TimeoutSeconds is the HTTP client timeout.
	TimeoutSeconds int
	// MaxRetries is the maximum number of retry attempts on transient failures.
	MaxRetries int
	// RetryBackoffMs is the base backoff duration in milliseconds between retries.
	RetryBackoffMs int
}

// DefaultCreditBureauConfig returns sensible defaults for development.
func DefaultCreditBureauConfig() CreditBureauConfig {
	return CreditBureauConfig{
		PrimaryBureau:  BureauExperian,
		BaseURL:        "https://api.creditbureau.example.com",
		APIKey:         "dev-api-key",
		TimeoutSeconds: 10,
		MaxRetries:     3,
		RetryBackoffMs: 200,
	}
}

// CreditReport represents a parsed credit report from a bureau.
type CreditReport struct {
	Bureau         Bureau
	ApplicantID    string
	Score          int
	ScoreModel     string // e.g. "FICO8", "VantageScore3"
	ReportDate     time.Time
	AccountCount   int
	TotalDebt      string // decimal string
	OldestAccount  time.Time
	RecentInquiry  time.Time
	DerogCount     int
	PaymentHistory string // "GOOD", "FAIR", "POOR"
}

// HTTPClient defines the interface for making HTTP requests to credit bureaus.
// This enables testing with mock implementations.
type HTTPClient interface {
	// FetchCreditReport retrieves a credit report from the specified bureau.
	FetchCreditReport(ctx context.Context, bureau Bureau, applicantID string) (CreditReport, error)
}

// CreditBureauAdapter is a structured adapter that simulates credit bureau
// API calls. It implements port.CreditBureauClient and is designed to be
// swapped with a real HTTP-based implementation when integrating with
// Experian, TransUnion, or Equifax APIs.
type CreditBureauAdapter struct {
	config CreditBureauConfig
	client HTTPClient // nil = use simulated responses
}

// NewCreditBureauAdapter creates a new adapter with the given configuration.
// If client is nil, simulated responses are used (suitable for development/testing).
func NewCreditBureauAdapter(config CreditBureauConfig, client HTTPClient) *CreditBureauAdapter {
	return &CreditBureauAdapter{
		config: config,
		client: client,
	}
}

// GetCreditScore retrieves a credit score for the given applicant.
// It implements port.CreditBureauClient.
//
// When a real HTTPClient is provided, the adapter calls the bureau API with
// retry logic. Otherwise, it returns a deterministic simulated score.
func (a *CreditBureauAdapter) GetCreditScore(ctx context.Context, applicantID string) (string, error) {
	if applicantID == "" {
		return "", fmt.Errorf("applicant ID is required")
	}

	// If a real HTTP client is provided, use it with retry logic.
	if a.client != nil {
		report, err := a.fetchWithRetry(ctx, applicantID)
		if err != nil {
			return "", fmt.Errorf("credit bureau request failed: %w", err)
		}
		return fmt.Sprintf("%d", report.Score), nil
	}

	// Simulated response: deterministic score based on applicant ID hash.
	report := a.simulateCreditReport(applicantID)
	return fmt.Sprintf("%d", report.Score), nil
}

// GetFullReport retrieves a complete credit report for the applicant.
// This method is not part of the minimal CreditBureauClient port but
// provides additional data for enhanced underwriting.
func (a *CreditBureauAdapter) GetFullReport(ctx context.Context, applicantID string) (CreditReport, error) {
	if applicantID == "" {
		return CreditReport{}, fmt.Errorf("applicant ID is required")
	}

	if a.client != nil {
		return a.fetchWithRetry(ctx, applicantID)
	}

	return a.simulateCreditReport(applicantID), nil
}

// fetchWithRetry calls the bureau API with exponential backoff retry logic.
func (a *CreditBureauAdapter) fetchWithRetry(ctx context.Context, applicantID string) (CreditReport, error) {
	var lastErr error

	for attempt := 0; attempt <= a.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter.
			backoff := time.Duration(a.config.RetryBackoffMs) * time.Millisecond * (1 << uint(attempt-1))
			jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
			select {
			case <-ctx.Done():
				return CreditReport{}, ctx.Err()
			case <-time.After(backoff + jitter):
			}
		}

		report, err := a.client.FetchCreditReport(ctx, a.config.PrimaryBureau, applicantID)
		if err == nil {
			return report, nil
		}
		lastErr = err
	}

	return CreditReport{}, fmt.Errorf("exhausted %d retries: %w", a.config.MaxRetries, lastErr)
}

// simulateCreditReport generates a deterministic simulated credit report.
// The score and attributes are derived from the applicant ID hash, making
// results reproducible for testing.
func (a *CreditBureauAdapter) simulateCreditReport(applicantID string) CreditReport {
	h := sha256.Sum256([]byte(applicantID))
	score := 300 + int(binary.BigEndian.Uint32(h[:4])%551)

	accountCount := 1 + int(binary.BigEndian.Uint16(h[4:6])%20)
	derogCount := int(binary.BigEndian.Uint16(h[6:8]) % 5)

	paymentHistory := "GOOD"
	if score < 600 {
		paymentHistory = "POOR"
	} else if score < 700 {
		paymentHistory = "FAIR"
	}

	scoreModel := "FICO8"
	if a.config.PrimaryBureau == BureauTransUnion {
		scoreModel = "VantageScore3"
	}

	return CreditReport{
		Bureau:         a.config.PrimaryBureau,
		ApplicantID:    applicantID,
		Score:          score,
		ScoreModel:     scoreModel,
		ReportDate:     time.Now().UTC(),
		AccountCount:   accountCount,
		TotalDebt:      fmt.Sprintf("%d", 1000+int(binary.BigEndian.Uint32(h[8:12])%500000)),
		OldestAccount:  time.Now().AddDate(-accountCount, 0, 0),
		RecentInquiry:  time.Now().AddDate(0, -1, 0),
		DerogCount:     derogCount,
		PaymentHistory: paymentHistory,
	}
}
