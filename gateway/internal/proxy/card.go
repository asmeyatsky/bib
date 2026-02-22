package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// CardProxy proxies HTTP requests to the card gRPC service.
type CardProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewCardProxy creates a new card service proxy.
func NewCardProxy(conn *ServiceConn, logger *slog.Logger) *CardProxy {
	return &CardProxy{conn: conn, logger: logger}
}

type issueCardReq struct {
	TenantID     string `json:"tenant_id"`
	AccountID    string `json:"account_id"`
	CardType     string `json:"card_type"`
	Currency     string `json:"currency"`
	DailyLimit   string `json:"daily_limit"`
	MonthlyLimit string `json:"monthly_limit"`
}

type issueCardResp struct {
	CardID string `json:"card_id"`
	Status string `json:"status"`
}

type cardResp struct {
	CardID       string `json:"card_id"`
	TenantID     string `json:"tenant_id"`
	AccountID    string `json:"account_id"`
	CardType     string `json:"card_type"`
	Status       string `json:"status"`
	Currency     string `json:"currency"`
	DailyLimit   string `json:"daily_limit"`
	MonthlyLimit string `json:"monthly_limit"`
	MaskedPAN    string `json:"masked_pan"`
	Version      int32  `json:"version"`
}

type authorizeTransactionReq struct {
	CardID           string `json:"card_id"`
	Amount           string `json:"amount"`
	Currency         string `json:"currency"`
	MerchantName     string `json:"merchant_name"`
	MerchantCategory string `json:"merchant_category"`
}

type authorizeTransactionResp struct {
	Approved      bool   `json:"approved"`
	DeclineReason string `json:"decline_reason,omitempty"`
}

type freezeCardResp struct {
	CardID string `json:"card_id"`
	Status string `json:"status"`
}

// IssueCard handles POST /api/v1/cards.
func (p *CardProxy) IssueCard(w http.ResponseWriter, r *http.Request) {
	var req issueCardReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp issueCardResp
	err := p.conn.Invoke(r.Context(), "/bib.card.v1.CardService/IssueCard", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetCard handles GET /api/v1/cards/{id}.
func (p *CardProxy) GetCard(w http.ResponseWriter, r *http.Request) {
	cardID := r.PathValue("id")
	if cardID == "" {
		writeError(w, http.StatusBadRequest, "card id is required")
		return
	}

	req := map[string]string{"card_id": cardID}
	var resp cardResp
	err := p.conn.Invoke(r.Context(), "/bib.card.v1.CardService/GetCard", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// FreezeCard handles POST /api/v1/cards/{id}/freeze.
func (p *CardProxy) FreezeCard(w http.ResponseWriter, r *http.Request) {
	cardID := r.PathValue("id")
	if cardID == "" {
		writeError(w, http.StatusBadRequest, "card id is required")
		return
	}

	req := map[string]string{"card_id": cardID}
	var resp freezeCardResp
	err := p.conn.Invoke(r.Context(), "/bib.card.v1.CardService/FreezeCard", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// AuthorizeTransaction handles POST /api/v1/cards/{id}/authorize.
func (p *CardProxy) AuthorizeTransaction(w http.ResponseWriter, r *http.Request) {
	cardID := r.PathValue("id")
	if cardID == "" {
		writeError(w, http.StatusBadRequest, "card id is required")
		return
	}

	var req authorizeTransactionReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.CardID = cardID

	var resp authorizeTransactionResp
	err := p.conn.Invoke(r.Context(), "/bib.card.v1.CardService/AuthorizeTransaction", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
