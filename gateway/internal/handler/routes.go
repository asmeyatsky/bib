package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bibbank/bib/gateway/internal/proxy"
)

// Proxies holds all backend service proxy instances.
type Proxies struct {
	Account   *proxy.AccountProxy
	Ledger    *proxy.LedgerProxy
	Payment   *proxy.PaymentProxy
	FX        *proxy.FXProxy
	Identity  *proxy.IdentityProxy
	Deposit   *proxy.DepositProxy
	Card      *proxy.CardProxy
	Lending   *proxy.LendingProxy
	Fraud     *proxy.FraudProxy
	Reporting *proxy.ReportingProxy
	Partner   *proxy.PartnerProxy
}

// RegisterRoutes registers all REST API routes on the given ServeMux.
func RegisterRoutes(mux *http.ServeMux, p *Proxies) {
	// Health
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/readyz", readyz)

	// --- Ledger ---
	mux.HandleFunc("POST /api/v1/ledger/entries", p.Ledger.PostEntry)
	mux.HandleFunc("GET /api/v1/ledger/entries/{id}", p.Ledger.GetEntry)
	mux.HandleFunc("GET /api/v1/ledger/balances/{account_code}", p.Ledger.GetBalance)

	// --- Accounts ---
	mux.HandleFunc("POST /api/v1/accounts", p.Account.OpenAccount)
	mux.HandleFunc("GET /api/v1/accounts/{id}", p.Account.GetAccount)
	mux.HandleFunc("POST /api/v1/accounts/{id}/freeze", p.Account.FreezeAccount)
	mux.HandleFunc("POST /api/v1/accounts/{id}/close", p.Account.CloseAccount)
	mux.HandleFunc("GET /api/v1/accounts", p.Account.ListAccounts)

	// --- Payments ---
	mux.HandleFunc("POST /api/v1/payments", p.Payment.InitiatePayment)
	mux.HandleFunc("GET /api/v1/payments/{id}", p.Payment.GetPayment)
	mux.HandleFunc("GET /api/v1/payments", p.Payment.ListPayments)

	// --- FX ---
	mux.HandleFunc("GET /api/v1/fx/rates/{pair}", p.FX.GetRate)
	mux.HandleFunc("POST /api/v1/fx/convert", p.FX.Convert)

	// --- Identity ---
	mux.HandleFunc("POST /api/v1/identity/verifications", p.Identity.InitiateVerification)
	mux.HandleFunc("GET /api/v1/identity/verifications/{id}", p.Identity.GetVerification)

	// --- Deposits ---
	mux.HandleFunc("POST /api/v1/deposits/products", p.Deposit.CreateProduct)
	mux.HandleFunc("POST /api/v1/deposits/positions", p.Deposit.OpenPosition)
	mux.HandleFunc("GET /api/v1/deposits/positions/{id}", p.Deposit.GetPosition)

	// --- Cards ---
	mux.HandleFunc("POST /api/v1/cards", p.Card.IssueCard)
	mux.HandleFunc("GET /api/v1/cards/{id}", p.Card.GetCard)
	mux.HandleFunc("POST /api/v1/cards/{id}/freeze", p.Card.FreezeCard)
	mux.HandleFunc("POST /api/v1/cards/{id}/authorize", p.Card.AuthorizeTransaction)

	// --- Lending ---
	mux.HandleFunc("POST /api/v1/loans/applications", p.Lending.SubmitApplication)
	mux.HandleFunc("GET /api/v1/loans/applications/{id}", p.Lending.GetApplication)
	mux.HandleFunc("POST /api/v1/loans/disburse", p.Lending.DisburseLoan)
	mux.HandleFunc("GET /api/v1/loans/{id}", p.Lending.GetLoan)
	mux.HandleFunc("POST /api/v1/loans/{id}/payments", p.Lending.MakePayment)

	// --- Fraud ---
	mux.HandleFunc("POST /api/v1/fraud/assessments", p.Fraud.AssessTransaction)
	mux.HandleFunc("GET /api/v1/fraud/assessments/{id}", p.Fraud.GetAssessment)

	// --- Reporting ---
	mux.HandleFunc("POST /api/v1/reports", p.Reporting.GenerateReport)
	mux.HandleFunc("GET /api/v1/reports/{id}", p.Reporting.GetReport)
	mux.HandleFunc("POST /api/v1/reports/{id}/submit", p.Reporting.SubmitReport)

	// --- Partner / Embedded Finance ---
	if p.Partner != nil {
		mux.HandleFunc("POST /api/v1/partner/accounts", p.Partner.CreateAccount)
		mux.HandleFunc("POST /api/v1/partner/payments", p.Partner.InitiatePayment)
		mux.HandleFunc("GET /api/v1/partner/balances/{account_code}", p.Partner.GetBalance)
		mux.HandleFunc("POST /api/v1/partner/webhooks", p.Partner.RegisterWebhook)
		mux.HandleFunc("GET /api/v1/partner/webhooks", p.Partner.ListWebhooks)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
