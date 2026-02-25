package grpc

import (
	"context"
	"log/slog"
	"regexp"
	"time"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/payment-service/internal/application/dto"
	"github.com/bibbank/bib/services/payment-service/internal/application/usecase"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// Compile-time assertion that PaymentHandler implements PaymentServiceServer.
var _ PaymentServiceServer = (*PaymentHandler)(nil)

// PaymentHandler implements the gRPC PaymentService server.
type PaymentHandler struct {
	UnimplementedPaymentServiceServer
	initiatePayment *usecase.InitiatePayment
	getPayment      *usecase.GetPayment
	listPayments    *usecase.ListPayments

	logger *slog.Logger
}

func NewPaymentHandler(
	initiatePayment *usecase.InitiatePayment,
	getPayment *usecase.GetPayment,
	listPayments *usecase.ListPayments,
	logger *slog.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		initiatePayment: initiatePayment,
		getPayment:      getPayment,
		listPayments:    listPayments,

		logger: logger}
}

// InitiatePayment implements PaymentServiceServer by delegating to HandleInitiatePayment.
func (h *PaymentHandler) InitiatePayment(ctx context.Context, req *InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
	return h.HandleInitiatePayment(ctx, req)
}

// GetPayment implements PaymentServiceServer by delegating to HandleGetPayment.
func (h *PaymentHandler) GetPayment(ctx context.Context, req *GetPaymentRequestMsg) (*GetPaymentResponseMsg, error) {
	return h.HandleGetPayment(ctx, req)
}

// ListPayments implements PaymentServiceServer by delegating to HandleListPayments.
func (h *PaymentHandler) ListPayments(ctx context.Context, req *ListPaymentsRequestMsg) (*ListPaymentsResponseMsg, error) {
	return h.HandleListPayments(ctx, req)
}

// Temporary gRPC message types until proto generation is wired.

type InitiatePaymentRequest struct {
	TenantID              string `json:"tenant_id"`
	SourceAccountID       string `json:"source_account_id"`
	DestinationAccountID  string `json:"destination_account_id,omitempty"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	RoutingNumber         string `json:"routing_number,omitempty"`
	ExternalAccountNumber string `json:"external_account_number,omitempty"`
	DestinationCountry    string `json:"destination_country,omitempty"`
	Reference             string `json:"reference,omitempty"`
	Description           string `json:"description,omitempty"`
}

type InitiatePaymentResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Rail      string `json:"rail"`
	CreatedAt string `json:"created_at"`
}

type GetPaymentRequestMsg struct {
	PaymentID string `json:"payment_id"`
}

type PaymentOrderMsg struct {
	ID                    string `json:"id"`
	TenantID              string `json:"tenant_id"`
	SourceAccountID       string `json:"source_account_id"`
	DestinationAccountID  string `json:"destination_account_id"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	RoutingNumber         string `json:"routing_number"`
	ExternalAccountNumber string `json:"external_account_number"`
	Rail                  string `json:"rail"`
	Status                string `json:"status"`
	Reference             string `json:"reference"`
	Description           string `json:"description"`
	FailureReason         string `json:"failure_reason,omitempty"`
	InitiatedAt           string `json:"initiated_at"`
	SettledAt             string `json:"settled_at,omitempty"`
	UpdatedAt             string `json:"updated_at"`
	CreatedAt             string `json:"created_at"`
	Version               int32  `json:"version"`
}

type GetPaymentResponseMsg struct {
	Payment *PaymentOrderMsg `json:"payment"`
}

type ListPaymentsRequestMsg struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	PageSize  int32  `json:"page_size"`
	Offset    int32  `json:"offset"`
}

type ListPaymentsResponseMsg struct {
	Payments   []*PaymentOrderMsg `json:"payments"`
	TotalCount int32              `json:"total_count"`
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
		h.logger.Error("handler error", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &InitiatePaymentResponse{
		ID:        result.ID.String(),
		Status:    result.Status,
		Rail:      result.Rail,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
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
		h.logger.Error("handler error", "error", err)
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
		h.logger.Error("handler error", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	var payments []*PaymentOrderMsg
	for _, p := range result.Payments {
		payments = append(payments, toPaymentOrderMsg(p))
	}

	return &ListPaymentsResponseMsg{
		Payments:   payments,
		TotalCount: int32(result.TotalCount), //nolint:gosec // bounded
	}, nil
}

func toPaymentOrderMsg(r dto.PaymentOrderResponse) *PaymentOrderMsg {
	msg := &PaymentOrderMsg{
		ID:                    r.ID.String(),
		TenantID:              r.TenantID.String(),
		SourceAccountID:       r.SourceAccountID.String(),
		DestinationAccountID:  r.DestinationAccountID.String(),
		Amount:                r.Amount.StringFixed(2),
		Currency:              r.Currency,
		Rail:                  r.Rail,
		Status:                r.Status,
		RoutingNumber:         r.RoutingNumber,
		ExternalAccountNumber: r.ExternalAccountNumber,
		Reference:             r.Reference,
		Description:           r.Description,
		FailureReason:         r.FailureReason,
		InitiatedAt:           r.InitiatedAt.Format(time.RFC3339),
		Version:               int32(r.Version), //nolint:gosec // bounded
		CreatedAt:             r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             r.UpdatedAt.Format(time.RFC3339),
	}
	if r.SettledAt != nil {
		msg.SettledAt = r.SettledAt.Format(time.RFC3339)
	}
	return msg
}
