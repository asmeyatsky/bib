package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// IdentityProxy proxies HTTP requests to the identity gRPC service.
type IdentityProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewIdentityProxy creates a new identity service proxy.
func NewIdentityProxy(conn *ServiceConn, logger *slog.Logger) *IdentityProxy {
	return &IdentityProxy{conn: conn, logger: logger}
}

type initiateVerificationReq struct {
	TenantID    string `json:"tenant_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	DateOfBirth string `json:"date_of_birth"`
	Country     string `json:"country"`
}

type verificationMsg struct {
	ID                 string     `json:"id"`
	TenantID           string     `json:"tenant_id"`
	ApplicantFirstName string     `json:"applicant_first_name"`
	ApplicantLastName  string     `json:"applicant_last_name"`
	ApplicantEmail     string     `json:"applicant_email"`
	ApplicantDOB       string     `json:"applicant_dob"`
	ApplicantCountry   string     `json:"applicant_country"`
	Status             string     `json:"status"`
	CreatedAt          string     `json:"created_at"`
	UpdatedAt          string     `json:"updated_at"`
	Checks             []checkMsg `json:"checks"`
	Version            int32      `json:"version"`
}

type checkMsg struct {
	ID                string `json:"id"`
	CheckType         string `json:"check_type"`
	Status            string `json:"status"`
	Provider          string `json:"provider"`
	ProviderReference string `json:"provider_reference"`
	CompletedAt       string `json:"completed_at,omitempty"`
	FailureReason     string `json:"failure_reason,omitempty"`
}

type verificationResp struct {
	Verification verificationMsg `json:"verification"`
}

// InitiateVerification handles POST /api/v1/identity/verifications.
func (p *IdentityProxy) InitiateVerification(w http.ResponseWriter, r *http.Request) {
	var req initiateVerificationReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp verificationResp
	err := p.conn.Invoke(r.Context(), "/bib.identity.v1.IdentityService/InitiateVerification", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetVerification handles GET /api/v1/identity/verifications/{id}.
func (p *IdentityProxy) GetVerification(w http.ResponseWriter, r *http.Request) {
	verificationID := r.PathValue("id")
	if verificationID == "" {
		writeError(w, http.StatusBadRequest, "verification id is required")
		return
	}

	req := map[string]string{"id": verificationID}
	var resp verificationResp
	err := p.conn.Invoke(r.Context(), "/bib.identity.v1.IdentityService/GetVerification", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
