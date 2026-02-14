//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var gatewayURL string

func TestMain(m *testing.M) {
	gatewayURL = os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	// Wait for gateway to be ready
	for i := 0; i < 30; i++ {
		resp, err := http.Get(gatewayURL + "/healthz")
		if err == nil && resp.StatusCode == 200 {
			break
		}
		time.Sleep(2 * time.Second)
	}

	os.Exit(m.Run())
}

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(gatewayURL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	assert.Equal(t, "ok", body["status"])
}

func TestOnboardingFlow(t *testing.T) {
	t.Skip("Requires full stack running - enable in CI")

	// Step 1: Initiate identity verification
	verificationReq := map[string]interface{}{
		"tenant_id":     "00000000-0000-0000-0000-000000000010",
		"first_name":    "John",
		"last_name":     "Doe",
		"email":         "john.doe@example.com",
		"date_of_birth": "1990-01-15",
		"country":       "US",
	}
	resp := postJSON(t, "/api/v1/identity/verifications", verificationReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 2: Open account
	accountReq := map[string]interface{}{
		"tenant_id": "00000000-0000-0000-0000-000000000010",
		"type":      "CHECKING",
		"currency":  "USD",
		"holder": map[string]string{
			"first_name": "John",
			"last_name":  "Doe",
			"email":      "john.doe@example.com",
		},
	}
	resp = postJSON(t, "/api/v1/accounts", accountReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPaymentFlow(t *testing.T) {
	t.Skip("Requires full stack running - enable in CI")

	// Step 1: Initiate payment
	paymentReq := map[string]interface{}{
		"tenant_id":         "00000000-0000-0000-0000-000000000010",
		"source_account_id": "00000000-0000-0000-0000-000000000020",
		"amount":            map[string]string{"amount": "100.00", "currency": "USD"},
		"rail":              "ACH",
		"routing_number":    "021000021",
		"external_account":  "123456789",
		"description":       "Test payment",
	}
	resp := postJSON(t, "/api/v1/payments", paymentReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func postJSON(t *testing.T, path string, body interface{}) *http.Response {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	resp, err := http.Post(gatewayURL+path, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	return resp
}

func getJSON(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(gatewayURL + path)
	require.NoError(t, err)
	return resp
}
