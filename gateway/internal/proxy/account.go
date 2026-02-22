package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// AccountProxy proxies HTTP requests to the account gRPC service.
type AccountProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewAccountProxy creates a new account service proxy.
func NewAccountProxy(conn *ServiceConn, logger *slog.Logger) *AccountProxy {
	return &AccountProxy{conn: conn, logger: logger}
}

// --- Request/Response types matching the account service handler ---

type openAccountReq struct {
	TenantID               string `json:"tenant_id"`
	AccountType            string `json:"account_type"`
	Currency               string `json:"currency"`
	HolderFirstName        string `json:"holder_first_name"`
	HolderLastName         string `json:"holder_last_name"`
	HolderEmail            string `json:"holder_email"`
	IdentityVerificationID string `json:"identity_verification_id,omitempty"`
}

type openAccountResp struct {
	AccountID         string `json:"account_id"`
	AccountNumber     string `json:"account_number"`
	Status            string `json:"status"`
	LedgerAccountCode string `json:"ledger_account_code"`
}

type accountResp struct {
	AccountID         string `json:"account_id"`
	TenantID          string `json:"tenant_id"`
	AccountNumber     string `json:"account_number"`
	AccountType       string `json:"account_type"`
	Status            string `json:"status"`
	Currency          string `json:"currency"`
	LedgerAccountCode string `json:"ledger_account_code"`
	HolderFirstName   string `json:"holder_first_name"`
	HolderLastName    string `json:"holder_last_name"`
	HolderEmail       string `json:"holder_email"`
	Version           int32  `json:"version"`
}

type listAccountsResp struct {
	Accounts   []accountResp `json:"accounts"`
	TotalCount int32         `json:"total_count"`
}

type freezeCloseReq struct {
	Reason string `json:"reason"`
}

// OpenAccount handles POST /api/v1/accounts.
func (p *AccountProxy) OpenAccount(w http.ResponseWriter, r *http.Request) {
	var req openAccountReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp openAccountResp
	err := p.conn.Invoke(r.Context(), "/bib.account.v1.AccountService/OpenAccount", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetAccount handles GET /api/v1/accounts/{id}.
func (p *AccountProxy) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	req := map[string]string{"account_id": accountID}
	var resp accountResp
	err := p.conn.Invoke(r.Context(), "/bib.account.v1.AccountService/GetAccount", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// FreezeAccount handles POST /api/v1/accounts/{id}/freeze.
func (p *AccountProxy) FreezeAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	var body freezeCloseReq
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req := map[string]string{
		"account_id": accountID,
		"reason":     body.Reason,
	}
	var resp accountResp
	err := p.conn.Invoke(r.Context(), "/bib.account.v1.AccountService/FreezeAccount", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// CloseAccount handles POST /api/v1/accounts/{id}/close.
func (p *AccountProxy) CloseAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("id")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "account id is required")
		return
	}

	var body freezeCloseReq
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req := map[string]string{
		"account_id": accountID,
		"reason":     body.Reason,
	}
	var resp accountResp
	err := p.conn.Invoke(r.Context(), "/bib.account.v1.AccountService/CloseAccount", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListAccounts handles GET /api/v1/accounts.
func (p *AccountProxy) ListAccounts(w http.ResponseWriter, r *http.Request) {
	req := map[string]interface{}{
		"tenant_id": r.URL.Query().Get("tenant_id"),
		"holder_id": r.URL.Query().Get("holder_id"),
	}

	if req["tenant_id"] == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req["tenant_id"] = claims.TenantID.String()
		}
	}

	var resp listAccountsResp
	err := p.conn.Invoke(r.Context(), "/bib.account.v1.AccountService/ListAccounts", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
