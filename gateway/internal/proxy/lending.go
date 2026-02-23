package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// LendingProxy proxies HTTP requests to the lending gRPC service.
type LendingProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewLendingProxy creates a new lending service proxy.
func NewLendingProxy(conn *ServiceConn, logger *slog.Logger) *LendingProxy {
	return &LendingProxy{conn: conn, logger: logger}
}

type submitLoanApplicationReq struct {
	TenantID        string `json:"tenant_id"`
	ApplicantID     string `json:"applicant_id"`
	RequestedAmount string `json:"requested_amount"`
	Currency        string `json:"currency"`
	Purpose         string `json:"purpose"`
	TermMonths      int    `json:"term_months"`
}

type loanApplicationResp struct {
	ApplicationID string `json:"application_id"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type disburseLoanReq struct {
	TenantID          string `json:"tenant_id"`
	ApplicationID     string `json:"application_id"`
	BorrowerAccountID string `json:"borrower_account_id"`
	InterestRateBps   int    `json:"interest_rate_bps"`
}

type loanResp struct {
	LoanID    string `json:"loan_id"`
	Status    string `json:"status"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"created_at"`
}

type makeLoanPaymentReq struct {
	TenantID string `json:"tenant_id"`
	LoanID   string `json:"loan_id"`
	Amount   string `json:"amount"`
}

type loanPaymentResp struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
}

// SubmitApplication handles POST /api/v1/loans/applications.
func (p *LendingProxy) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	var req submitLoanApplicationReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp loanApplicationResp
	err := p.conn.Invoke(r.Context(), "/bib.lending.v1.LendingService/SubmitApplication", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetApplication handles GET /api/v1/loans/applications/{id}.
func (p *LendingProxy) GetApplication(w http.ResponseWriter, r *http.Request) {
	applicationID := r.PathValue("id")
	if applicationID == "" {
		writeError(w, http.StatusBadRequest, "application id is required")
		return
	}

	tenantID := ""
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		tenantID = claims.TenantID.String()
	}

	req := map[string]string{
		"tenant_id":      tenantID,
		"application_id": applicationID,
	}
	var resp loanApplicationResp
	err := p.conn.Invoke(r.Context(), "/bib.lending.v1.LendingService/GetApplication", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// DisburseLoan handles POST /api/v1/loans/disburse.
func (p *LendingProxy) DisburseLoan(w http.ResponseWriter, r *http.Request) {
	var req disburseLoanReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp loanResp
	err := p.conn.Invoke(r.Context(), "/bib.lending.v1.LendingService/DisburseLoan", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetLoan handles GET /api/v1/loans/{id}.
func (p *LendingProxy) GetLoan(w http.ResponseWriter, r *http.Request) {
	loanID := r.PathValue("id")
	if loanID == "" {
		writeError(w, http.StatusBadRequest, "loan id is required")
		return
	}

	tenantID := ""
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		tenantID = claims.TenantID.String()
	}

	req := map[string]string{
		"tenant_id": tenantID,
		"loan_id":   loanID,
	}
	var resp loanResp
	err := p.conn.Invoke(r.Context(), "/bib.lending.v1.LendingService/GetLoan", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// MakePayment handles POST /api/v1/loans/{id}/payments.
func (p *LendingProxy) MakePayment(w http.ResponseWriter, r *http.Request) {
	loanID := r.PathValue("id")
	if loanID == "" {
		writeError(w, http.StatusBadRequest, "loan id is required")
		return
	}

	var req makeLoanPaymentReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.LoanID = loanID
	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp loanPaymentResp
	err := p.conn.Invoke(r.Context(), "/bib.lending.v1.LendingService/MakePayment", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}
