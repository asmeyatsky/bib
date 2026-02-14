package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/fx-service/internal/application/dto"
	"github.com/bibbank/bib/services/fx-service/internal/application/usecase"
)

// Handler implements the FX service gRPC methods.
// In production this would implement a generated protobuf service interface;
// for now it provides typed methods that can be registered on a gRPC server.
type Handler struct {
	getRate    *usecase.GetExchangeRate
	convert    *usecase.ConvertAmount
	revaluate  *usecase.Revaluate
	logger     *slog.Logger
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

// GetExchangeRate returns the current exchange rate for a currency pair.
func (h *Handler) GetExchangeRate(ctx context.Context, tenantID, baseCurrency, quoteCurrency string) (dto.ExchangeRateResponse, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return dto.ExchangeRateResponse{}, fmt.Errorf("invalid tenant ID: %w", err)
	}

	req := dto.GetExchangeRateRequest{
		TenantID:      tid,
		BaseCurrency:  baseCurrency,
		QuoteCurrency: quoteCurrency,
	}

	resp, err := h.getRate.Execute(ctx, req)
	if err != nil {
		h.logger.Error("GetExchangeRate failed", "error", err, "pair", baseCurrency+"/"+quoteCurrency)
		return dto.ExchangeRateResponse{}, err
	}

	h.logger.Info("GetExchangeRate succeeded", "pair", baseCurrency+"/"+quoteCurrency, "rate", resp.Rate.String())
	return resp, nil
}

// ConvertAmount converts an amount between two currencies.
func (h *Handler) ConvertAmount(ctx context.Context, tenantID, fromCurrency, toCurrency, amount string) (dto.ConvertAmountResponse, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return dto.ConvertAmountResponse{}, fmt.Errorf("invalid tenant ID: %w", err)
	}

	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return dto.ConvertAmountResponse{}, fmt.Errorf("invalid amount %q: %w", amount, err)
	}

	req := dto.ConvertAmountRequest{
		TenantID:     tid,
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Amount:       amt,
	}

	resp, err := h.convert.Execute(ctx, req)
	if err != nil {
		h.logger.Error("ConvertAmount failed", "error", err, "from", fromCurrency, "to", toCurrency)
		return dto.ConvertAmountResponse{}, err
	}

	h.logger.Info("ConvertAmount succeeded",
		"from", fromCurrency, "to", toCurrency,
		"original", resp.OriginalAmount.String(),
		"converted", resp.ConvertedAmount.String(),
	)
	return resp, nil
}

// Revaluate runs an ASC 830 FX revaluation.
func (h *Handler) Revaluate(ctx context.Context, tenantID, functionalCurrency string, positions []dto.ForeignCurrencyPositionDTO) (dto.RevaluateResponse, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return dto.RevaluateResponse{}, fmt.Errorf("invalid tenant ID: %w", err)
	}

	req := dto.RevaluateRequest{
		TenantID:           tid,
		FunctionalCurrency: functionalCurrency,
		Positions:          positions,
	}

	resp, err := h.revaluate.Execute(ctx, req)
	if err != nil {
		h.logger.Error("Revaluate failed", "error", err, "tenant", tenantID)
		return dto.RevaluateResponse{}, err
	}

	h.logger.Info("Revaluate succeeded",
		"tenant", tenantID,
		"functional_currency", functionalCurrency,
		"total_gain_loss", resp.TotalGainLoss.String(),
		"entries", len(resp.Entries),
	)
	return resp, nil
}
