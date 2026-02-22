package proxy

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/bibbank/bib/pkg/auth"
)

// FXProxy proxies HTTP requests to the FX gRPC service.
type FXProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewFXProxy creates a new FX service proxy.
func NewFXProxy(conn *ServiceConn, logger *slog.Logger) *FXProxy {
	return &FXProxy{conn: conn, logger: logger}
}

type exchangeRateResp struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Rate          string `json:"rate"`
	Timestamp     string `json:"timestamp"`
}

type convertReq struct {
	TenantID     string `json:"tenant_id"`
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	Amount       string `json:"amount"`
}

type convertResp struct {
	OriginalAmount  string `json:"original_amount"`
	ConvertedAmount string `json:"converted_amount"`
	FromCurrency    string `json:"from_currency"`
	ToCurrency      string `json:"to_currency"`
	Rate            string `json:"rate"`
}

// GetRate handles GET /api/v1/fx/rates/{pair}.
// The pair is expected in the format "USDEUR" or "USD-EUR".
func (p *FXProxy) GetRate(w http.ResponseWriter, r *http.Request) {
	pair := r.PathValue("pair")
	if pair == "" {
		writeError(w, http.StatusBadRequest, "currency pair is required")
		return
	}

	var baseCurrency, quoteCurrency string
	if strings.Contains(pair, "-") {
		parts := strings.SplitN(pair, "-", 2)
		baseCurrency = parts[0]
		quoteCurrency = parts[1]
	} else if len(pair) == 6 {
		baseCurrency = pair[:3]
		quoteCurrency = pair[3:]
	} else {
		writeError(w, http.StatusBadRequest, "invalid currency pair format; use USDEUR or USD-EUR")
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			tenantID = claims.TenantID.String()
		}
	}

	req := map[string]string{
		"tenant_id":      tenantID,
		"base_currency":  strings.ToUpper(baseCurrency),
		"quote_currency": strings.ToUpper(quoteCurrency),
	}

	var resp exchangeRateResp
	err := p.conn.Invoke(r.Context(), "/bib.fx.v1.FXService/GetExchangeRate", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Convert handles POST /api/v1/fx/convert.
func (p *FXProxy) Convert(w http.ResponseWriter, r *http.Request) {
	var req convertReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp convertResp
	err := p.conn.Invoke(r.Context(), "/bib.fx.v1.FXService/ConvertAmount", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
