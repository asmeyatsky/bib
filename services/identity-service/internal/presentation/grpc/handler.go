package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/application/usecase"
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

// IdentityHandler implements the gRPC IdentityService server.
type IdentityHandler struct {
	initiateVerification *usecase.InitiateVerification
	getVerification      *usecase.GetVerification
	completeCheck        *usecase.CompleteCheck
	listVerifications    *usecase.ListVerifications
}

func NewIdentityHandler(
	initiateVerification *usecase.InitiateVerification,
	getVerification *usecase.GetVerification,
	completeCheck *usecase.CompleteCheck,
	listVerifications *usecase.ListVerifications,
) *IdentityHandler {
	return &IdentityHandler{
		initiateVerification: initiateVerification,
		getVerification:      getVerification,
		completeCheck:        completeCheck,
		listVerifications:    listVerifications,
	}
}

// Temporary gRPC message types until proto generation is wired.

type InitiateVerificationRequest struct {
	TenantID    string
	FirstName   string
	LastName    string
	Email       string
	DateOfBirth string
	Country     string
}

type InitiateVerificationResponse struct {
	Verification *VerificationMsg
}

type GetVerificationRequest struct {
	ID string
}

type GetVerificationResponse struct {
	Verification *VerificationMsg
}

type CompleteCheckRequest struct {
	VerificationID string
	CheckID        string
	Status         string
	FailureReason  string
}

type CompleteCheckResponse struct {
	Verification *VerificationMsg
}

type VerificationMsg struct {
	CreatedAt          *timestamppb.Timestamp
	UpdatedAt          *timestamppb.Timestamp
	ID                 string
	TenantID           string
	ApplicantFirstName string
	ApplicantLastName  string
	ApplicantEmail     string
	ApplicantDOB       string
	ApplicantCountry   string
	Status             string
	Checks             []*CheckMsg
	Version            int32
}

type CheckMsg struct {
	CompletedAt       *timestamppb.Timestamp
	ID                string
	CheckType         string
	Status            string
	Provider          string
	ProviderReference string
	FailureReason     string
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
		// TODO: log original error server-side: err
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
		// TODO: log original error server-side: err
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
		// TODO: log original error server-side: err
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
			cm.CompletedAt = timestamppb.New(*c.CompletedAt)
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
		CreatedAt:          timestamppb.New(r.CreatedAt),
		UpdatedAt:          timestamppb.New(r.UpdatedAt),
	}
}
