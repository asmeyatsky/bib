package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/application/usecase"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// Compile-time assertion that IdentityHandler implements IdentityServiceServer.
var _ IdentityServiceServer = (*IdentityHandler)(nil)

// IdentityHandler implements the gRPC IdentityService server.
type IdentityHandler struct {
	UnimplementedIdentityServiceServer
	initiateVerification *usecase.InitiateVerification
	getVerification      *usecase.GetVerification
	completeCheck        *usecase.CompleteCheck
	listVerifications    *usecase.ListVerifications
	logger               *slog.Logger
}

func NewIdentityHandler(
	initiateVerification *usecase.InitiateVerification,
	getVerification *usecase.GetVerification,
	completeCheck *usecase.CompleteCheck,
	listVerifications *usecase.ListVerifications,
	logger *slog.Logger,
) *IdentityHandler {
	return &IdentityHandler{
		initiateVerification: initiateVerification,
		getVerification:      getVerification,
		completeCheck:        completeCheck,
		listVerifications:    listVerifications,
		logger:               logger,
	}
}

// InitiateVerification implements IdentityServiceServer by delegating to HandleInitiateVerification.
func (h *IdentityHandler) InitiateVerification(ctx context.Context, req *InitiateVerificationRequest) (*InitiateVerificationResponse, error) {
	return h.HandleInitiateVerification(ctx, req)
}

// GetVerification implements IdentityServiceServer by delegating to HandleGetVerification.
func (h *IdentityHandler) GetVerification(ctx context.Context, req *GetVerificationRequest) (*GetVerificationResponse, error) {
	return h.HandleGetVerification(ctx, req)
}

// CompleteCheck implements IdentityServiceServer by delegating to HandleCompleteCheck.
func (h *IdentityHandler) CompleteCheck(ctx context.Context, req *CompleteCheckRequest) (*CompleteCheckResponse, error) {
	return h.HandleCompleteCheck(ctx, req)
}

// Temporary gRPC message types until proto generation is wired.

type InitiateVerificationRequest struct {
	TenantID    string `json:"tenant_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	DateOfBirth string `json:"date_of_birth"`
	Country     string `json:"country"`
}

type InitiateVerificationResponse struct {
	Verification *VerificationMsg `json:"verification"`
}

type GetVerificationRequest struct {
	ID string `json:"id"`
}

type GetVerificationResponse struct {
	Verification *VerificationMsg `json:"verification"`
}

type CompleteCheckRequest struct {
	VerificationID string `json:"verification_id"`
	CheckID        string `json:"check_id"`
	Status         string `json:"status"`
	FailureReason  string `json:"failure_reason"`
}

type CompleteCheckResponse struct {
	Verification *VerificationMsg `json:"verification"`
}

type VerificationMsg struct {
	ID                 string      `json:"id"`
	TenantID           string      `json:"tenant_id"`
	ApplicantFirstName string      `json:"applicant_first_name"`
	ApplicantLastName  string      `json:"applicant_last_name"`
	ApplicantEmail     string      `json:"applicant_email"`
	ApplicantDOB       string      `json:"applicant_dob"`
	ApplicantCountry   string      `json:"applicant_country"`
	Status             string      `json:"status"`
	CreatedAt          string      `json:"created_at"`
	UpdatedAt          string      `json:"updated_at"`
	Checks             []*CheckMsg `json:"checks"`
	Version            int32       `json:"version"`
}

type CheckMsg struct {
	ID                string `json:"id"`
	CheckType         string `json:"check_type"`
	Status            string `json:"status"`
	Provider          string `json:"provider"`
	ProviderReference string `json:"provider_reference"`
	CompletedAt       string `json:"completed_at,omitempty"`
	FailureReason     string `json:"failure_reason,omitempty"`
}

func (h *IdentityHandler) HandleInitiateVerification(ctx context.Context, req *InitiateVerificationRequest) (*InitiateVerificationResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	result, err := h.initiateVerification.Execute(ctx, dto.InitiateVerificationRequest{
		TenantID:    tenantID,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		DateOfBirth: req.DateOfBirth,
		Country:     req.Country,
	})
	if err != nil {
		h.logger.Error("initiate verification failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &InitiateVerificationResponse{
		Verification: toVerificationMsg(result),
	}, nil
}

func (h *IdentityHandler) HandleGetVerification(ctx context.Context, req *GetVerificationRequest) (*GetVerificationResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	result, err := h.getVerification.Execute(ctx, dto.GetVerificationRequest{
		ID: id,
	})
	if err != nil {
		h.logger.Error("get verification failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetVerificationResponse{
		Verification: toVerificationMsg(result),
	}, nil
}

func (h *IdentityHandler) HandleCompleteCheck(ctx context.Context, req *CompleteCheckRequest) (*CompleteCheckResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	verificationID, err := uuid.Parse(req.VerificationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid verification_id: %v", err)
	}

	checkID, err := uuid.Parse(req.CheckID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid check_id: %v", err)
	}

	result, err := h.completeCheck.Execute(ctx, dto.CompleteCheckRequest{
		VerificationID: verificationID,
		CheckID:        checkID,
		Status:         req.Status,
		FailureReason:  req.FailureReason,
	})
	if err != nil {
		h.logger.Error("handler error", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &CompleteCheckResponse{
		Verification: toVerificationMsg(result),
	}, nil
}

func toVerificationMsg(r dto.VerificationResponse) *VerificationMsg {
	var checks []*CheckMsg
	for _, c := range r.Checks {
		cm := &CheckMsg{
			ID:                c.ID.String(),
			CheckType:         c.CheckType,
			Status:            c.Status,
			Provider:          c.Provider,
			ProviderReference: c.ProviderReference,
			FailureReason:     c.FailureReason,
		}
		if c.CompletedAt != nil {
			cm.CompletedAt = c.CompletedAt.Format(time.RFC3339)
		}
		checks = append(checks, cm)
	}

	return &VerificationMsg{
		ID:                 r.ID.String(),
		TenantID:           r.TenantID.String(),
		ApplicantFirstName: r.ApplicantFirstName,
		ApplicantLastName:  r.ApplicantLastName,
		ApplicantEmail:     r.ApplicantEmail,
		ApplicantDOB:       r.ApplicantDOB,
		ApplicantCountry:   r.ApplicantCountry,
		Status:             r.Status,
		Checks:             checks,
		Version:            int32(r.Version), //nolint:gosec
		CreatedAt:          r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          r.UpdatedAt.Format(time.RFC3339),
	}
}
