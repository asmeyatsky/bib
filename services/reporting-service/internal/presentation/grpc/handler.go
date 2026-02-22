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

// ReportingHandler handles gRPC requests for the reporting service.
type ReportingHandler struct {
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
func (h *ReportingHandler) GenerateReport(ctx context.Context, tenantID, reportType, period string) (dto.GenerateReportResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor); err != nil {
		return dto.GenerateReportResponse{}, err
	}

	tid, err := tenantIDFromContext(ctx)
	if err != nil {
		return dto.GenerateReportResponse{}, err
	}

	req := dto.GenerateReportRequest{
		TenantID:   tid,
		ReportType: reportType,
		Period:     period,
	}

	resp, err := h.generateReport.Execute(ctx, req)
	if err != nil {
		// TODO: log original error server-side: err
		return dto.GenerateReportResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}

// GetReport handles the get report request.
func (h *ReportingHandler) GetReport(ctx context.Context, reportID string) (dto.GetReportResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return dto.GetReportResponse{}, err
	}

	id, err := uuid.Parse(reportID)
	if err != nil {
		return dto.GetReportResponse{}, fmt.Errorf("invalid report ID: %w", err)
	}

	req := dto.GetReportRequest{
		ID: id,
	}

	resp, err := h.getReport.Execute(ctx, req)
	if err != nil {
		// TODO: log original error server-side: err
		return dto.GetReportResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}

// SubmitReport handles the submit report request.
func (h *ReportingHandler) SubmitReport(ctx context.Context, reportID string) (dto.SubmitReportResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin); err != nil {
		return dto.SubmitReportResponse{}, err
	}

	id, err := uuid.Parse(reportID)
	if err != nil {
		return dto.SubmitReportResponse{}, fmt.Errorf("invalid report ID: %w", err)
	}

	req := dto.SubmitReportRequest{
		ID: id,
	}

	resp, err := h.submitReport.Execute(ctx, req)
	if err != nil {
		// TODO: log original error server-side: err
		return dto.SubmitReportResponse{}, status.Error(codes.Internal, "internal error")
	}
	return resp, nil
}
