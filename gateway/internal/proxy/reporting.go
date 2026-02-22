package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// ReportingProxy proxies HTTP requests to the reporting gRPC service.
type ReportingProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewReportingProxy creates a new reporting service proxy.
func NewReportingProxy(conn *ServiceConn, logger *slog.Logger) *ReportingProxy {
	return &ReportingProxy{conn: conn, logger: logger}
}

type generateReportReq struct {
	TenantID   string `json:"tenant_id"`
	ReportType string `json:"report_type"`
	Period     string `json:"period"`
}

type generateReportResp struct {
	ReportID  string `json:"report_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type getReportResp struct {
	ReportID   string `json:"report_id"`
	TenantID   string `json:"tenant_id"`
	ReportType string `json:"report_type"`
	Period     string `json:"period"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type submitReportResp struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
}

// GenerateReport handles POST /api/v1/reports.
func (p *ReportingProxy) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req generateReportReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp generateReportResp
	err := p.conn.Invoke(r.Context(), "/bib.reporting.v1.ReportingService/GenerateReport", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetReport handles GET /api/v1/reports/{id}.
func (p *ReportingProxy) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("id")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "report id is required")
		return
	}

	req := map[string]string{"report_id": reportID}
	var resp getReportResp
	err := p.conn.Invoke(r.Context(), "/bib.reporting.v1.ReportingService/GetReport", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SubmitReport handles POST /api/v1/reports/{id}/submit.
func (p *ReportingProxy) SubmitReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("id")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "report id is required")
		return
	}

	req := map[string]string{"report_id": reportID}
	var resp submitReportResp
	err := p.conn.Invoke(r.Context(), "/bib.reporting.v1.ReportingService/SubmitReport", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
