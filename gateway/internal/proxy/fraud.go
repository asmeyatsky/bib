package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// FraudProxy proxies HTTP requests to the fraud gRPC service.
type FraudProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewFraudProxy creates a new fraud service proxy.
func NewFraudProxy(conn *ServiceConn, logger *slog.Logger) *FraudProxy {
	return &FraudProxy{conn: conn, logger: logger}
}

type assessTransactionReq struct {
	Metadata        map[string]string `json:"metadata,omitempty"`
	TenantID        string            `json:"tenant_id"`
	TransactionID   string            `json:"transaction_id"`
	AccountID       string            `json:"account_id"`
	Amount          string            `json:"amount"`
	Currency        string            `json:"currency"`
	TransactionType string            `json:"transaction_type"`
}

type assessTransactionResp struct {
	AssessmentID string   `json:"assessment_id"`
	RiskLevel    string   `json:"risk_level"`
	Decision     string   `json:"decision"`
	Signals      []string `json:"signals"`
	RiskScore    int      `json:"risk_score"`
}

type getAssessmentResp struct {
	AssessmentID    string   `json:"assessment_id"`
	TransactionID   string   `json:"transaction_id"`
	AccountID       string   `json:"account_id"`
	Amount          string   `json:"amount"`
	Currency        string   `json:"currency"`
	TransactionType string   `json:"transaction_type"`
	RiskLevel       string   `json:"risk_level"`
	Decision        string   `json:"decision"`
	Signals         []string `json:"signals"`
	RiskScore       int      `json:"risk_score"`
}

// AssessTransaction handles POST /api/v1/fraud/assessments.
func (p *FraudProxy) AssessTransaction(w http.ResponseWriter, r *http.Request) {
	var req assessTransactionReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp assessTransactionResp
	err := p.conn.Invoke(r.Context(), "/bib.fraud.v1.FraudService/AssessTransaction", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetAssessment handles GET /api/v1/fraud/assessments/{id}.
func (p *FraudProxy) GetAssessment(w http.ResponseWriter, r *http.Request) {
	assessmentID := r.PathValue("id")
	if assessmentID == "" {
		writeError(w, http.StatusBadRequest, "assessment id is required")
		return
	}

	tenantID := ""
	if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
		tenantID = claims.TenantID.String()
	}

	req := map[string]string{
		"tenant_id":     tenantID,
		"assessment_id": assessmentID,
	}
	var resp getAssessmentResp
	err := p.conn.Invoke(r.Context(), "/bib.fraud.v1.FraudService/GetAssessment", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
