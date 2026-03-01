package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
)

// Compile-time interface check.
var _ port.VerificationProvider = (*PersonaClient)(nil)

// PersonaClient implements port.VerificationProvider using the Persona API.
type PersonaClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewPersonaClient creates a new Persona API client.
func NewPersonaClient(apiKey, baseURL string) *PersonaClient {
	return &PersonaClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// personaInquiryResponse represents the Persona API inquiry response.
type personaInquiryResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Status string `json:"status"`
		} `json:"attributes"`
	} `json:"data"`
}

// InitiateCheck starts a verification check via the Persona API.
func (c *PersonaClient) InitiateCheck(ctx context.Context, checkType valueobject.CheckType, applicant port.ApplicantInfo) (string, error) {
	payload := fmt.Sprintf(`{
		"data": {
			"attributes": {
				"inquiry-template-id": %q,
				"fields": {
					"name-first": {"type": "string", "value": %q},
					"name-last": {"type": "string", "value": %q},
					"email-address": {"type": "string", "value": %q},
					"birthdate": {"type": "string", "value": %q},
					"address-country-code": {"type": "string", "value": %q}
				}
			}
		}
	}`, checkType.String(), applicant.FirstName, applicant.LastName, applicant.Email, applicant.DateOfBirth, applicant.Country)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/inquiries", strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Persona-Version", "2023-01-05")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("persona API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("persona API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result personaInquiryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data.ID, nil
}

// GetCheckResult retrieves the result of a previously initiated check.
func (c *PersonaClient) GetCheckResult(ctx context.Context, providerRef string) (valueobject.VerificationStatus, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/inquiries/"+providerRef, nil)
	if err != nil {
		return valueobject.VerificationStatus{}, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Persona-Version", "2023-01-05")

	resp, err := c.client.Do(req)
	if err != nil {
		return valueobject.VerificationStatus{}, "", fmt.Errorf("persona API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return valueobject.VerificationStatus{}, "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return valueobject.VerificationStatus{}, "", fmt.Errorf("persona API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result personaInquiryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return valueobject.VerificationStatus{}, "", fmt.Errorf("failed to parse response: %w", err)
	}

	status, failureReason := mapPersonaStatus(result.Data.Attributes.Status)
	return status, failureReason, nil
}

// mapPersonaStatus maps Persona inquiry statuses to domain VerificationStatus values.
func mapPersonaStatus(personaStatus string) (valueobject.VerificationStatus, string) {
	switch personaStatus {
	case "completed", "approved":
		return valueobject.StatusApproved, ""
	case "declined", "failed":
		return valueobject.StatusRejected, "verification_" + personaStatus
	case "expired":
		return valueobject.StatusExpired, "inquiry_expired"
	case "pending", "created":
		return valueobject.StatusPending, ""
	default:
		return valueobject.StatusInProgress, ""
	}
}
