package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// DepositProxy proxies HTTP requests to the deposit gRPC service.
type DepositProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewDepositProxy creates a new deposit service proxy.
func NewDepositProxy(conn *ServiceConn, logger *slog.Logger) *DepositProxy {
	return &DepositProxy{conn: conn, logger: logger}
}

type interestTier struct {
	MinBalance string `json:"min_balance"`
	MaxBalance string `json:"max_balance"`
	RateBps    int32  `json:"rate_bps"`
}

type createProductReq struct {
	TenantID string         `json:"tenant_id"`
	Name     string         `json:"name"`
	Currency string         `json:"currency"`
	Tiers    []interestTier `json:"tiers"`
	TermDays int32          `json:"term_days"`
}

type depositProductMsg struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Name      string         `json:"name"`
	Currency  string         `json:"currency"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Tiers     []interestTier `json:"tiers"`
	TermDays  int32          `json:"term_days"`
	Version   int32          `json:"version"`
	IsActive  bool           `json:"is_active"`
}

type createProductResp struct {
	Product depositProductMsg `json:"product"`
}

type openPositionReq struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	ProductID string `json:"product_id"`
	Principal string `json:"principal"`
}

type depositPositionMsg struct {
	AccruedInterest string `json:"accrued_interest"`
	CreatedAt       string `json:"created_at"`
	AccountID       string `json:"account_id"`
	ProductID       string `json:"product_id"`
	Principal       string `json:"principal"`
	Currency        string `json:"currency"`
	OpenedAt        string `json:"opened_at"`
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	MaturityDate    string `json:"maturity_date,omitempty"`
	LastAccrualDate string `json:"last_accrual_date"`
	UpdatedAt       string `json:"updated_at"`
	Status          string `json:"status"`
	Version         int32  `json:"version"`
}

type openPositionResp struct {
	Position depositPositionMsg `json:"position"`
}

type getPositionResp struct {
	Position depositPositionMsg `json:"position"`
}

// CreateProduct handles POST /api/v1/deposits/products.
func (p *DepositProxy) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp createProductResp
	err := p.conn.Invoke(r.Context(), "/bib.deposit.v1.DepositService/CreateProduct", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// OpenPosition handles POST /api/v1/deposits/positions.
func (p *DepositProxy) OpenPosition(w http.ResponseWriter, r *http.Request) {
	var req openPositionReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp openPositionResp
	err := p.conn.Invoke(r.Context(), "/bib.deposit.v1.DepositService/OpenPosition", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetPosition handles GET /api/v1/deposits/positions/{id}.
func (p *DepositProxy) GetPosition(w http.ResponseWriter, r *http.Request) {
	positionID := r.PathValue("id")
	if positionID == "" {
		writeError(w, http.StatusBadRequest, "position id is required")
		return
	}

	req := map[string]string{"id": positionID}
	var resp getPositionResp
	err := p.conn.Invoke(r.Context(), "/bib.deposit.v1.DepositService/GetPosition", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
