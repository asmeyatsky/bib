package grpc

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
)

var currencyCodeRE = regexp.MustCompile(`^[A-Z]{3}$`)

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

// PaymentHandler implements the gRPC PaymentService server.
type PaymentHandler struct {
	initiatePayment *usecase.InitiatePayment
	getPayment      *usecase.GetPayment
	listPayments    *usecase.ListPayments
}

func NewPaymentHandler(
	initiatePayment *usecase.InitiatePayment,
	getPayment *usecase.GetPayment,
	listPayments *usecase.ListPayments,
) *PaymentHandler {
	return &PaymentHandler{
		initiatePayment: initiatePayment,
		getPayment:      getPayment,
		listPayments:    listPayments,
	}
}

// Temporary gRPC message types until proto generation is wired.

type InitiatePaymentRequest struct {
	TenantID              string
	SourceAccountID       string
	DestinationAccountID  string
	Amount                string
	Currency              string
	RoutingNumber         string
	ExternalAccountNumber string
	DestinationCountry    string
	Reference             string
	Description           string
}

type InitiatePaymentResponse struct {
	ID        string
	Status    string
	Rail      string
	CreatedAt *timestamppb.Timestamp
}

type GetPaymentRequestMsg struct {
	PaymentID string
}

type PaymentOrderMsg struct {
	ID                    string
	TenantID              string
	SourceAccountID       string
	DestinationAccountID  string
	Amount                string
	Currency              string
	Rail                  string
	Status                string
	RoutingNumber         string
	ExternalAccountNumber string
	Reference             string
	Description           string
	FailureReason         string
	InitiatedAt           *timestamppb.Timestamp
	SettledAt             *timestamppb.Timestamp
	Version               int32
	CreatedAt             *timestamppb.Timestamp
	UpdatedAt             *timestamppb.Timestamp
}

type GetPaymentResponseMsg struct {
	Payment *PaymentOrderMsg
}

type ListPaymentsRequestMsg struct {
	TenantID  string
	AccountID string
	PageSize  int32
	Offset    int32
}

type ListPaymentsResponseMsg struct {
	Payments   []*PaymentOrderMsg
	TotalCount int32
}

func (h *PaymentHandler) HandleInitiatePayment(ctx context.Context, req *InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
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

	sourceAcctID, err := uuid.Parse(req.SourceAccountID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source_account_id: %v", err)
	}

	var destAcctID uuid.UUID
	if req.DestinationAccountID != "" {
		destAcctID, err = uuid.Parse(req.DestinationAccountID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid destination_account_id: %v", err)
		}
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amount.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}
	if !currencyCodeRE.MatchString(req.Currency) {
		return nil, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}

	result, err := h.initiatePayment.Execute(ctx, dto.InitiatePaymentRequest{
		TenantID:              tenantID,
		SourceAccountID:       sourceAcctID,
		DestinationAccountID:  destAcctID,
		Amount:                amount,
		Currency:              req.Currency,
		RoutingNumber:         req.RoutingNumber,
		ExternalAccountNumber: req.ExternalAccountNumber,
		DestinationCountry:    req.DestinationCountry,
		Reference:             req.Reference,
		Description:           req.Description,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &InitiatePaymentResponse{
		ID:        result.ID.String(),
		Status:    result.Status,
		Rail:      result.Rail,
		CreatedAt: timestamppb.New(result.CreatedAt),
	}, nil
}

func (h *PaymentHandler) HandleGetPayment(ctx context.Context, req *GetPaymentRequestMsg) (*GetPaymentResponseMsg, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	paymentID, err := uuid.Parse(req.PaymentID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment_id: %v", err)
	}

	result, err := h.getPayment.Execute(ctx, dto.GetPaymentRequest{
		PaymentID: paymentID,
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetPaymentResponseMsg{
		Payment: toPaymentOrderMsg(result),
	}, nil
}

func (h *PaymentHandler) HandleListPayments(ctx context.Context, req *ListPaymentsRequestMsg) (*ListPaymentsResponseMsg, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize < 0 || pageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page_size must be between 1 and 100")
	}
	if req.Offset < 0 {
		return nil, status.Error(codes.InvalidArgument, "offset must be >= 0")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var accountID uuid.UUID
	if req.AccountID != "" {
		accountID, err = uuid.Parse(req.AccountID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid account_id: %v", err)
		}
	}

	result, err := h.listPayments.Execute(ctx, dto.ListPaymentsRequest{
		TenantID:  tenantID,
		AccountID: accountID,
		PageSize:  int(pageSize),
		Offset:    int(req.Offset),
	})
	if err != nil {
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}

	var payments []*PaymentOrderMsg
	for _, p := range result.Payments {
		payments = append(payments, toPaymentOrderMsg(p))
	}

	return &ListPaymentsResponseMsg{
		Payments:   payments,
		TotalCount: int32(result.TotalCount),
	}, nil
}

func toPaymentOrderMsg(r dto.PaymentOrderResponse) *PaymentOrderMsg {
	msg := &PaymentOrderMsg{
		ID:                    r.ID.String(),
		TenantID:              r.TenantID.String(),
		SourceAccountID:       r.SourceAccountID.String(),
		DestinationAccountID:  r.DestinationAccountID.String(),
		Amount:                r.Amount.String(),
		Currency:              r.Currency,
		Rail:                  r.Rail,
		Status:                r.Status,
		RoutingNumber:         r.RoutingNumber,
		ExternalAccountNumber: r.ExternalAccountNumber,
		Reference:             r.Reference,
		Description:           r.Description,
		FailureReason:         r.FailureReason,
		InitiatedAt:           timestamppb.New(r.InitiatedAt),
		Version:               int32(r.Version),
		CreatedAt:             timestamppb.New(r.CreatedAt),
		UpdatedAt:             timestamppb.New(r.UpdatedAt),
	}
	if r.SettledAt != nil {
		msg.SettledAt = timestamppb.New(*r.SettledAt)
	}
	return msg
}
