package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bibbank/bib/services/reporting-service/internal/application/dto"
	"github.com/bibbank/bib/services/reporting-service/internal/application/usecase"
)

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
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return dto.GenerateReportResponse{}, fmt.Errorf("invalid tenant ID: %w", err)
	}

	req := dto.GenerateReportRequest{
		TenantID:   tid,
		ReportType: reportType,
		Period:     period,
	}

	return h.generateReport.Execute(ctx, req)
}

// GetReport handles the get report request.
func (h *ReportingHandler) GetReport(ctx context.Context, reportID string) (dto.GetReportResponse, error) {
	id, err := uuid.Parse(reportID)
	if err != nil {
		return dto.GetReportResponse{}, fmt.Errorf("invalid report ID: %w", err)
	}

	req := dto.GetReportRequest{
		ID: id,
	}

	return h.getReport.Execute(ctx, req)
}

// SubmitReport handles the submit report request.
func (h *ReportingHandler) SubmitReport(ctx context.Context, reportID string) (dto.SubmitReportResponse, error) {
	id, err := uuid.Parse(reportID)
	if err != nil {
		return dto.SubmitReportResponse{}, fmt.Errorf("invalid report ID: %w", err)
	}

	req := dto.SubmitReportRequest{
		ID: id,
	}

	return h.submitReport.Execute(ctx, req)
}
