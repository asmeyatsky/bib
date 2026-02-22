package grpc

import (
	"context"
	"log/slog"
	"regexp"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/application/usecase"
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

// Compile-time assertion that Handler implements FXServiceServer.
var _ FXServiceServer = (*Handler)(nil)

// Handler implements the FXServiceServer gRPC interface.
type Handler struct {
	UnimplementedFXServiceServer
	getRate   *usecase.GetExchangeRate
	convert   *usecase.ConvertAmount
	revaluate *usecase.Revaluate
	logger    *slog.Logger
}

// NewHandler creates a new gRPC Handler.
func NewHandler(
	getRate *usecase.GetExchangeRate,
	convert *usecase.ConvertAmount,
	revaluate *usecase.Revaluate,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		getRate:   getRate,
		convert:   convert,
		revaluate: revaluate,
		logger:    logger,
	}
}

// Proto-aligned request/response message types.

// MoneyMsg represents the proto Money message.
type MoneyMsg struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// ExchangeRateMsg represents the proto ExchangeRate message.
type ExchangeRateMsg struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Rate          string `json:"rate"`
	InverseRate   string `json:"inverse_rate"`
	Provider      string `json:"provider"`
}

// GetExchangeRateRequest represents the proto GetExchangeRateRequest message.
type GetExchangeRateRequest struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
}

// GetExchangeRateResponse represents the proto GetExchangeRateResponse message.
type GetExchangeRateResponse struct {
	Rate *ExchangeRateMsg `json:"rate"`
}

// ConvertAmountRequest represents the proto ConvertAmountRequest message.
type ConvertAmountRequest struct {
	Amount         *MoneyMsg `json:"amount"`
	TargetCurrency string    `json:"target_currency"`
}

// ConvertAmountResponse represents the proto ConvertAmountResponse message.
type ConvertAmountResponse struct {
	ConvertedAmount *MoneyMsg        `json:"converted_amount"`
	RateUsed        *ExchangeRateMsg `json:"rate_used"`
}

// ListExchangeRatesRequest represents the proto ListExchangeRatesRequest message.
type ListExchangeRatesRequest struct {
	BaseCurrency string `json:"base_currency"`
	PageSize     int32  `json:"page_size"`
	PageToken    string `json:"page_token"`
}

// ListExchangeRatesResponse represents the proto ListExchangeRatesResponse message.
type ListExchangeRatesResponse struct {
	Rates         []*ExchangeRateMsg `json:"rates"`
	NextPageToken string             `json:"next_page_token"`
	TotalCount    int32              `json:"total_count"`
}

// RevaluateRequest represents the proto RevaluateRequest message.
type RevaluateRequest struct {
	TenantID           string `json:"tenant_id"`
	AsOfDate           string `json:"as_of_date"`
	FunctionalCurrency string `json:"functional_currency"`
}

// RevaluateResponse represents the proto RevaluateResponse message.
type RevaluateResponse struct {
	AccountsProcessed int32     `json:"accounts_processed"`
	TotalGainLoss     *MoneyMsg `json:"total_gain_loss"`
}

// GetExchangeRate returns the current exchange rate for a currency pair.
func (h *Handler) GetExchangeRate(ctx context.Context, req *GetExchangeRateRequest) (*GetExchangeRateResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.BaseCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "base_currency is required")
	}
	if !currencyCodeRE.MatchString(req.BaseCurrency) {
		return nil, status.Error(codes.InvalidArgument, "base_currency must be a 3-letter uppercase ISO code")
	}
	if req.QuoteCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "quote_currency is required")
	}
	if !currencyCodeRE.MatchString(req.QuoteCurrency) {
		return nil, status.Error(codes.InvalidArgument, "quote_currency must be a 3-letter uppercase ISO code")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	dtoReq := dto.GetExchangeRateRequest{
		TenantID:      tenantID,
		BaseCurrency:  req.BaseCurrency,
		QuoteCurrency: req.QuoteCurrency,
	}

	resp, err := h.getRate.Execute(ctx, dtoReq)
	if err != nil {
		h.logger.Error("GetExchangeRate failed", "error", err, "pair", req.BaseCurrency+"/"+req.QuoteCurrency)
		return nil, status.Error(codes.Internal, "internal error")
	}

	h.logger.Info("GetExchangeRate succeeded", "pair", req.BaseCurrency+"/"+req.QuoteCurrency, "rate", resp.Rate.String())
	return &GetExchangeRateResponse{
		Rate: &ExchangeRateMsg{
			ID:            resp.ID.String(),
			BaseCurrency:  resp.BaseCurrency,
			QuoteCurrency: resp.QuoteCurrency,
			Rate:          resp.Rate.String(),
			InverseRate:   resp.InverseRate.String(),
			Provider:      resp.Provider,
		},
	}, nil
}

// ConvertAmount converts an amount between two currencies.
func (h *Handler) ConvertAmount(ctx context.Context, req *ConvertAmountRequest) (*ConvertAmountResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.Amount == nil {
		return nil, status.Error(codes.InvalidArgument, "amount is required")
	}

	amt, err := decimal.NewFromString(req.Amount.Amount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}
	if !amt.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	fromCurrency := req.Amount.Currency
	if fromCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "source currency is required")
	}
	if !currencyCodeRE.MatchString(fromCurrency) {
		return nil, status.Error(codes.InvalidArgument, "source currency must be a 3-letter uppercase ISO code")
	}
	if req.TargetCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "target_currency is required")
	}
	if !currencyCodeRE.MatchString(req.TargetCurrency) {
		return nil, status.Error(codes.InvalidArgument, "target_currency must be a 3-letter uppercase ISO code")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	dtoReq := dto.ConvertAmountRequest{
		TenantID:     tenantID,
		FromCurrency: fromCurrency,
		ToCurrency:   req.TargetCurrency,
		Amount:       amt,
	}

	resp, err := h.convert.Execute(ctx, dtoReq)
	if err != nil {
		h.logger.Error("ConvertAmount failed", "error", err, "from", fromCurrency, "to", req.TargetCurrency)
		return nil, status.Error(codes.Internal, "internal error")
	}

	h.logger.Info("ConvertAmount succeeded",
		"from", fromCurrency, "to", req.TargetCurrency,
		"original", resp.OriginalAmount.String(),
		"converted", resp.ConvertedAmount.String(),
	)
	return &ConvertAmountResponse{
		ConvertedAmount: &MoneyMsg{
			Amount:   resp.ConvertedAmount.String(),
			Currency: req.TargetCurrency,
		},
		RateUsed: &ExchangeRateMsg{
			BaseCurrency:  resp.FromCurrency,
			QuoteCurrency: resp.ToCurrency,
			Rate:          resp.Rate.String(),
		},
	}, nil
}

// ListExchangeRates returns available exchange rates.
func (h *Handler) ListExchangeRates(ctx context.Context, req *ListExchangeRatesRequest) (*ListExchangeRatesResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.PageSize < 0 || req.PageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page_size must be between 1 and 100")
	}

	// ListExchangeRates not yet implemented in use case layer.
	return nil, status.Errorf(codes.Unimplemented, "ListExchangeRates not yet implemented")
}

// Revaluate runs an ASC 830 FX revaluation.
func (h *Handler) Revaluate(ctx context.Context, req *RevaluateRequest) (*RevaluateResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.FunctionalCurrency == "" {
		return nil, status.Error(codes.InvalidArgument, "functional_currency is required")
	}
	if !currencyCodeRE.MatchString(req.FunctionalCurrency) {
		return nil, status.Error(codes.InvalidArgument, "functional_currency must be a 3-letter uppercase ISO code")
	}

	dtoReq := dto.RevaluateRequest{
		TenantID:           tenantID,
		FunctionalCurrency: req.FunctionalCurrency,
		Positions:          nil, // Positions loaded by use case from repository
	}

	resp, err := h.revaluate.Execute(ctx, dtoReq)
	if err != nil {
		h.logger.Error("Revaluate failed", "error", err, "tenant", tenantID.String())
		return nil, status.Error(codes.Internal, "internal error")
	}

	h.logger.Info("Revaluate succeeded",
		"tenant", tenantID.String(),
		"functional_currency", req.FunctionalCurrency,
		"total_gain_loss", resp.TotalGainLoss.String(),
		"entries", len(resp.Entries),
	)
	return &RevaluateResponse{
		AccountsProcessed: int32(len(resp.Entries)),
		TotalGainLoss: &MoneyMsg{
			Amount:   resp.TotalGainLoss.String(),
			Currency: req.FunctionalCurrency,
		},
	}, nil
}
