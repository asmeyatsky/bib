package proxy

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// PartnerProxy handles embedded finance / partner API requests.
// It provides API key authentication, per-partner rate limiting,
// and webhook registration for partner integrations.
type PartnerProxy struct {
	logger       *slog.Logger
	partners     map[string]*PartnerConfig
	webhooks     map[string][]WebhookRegistration
	accountConn  *ServiceConn
	paymentConn  *ServiceConn
	ledgerConn   *ServiceConn
	rateLimiters map[string]*partnerRateLimiter
	mu           sync.RWMutex
}

// PartnerConfig holds the configuration for a partner integration.
type PartnerConfig struct {
	PartnerID        string
	PartnerName      string
	APIKey           string
	WebhookSecret    string
	AllowedEndpoints []string
	RateLimit        int
	IsActive         bool
}

// WebhookRegistration represents a partner's webhook endpoint.
type WebhookRegistration struct {
	ID        string   `json:"id"`
	PartnerID string   `json:"partner_id"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret,omitempty"`
	CreatedAt string   `json:"created_at"`
	Events    []string `json:"events"`
}

type partnerRateLimiter struct {
	lastRefill time.Time
	tokens     float64
	maxTokens  float64
	refillRate float64
	mu         sync.Mutex
}

func newPartnerRateLimiter(rps int) *partnerRateLimiter {
	return &partnerRateLimiter{
		tokens:     float64(rps),
		maxTokens:  float64(rps),
		refillRate: float64(rps),
		lastRefill: time.Now(),
	}
}

func (rl *partnerRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

// NewPartnerProxy creates a new partner API proxy.
func NewPartnerProxy(
	accountConn, paymentConn, ledgerConn *ServiceConn,
	logger *slog.Logger,
) *PartnerProxy {
	return &PartnerProxy{
		logger:       logger,
		partners:     make(map[string]*PartnerConfig),
		webhooks:     make(map[string][]WebhookRegistration),
		accountConn:  accountConn,
		paymentConn:  paymentConn,
		ledgerConn:   ledgerConn,
		rateLimiters: make(map[string]*partnerRateLimiter),
	}
}

// RegisterPartner adds a partner configuration. In production, this would
// be backed by a database.
func (p *PartnerProxy) RegisterPartner(config PartnerConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.partners[config.APIKey] = &config
	p.rateLimiters[config.PartnerID] = newPartnerRateLimiter(config.RateLimit)
}

// authenticatePartner validates the API key and returns the partner config.
func (p *PartnerProxy) authenticatePartner(r *http.Request) (*PartnerConfig, error) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		return nil, fmt.Errorf("missing X-API-Key header")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	for key, partner := range p.partners {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 1 {
			if !partner.IsActive {
				return nil, fmt.Errorf("partner integration is disabled")
			}
			return partner, nil
		}
	}
	return nil, fmt.Errorf("invalid API key")
}

// checkRateLimit checks if the partner has exceeded their rate limit.
func (p *PartnerProxy) checkRateLimit(partnerID string) bool {
	p.mu.RLock()
	rl, ok := p.rateLimiters[partnerID]
	p.mu.RUnlock()
	if !ok {
		return false
	}
	return rl.allow()
}

// --- HTTP Handlers ---

// CreateAccount handles POST /api/v1/partner/accounts.
func (p *PartnerProxy) CreateAccount(w http.ResponseWriter, r *http.Request) {
	partner, err := p.authenticatePartner(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if !p.checkRateLimit(partner.PartnerID) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	var req map[string]interface{}
	if err = readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Add partner context
	req["partner_id"] = partner.PartnerID

	var resp map[string]interface{}
	err = p.accountConn.Invoke(r.Context(), "/bib.account.v1.AccountService/OpenAccount", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}

	p.logger.Info("partner account created",
		"partner_id", partner.PartnerID,
		"partner_name", partner.PartnerName,
	)
	writeJSON(w, http.StatusCreated, resp)
}

// InitiatePayment handles POST /api/v1/partner/payments.
func (p *PartnerProxy) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	partner, err := p.authenticatePartner(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if !p.checkRateLimit(partner.PartnerID) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	var req map[string]interface{}
	if err = readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req["partner_id"] = partner.PartnerID

	var resp map[string]interface{}
	err = p.paymentConn.Invoke(r.Context(), "/bib.payment.v1.PaymentService/InitiatePayment", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}

	p.logger.Info("partner payment initiated",
		"partner_id", partner.PartnerID,
	)
	writeJSON(w, http.StatusCreated, resp)
}

// GetBalance handles GET /api/v1/partner/balances/{account_code}.
func (p *PartnerProxy) GetBalance(w http.ResponseWriter, r *http.Request) {
	partner, err := p.authenticatePartner(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if !p.checkRateLimit(partner.PartnerID) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	accountCode := r.PathValue("account_code")
	if accountCode == "" {
		writeError(w, http.StatusBadRequest, "account code is required")
		return
	}

	req := map[string]string{
		"account_code": accountCode,
		"partner_id":   partner.PartnerID,
	}
	var resp map[string]interface{}
	err = p.ledgerConn.Invoke(r.Context(), "/bib.ledger.v1.LedgerService/GetBalance", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// RegisterWebhook handles POST /api/v1/partner/webhooks.
func (p *PartnerProxy) RegisterWebhook(w http.ResponseWriter, r *http.Request) {
	partner, err := p.authenticatePartner(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "webhook URL is required")
		return
	}
	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "at least one event type is required")
		return
	}

	registration := WebhookRegistration{
		ID:        fmt.Sprintf("wh-%s-%d", partner.PartnerID, time.Now().UnixNano()),
		PartnerID: partner.PartnerID,
		URL:       req.URL,
		Events:    req.Events,
		Secret:    partner.WebhookSecret,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	p.mu.Lock()
	p.webhooks[partner.PartnerID] = append(p.webhooks[partner.PartnerID], registration)
	p.mu.Unlock()

	p.logger.Info("partner webhook registered",
		"partner_id", partner.PartnerID,
		"webhook_url", req.URL,
		"events", req.Events,
	)

	// Return registration without the secret
	registration.Secret = ""
	writeJSON(w, http.StatusCreated, registration)
}

// ListWebhooks handles GET /api/v1/partner/webhooks.
func (p *PartnerProxy) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	partner, err := p.authenticatePartner(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	p.mu.RLock()
	hooks := p.webhooks[partner.PartnerID]
	p.mu.RUnlock()

	// Strip secrets before returning
	sanitized := make([]WebhookRegistration, len(hooks))
	for i, h := range hooks {
		sanitized[i] = h
		sanitized[i].Secret = ""
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
		"webhooks": sanitized,
		"count":    len(sanitized),
	})
}
