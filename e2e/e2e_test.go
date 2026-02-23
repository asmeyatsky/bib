//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// gatewayURL returns the base URL of the gateway under test.
func gatewayURL() string {
	if url := os.Getenv("GATEWAY_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

// jwtSecret returns the HMAC signing key shared with the gateway.
func jwtSecret() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "test-e2e-secret"
}

// testTenantID is a fixed tenant UUID used across all e2e tests.
var testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000010")

// testUserID is a fixed user UUID used across all e2e tests.
var testUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// getTestToken generates a valid JWT token for e2e tests using the shared
// HMAC secret. The token carries admin + operator roles so that all endpoints
// are accessible.
func getTestToken(t *testing.T) string {
	t.Helper()

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       "bib-gateway",
		"sub":       testUserID.String(),
		"exp":       jwt.NewNumericDate(now.Add(1 * time.Hour)),
		"iat":       jwt.NewNumericDate(now),
		"nbf":       jwt.NewNumericDate(now),
		"jti":       uuid.New().String(),
		"user_id":   testUserID.String(),
		"tenant_id": testTenantID.String(),
		"roles":     []string{"admin", "operator"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret()))
	require.NoError(t, err, "failed to sign test JWT")
	return signed
}

// newClient returns an *http.Client with a reasonable timeout for e2e calls.
func newClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

// doJSON performs an HTTP request with JSON body and auth header, returning the
// parsed response body as a map and the raw *http.Response (so callers can
// assert on status codes). The response body is closed before returning.
func doJSON(t *testing.T, client *http.Client, method, url, token string, body interface{}) (map[string]interface{}, *http.Response) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	if len(respBody) > 0 {
		err = json.Unmarshal(respBody, &result)
		if err != nil {
			// Not all responses are JSON objects (could be arrays, etc.).
			// Store raw body for debugging.
			t.Logf("response body (non-JSON-object): %s", string(respBody))
		}
	}
	return result, resp
}

// postJSON is a convenience wrapper around http.Post with JSON content.
// It does NOT include an auth token (legacy helper kept for health check).
func postJSON(t *testing.T, path string, body interface{}) *http.Response {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	resp, err := http.Post(gatewayURL()+path, "application/json", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)
	return resp
}

// getJSON is a convenience wrapper around http.Get (no auth token).
func getJSON(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(gatewayURL() + path)
	require.NoError(t, err)
	return resp
}

// ---------------------------------------------------------------------------
// TestMain — wait for gateway readiness
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	base := gatewayURL()

	// Wait for gateway to be ready (up to 60 s).
	ready := false
	for i := 0; i < 30; i++ {
		resp, err := http.Get(base + "/healthz")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			ready = true
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	if !ready {
		fmt.Fprintf(os.Stderr, "gateway at %s did not become healthy within 60 s\n", base)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Health check (no auth required)
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(gatewayURL() + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

// ---------------------------------------------------------------------------
// Onboarding flow — identity verification then account opening
// ---------------------------------------------------------------------------

func TestOnboardingFlow(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// Step 1: Initiate identity verification.
	verificationReq := map[string]interface{}{
		"tenant_id":     testTenantID.String(),
		"first_name":    "John",
		"last_name":     "Doe",
		"email":         "john.doe@example.com",
		"date_of_birth": "1990-01-15",
		"country":       "US",
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/identity/verifications", token, verificationReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "initiate verification failed: %v", result)

	verification, ok := result["verification"].(map[string]interface{})
	require.True(t, ok, "response missing 'verification' object")
	verificationID, ok := verification["id"].(string)
	require.True(t, ok && verificationID != "", "verification id is missing or empty")
	assert.Equal(t, "John", verification["applicant_first_name"])
	assert.Equal(t, "Doe", verification["applicant_last_name"])
	assert.NotEmpty(t, verification["status"])

	// Step 2: Get verification by ID.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/identity/verifications/"+verificationID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get verification failed: %v", result)
	verification2 := result["verification"].(map[string]interface{})
	assert.Equal(t, verificationID, verification2["id"])

	// Step 3: Open account.
	accountReq := map[string]interface{}{
		"tenant_id":                testTenantID.String(),
		"account_type":             "CHECKING",
		"currency":                 "USD",
		"holder_first_name":        "John",
		"holder_last_name":         "Doe",
		"holder_email":             "john.doe@example.com",
		"identity_verification_id": verificationID,
	}
	result, resp = doJSON(t, client, "POST", base+"/api/v1/accounts", token, accountReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "open account failed: %v", result)

	accountID, ok := result["account_id"].(string)
	require.True(t, ok && accountID != "", "account_id is missing or empty")
	assert.NotEmpty(t, result["account_number"])
	assert.Equal(t, "ACTIVE", result["status"])
}

// ---------------------------------------------------------------------------
// Payment flow
// ---------------------------------------------------------------------------

func TestPaymentFlow(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// Step 1: Initiate payment.
	paymentReq := map[string]interface{}{
		"tenant_id":               testTenantID.String(),
		"source_account_id":       uuid.New().String(),
		"amount":                  "100.00",
		"currency":                "USD",
		"routing_number":          "021000021",
		"external_account_number": "123456789",
		"description":             "E2E test payment",
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/payments", token, paymentReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "initiate payment failed: %v", result)

	paymentID, ok := result["id"].(string)
	require.True(t, ok && paymentID != "", "payment id is missing or empty")
	assert.NotEmpty(t, result["status"])
	assert.NotEmpty(t, result["rail"])

	// Step 2: Get payment by ID.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/payments/"+paymentID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get payment failed: %v", result)

	payment, ok := result["payment"].(map[string]interface{})
	require.True(t, ok, "response missing 'payment' object")
	assert.Equal(t, paymentID, payment["id"])
	assert.Equal(t, "100.00", payment["amount"])
	assert.Equal(t, "USD", payment["currency"])

	// Step 3: List payments for tenant.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/payments", token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "list payments failed: %v", result)
	assert.NotNil(t, result["payments"], "payments array should be present")
}

// ---------------------------------------------------------------------------
// Account lifecycle — open, get, freeze, close
// ---------------------------------------------------------------------------

func TestAccountLifecycle(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// 1. Open account.
	openReq := map[string]interface{}{
		"tenant_id":        testTenantID.String(),
		"account_type":     "CHECKING",
		"currency":         "USD",
		"holder_first_name": "Alice",
		"holder_last_name":  "Smith",
		"holder_email":      "alice.smith@example.com",
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/accounts", token, openReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "open account failed: %v", result)

	accountID, ok := result["account_id"].(string)
	require.True(t, ok && accountID != "", "account_id missing")
	assert.NotEmpty(t, result["account_number"], "account_number should be set")
	assert.Equal(t, "ACTIVE", result["status"])

	// 2. Get account.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/accounts/"+accountID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get account failed: %v", result)
	assert.Equal(t, accountID, result["account_id"])
	assert.Equal(t, "CHECKING", result["account_type"])
	assert.Equal(t, "USD", result["currency"])
	assert.Equal(t, "Alice", result["holder_first_name"])

	// 3. Freeze account.
	freezeReq := map[string]interface{}{
		"reason": "Suspicious activity detected",
	}
	result, resp = doJSON(t, client, "POST", base+"/api/v1/accounts/"+accountID+"/freeze", token, freezeReq)
	require.Equal(t, http.StatusOK, resp.StatusCode, "freeze account failed: %v", result)
	assert.Equal(t, "FROZEN", result["status"])
	assert.Equal(t, accountID, result["account_id"])

	// 4. Close account.
	closeReq := map[string]interface{}{
		"reason": "Customer requested closure",
	}
	result, resp = doJSON(t, client, "POST", base+"/api/v1/accounts/"+accountID+"/close", token, closeReq)
	require.Equal(t, http.StatusOK, resp.StatusCode, "close account failed: %v", result)
	assert.Equal(t, "CLOSED", result["status"])
	assert.Equal(t, accountID, result["account_id"])
}

// ---------------------------------------------------------------------------
// Deposit flow — create product, open position, get position
// ---------------------------------------------------------------------------

func TestDepositFlow(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// 1. Create deposit product.
	productReq := map[string]interface{}{
		"tenant_id": testTenantID.String(),
		"name":      "12-Month Fixed Deposit",
		"currency":  "USD",
		"tiers": []map[string]interface{}{
			{"min_balance": "1000.00", "max_balance": "50000.00", "rate_bps": 450},
			{"min_balance": "50000.01", "max_balance": "1000000.00", "rate_bps": 500},
		},
		"term_days": 365,
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/deposits/products", token, productReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create deposit product failed: %v", result)

	product, ok := result["product"].(map[string]interface{})
	require.True(t, ok, "response missing 'product' object")
	productID, ok := product["id"].(string)
	require.True(t, ok && productID != "", "product id missing")
	assert.Equal(t, "12-Month Fixed Deposit", product["name"])
	assert.Equal(t, "USD", product["currency"])
	assert.Equal(t, true, product["is_active"])

	// 2. Open deposit position.
	positionReq := map[string]interface{}{
		"tenant_id":  testTenantID.String(),
		"account_id": uuid.New().String(),
		"product_id": productID,
		"principal":  "10000.00",
	}
	result, resp = doJSON(t, client, "POST", base+"/api/v1/deposits/positions", token, positionReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "open deposit position failed: %v", result)

	position, ok := result["position"].(map[string]interface{})
	require.True(t, ok, "response missing 'position' object")
	positionID, ok := position["id"].(string)
	require.True(t, ok && positionID != "", "position id missing")
	assert.Equal(t, "10000.00", position["principal"])
	assert.NotEmpty(t, position["status"])

	// 3. Get deposit position.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/deposits/positions/"+positionID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get deposit position failed: %v", result)

	position2, ok := result["position"].(map[string]interface{})
	require.True(t, ok, "response missing 'position' object")
	assert.Equal(t, positionID, position2["id"])
	assert.Equal(t, productID, position2["product_id"])
}

// ---------------------------------------------------------------------------
// Lending flow — submit loan application, check status
// ---------------------------------------------------------------------------

func TestLendingFlow(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// 1. Submit loan application.
	applicationReq := map[string]interface{}{
		"tenant_id":        testTenantID.String(),
		"applicant_id":     uuid.New().String(),
		"requested_amount": "25000.00",
		"currency":         "USD",
		"term_months":      36,
		"purpose":          "Home renovation",
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/loans/applications", token, applicationReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "submit loan application failed: %v", result)

	applicationID, ok := result["application_id"].(string)
	require.True(t, ok && applicationID != "", "application_id missing")
	assert.NotEmpty(t, result["status"])
	assert.NotEmpty(t, result["created_at"])

	// 2. Get loan application status.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/loans/applications/"+applicationID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get loan application failed: %v", result)
	assert.Equal(t, applicationID, result["application_id"])
	assert.NotEmpty(t, result["status"])
}

// ---------------------------------------------------------------------------
// Card issuance — issue virtual card, get card details
// ---------------------------------------------------------------------------

func TestCardIssuance(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// 1. Issue virtual card.
	issueReq := map[string]interface{}{
		"tenant_id":     testTenantID.String(),
		"account_id":    uuid.New().String(),
		"card_type":     "VIRTUAL",
		"currency":      "USD",
		"daily_limit":   "5000.00",
		"monthly_limit": "25000.00",
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/cards", token, issueReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "issue card failed: %v", result)

	cardID, ok := result["card_id"].(string)
	require.True(t, ok && cardID != "", "card_id missing")
	assert.NotEmpty(t, result["status"])

	// 2. Get card details.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/cards/"+cardID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get card failed: %v", result)
	assert.Equal(t, cardID, result["card_id"])
	assert.Equal(t, "VIRTUAL", result["card_type"])
	assert.Equal(t, "USD", result["currency"])
	assert.NotEmpty(t, result["masked_pan"], "masked_pan should be present")
	assert.Equal(t, "5000.00", result["daily_limit"])
	assert.Equal(t, "25000.00", result["monthly_limit"])
}

// ---------------------------------------------------------------------------
// FX rate query — get rate, convert amount
// ---------------------------------------------------------------------------

func TestFXRateQuery(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// 1. Get exchange rate for USD/EUR.
	result, resp := doJSON(t, client, "GET", base+"/api/v1/fx/rates/USD-EUR", token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get fx rate failed: %v", result)
	assert.Equal(t, "USD", result["base_currency"])
	assert.Equal(t, "EUR", result["quote_currency"])
	assert.NotEmpty(t, result["rate"], "rate should be present")
	assert.NotEmpty(t, result["timestamp"], "timestamp should be present")

	// 2. Convert amount.
	convertReq := map[string]interface{}{
		"tenant_id":     testTenantID.String(),
		"from_currency": "USD",
		"to_currency":   "EUR",
		"amount":        "1000.00",
	}
	result, resp = doJSON(t, client, "POST", base+"/api/v1/fx/convert", token, convertReq)
	require.Equal(t, http.StatusOK, resp.StatusCode, "fx convert failed: %v", result)
	assert.Equal(t, "1000.00", result["original_amount"])
	assert.Equal(t, "USD", result["from_currency"])
	assert.Equal(t, "EUR", result["to_currency"])
	assert.NotEmpty(t, result["converted_amount"], "converted_amount should be present")
	assert.NotEmpty(t, result["rate"], "rate should be present")
}

// ---------------------------------------------------------------------------
// Fraud assessment — submit transaction, get assessment result
// ---------------------------------------------------------------------------

func TestFraudAssessment(t *testing.T) {
	client := newClient()
	token := getTestToken(t)
	base := gatewayURL()

	// 1. Submit transaction for fraud assessment.
	assessReq := map[string]interface{}{
		"tenant_id":        testTenantID.String(),
		"transaction_id":   uuid.New().String(),
		"account_id":       uuid.New().String(),
		"amount":           "5000.00",
		"currency":         "USD",
		"transaction_type": "WIRE_TRANSFER",
		"metadata": map[string]string{
			"ip_address":          "192.168.1.100",
			"destination_country": "US",
		},
	}
	result, resp := doJSON(t, client, "POST", base+"/api/v1/fraud/assessments", token, assessReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "fraud assessment failed: %v", result)

	assessmentID, ok := result["assessment_id"].(string)
	require.True(t, ok && assessmentID != "", "assessment_id missing")
	assert.NotNil(t, result["risk_score"], "risk_score should be present")
	assert.NotEmpty(t, result["risk_level"], "risk_level should be present")
	assert.NotEmpty(t, result["decision"], "decision should be present")

	// Validate risk_score is a number between 0 and 100.
	riskScore, ok := result["risk_score"].(float64)
	require.True(t, ok, "risk_score should be a number")
	assert.GreaterOrEqual(t, riskScore, float64(0))
	assert.LessOrEqual(t, riskScore, float64(100))

	// 2. Get fraud assessment by ID.
	result, resp = doJSON(t, client, "GET", base+"/api/v1/fraud/assessments/"+assessmentID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "get fraud assessment failed: %v", result)
	assert.Equal(t, assessmentID, result["assessment_id"])
	assert.Equal(t, "5000.00", result["amount"])
	assert.Equal(t, "USD", result["currency"])
	assert.Equal(t, "WIRE_TRANSFER", result["transaction_type"])
	assert.NotEmpty(t, result["decision"])
}
