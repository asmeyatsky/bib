package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// PaymentProxy proxies HTTP requests to the payment gRPC service.
type PaymentProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewPaymentProxy creates a new payment service proxy.
func NewPaymentProxy(conn *ServiceConn, logger *slog.Logger) *PaymentProxy {
	return &PaymentProxy{conn: conn, logger: logger}
}

type initiatePaymentReq struct {
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

type initiatePaymentResp struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Rail      string `json:"rail"`
	CreatedAt string `json:"created_at"`
}

type paymentOrderMsg struct {
	ID                    string `json:"id"`
	TenantID              string `json:"tenant_id"`
	SourceAccountID       string `json:"source_account_id"`
	DestinationAccountID  string `json:"destination_account_id"`
	Amount                string `json:"amount"`
	Currency              string `json:"currency"`
	Rail                  string `json:"rail"`
	Status                string `json:"status"`
	RoutingNumber         string `json:"routing_number"`
	ExternalAccountNumber string `json:"external_account_number"`
	Reference             string `json:"reference"`
	Description           string `json:"description"`
	FailureReason         string `json:"failure_reason,omitempty"`
	InitiatedAt           string `json:"initiated_at"`
	SettledAt             string `json:"settled_at,omitempty"`
	Version               int32  `json:"version"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type getPaymentResp struct {
	Payment paymentOrderMsg `json:"payment"`
}

type listPaymentsResp struct {
	Payments   []paymentOrderMsg `json:"payments"`
	TotalCount int32             `json:"total_count"`
}

// InitiatePayment handles POST /api/v1/payments.
func (p *PaymentProxy) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	var req initiatePaymentReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp initiatePaymentResp
	err := p.conn.Invoke(r.Context(), "/bib.payment.v1.PaymentService/InitiatePayment", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetPayment handles GET /api/v1/payments/{id}.
func (p *PaymentProxy) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := r.PathValue("id")
	if paymentID == "" {
		writeError(w, http.StatusBadRequest, "payment id is required")
		return
	}

	req := map[string]string{"payment_id": paymentID}
	var resp getPaymentResp
	err := p.conn.Invoke(r.Context(), "/bib.payment.v1.PaymentService/GetPayment", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListPayments handles GET /api/v1/payments.
func (p *PaymentProxy) ListPayments(w http.ResponseWriter, r *http.Request) {
	req := map[string]interface{}{
		"tenant_id":  r.URL.Query().Get("tenant_id"),
		"account_id": r.URL.Query().Get("account_id"),
	}

	if req["tenant_id"] == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req["tenant_id"] = claims.TenantID.String()
		}
	}

	var resp listPaymentsResp
	err := p.conn.Invoke(r.Context(), "/bib.payment.v1.PaymentService/ListPayments", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
