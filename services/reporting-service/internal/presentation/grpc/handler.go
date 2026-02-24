package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/application/usecase"
)

// requireRole checks that the caller has at least one of the given roles.
func requireRole(ctx context.Context, roles ...string) error {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	for _, role := range roles {
		if claims.HasRole(role) {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "insufficient permissions")
}

// tenantIDFromContext extracts the tenant ID from JWT claims in the context.
func tenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return claims.TenantID, nil
}

// ---------------------------------------------------------------------------
// Request / Response types (stand-in for proto-generated messages)
// ---------------------------------------------------------------------------

// GenerateReportRequest represents the proto GenerateReportRequest message.
type GenerateReportRequest struct {
	TenantID   string `json:"tenant_id"`
	ReportType string `json:"report_type"`
	Period     string `json:"period"`
}

// GenerateReportResponse represents the proto GenerateReportResponse message.
type GenerateReportResponse struct {
	ReportID  string `json:"report_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GetReportRequest represents the proto GetReportRequest message.
type GetReportRequest struct {
	ReportID string `json:"report_id"`
}

// GetReportResponse represents the proto GetReportResponse message.
type GetReportResponse struct {
	ReportID   string `json:"report_id"`
	TenantID   string `json:"tenant_id"`
	ReportType string `json:"report_type"`
	Period     string `json:"period"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// SubmitReportRequest represents the proto SubmitReportRequest message.
type SubmitReportRequest struct {
	ReportID string `json:"report_id"`
}

// SubmitReportResponse represents the proto SubmitReportResponse message.
type SubmitReportResponse struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
}

// ---------------------------------------------------------------------------
// ReportingHandler
// ---------------------------------------------------------------------------

// ReportingHandler handles gRPC requests for the reporting service.
type ReportingHandler struct {
	UnimplementedReportingServiceServer
	generateReport *usecase.GenerateReportUseCase
	getReport      *usecase.GetReportUseCase
	submitReport   *usecase.SubmitReportUseCase
}

// NewReportingHandler creates a new ReportingHandler.
func NewReportingHandler(
	generateReport *usecase.GenerateReportUseCase,
	getReport *usecase.GetReportUseCase,
	submitReport *usecase.SubmitReportUseCase,
) *ReportingHandler {
	return &ReportingHandler{
		generateReport: generateReport,
		getReport:      getReport,
		submitReport:   submitReport,
	}
}

// GenerateReport handles the generate report request.
func (h *ReportingHandler) GenerateReport(ctx context.Context, req *GenerateReportRequest) (*GenerateReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor); err != nil {
		return nil, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	dtoReq := dto.GenerateReportRequest{
		TenantID:   tid,
		ReportType: req.ReportType,
		Period:     req.Period,
	}

	result, err := h.generateReport.Execute(ctx, dtoReq)
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &GenerateReportResponse{
		ReportID:  result.ID.String(),
		Status:    result.Status,
		CreatedAt: result.GeneratedAt,
	}, nil
}

// GetReport handles the get report request.
func (h *ReportingHandler) GetReport(ctx context.Context, req *GetReportRequest) (*GetReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.ReportID)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID: %w", err)
	}

	dtoReq := dto.GetReportRequest{
		ID: id,
	}

	result, err := h.getReport.Execute(ctx, dtoReq)
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &GetReportResponse{
		ReportID:   result.ID.String(),
		TenantID:   result.TenantID.String(),
		ReportType: result.ReportType,
		Period:     result.ReportingPeriod,
		Status:     result.Status,
		CreatedAt:  result.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  result.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// SubmitReport handles the submit report request.
func (h *ReportingHandler) SubmitReport(ctx context.Context, req *SubmitReportRequest) (*SubmitReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if err := requireRole(ctx, auth.RoleAdmin); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.ReportID)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID: %w", err)
	}

	dtoReq := dto.SubmitReportRequest{
		ID: id,
	}

	result, err := h.submitReport.Execute(ctx, dtoReq)
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &SubmitReportResponse{
		ReportID: result.ID.String(),
		Status:   result.Status,
	}, nil
}
