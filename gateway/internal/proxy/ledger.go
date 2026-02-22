package proxy

import (
	"log/slog"
	"net/http"

	"github.com/bibbank/bib/pkg/auth"
)

// LedgerProxy proxies HTTP requests to the ledger gRPC service.
type LedgerProxy struct {
	conn   *ServiceConn
	logger *slog.Logger
}

// NewLedgerProxy creates a new ledger service proxy.
func NewLedgerProxy(conn *ServiceConn, logger *slog.Logger) *LedgerProxy {
	return &LedgerProxy{conn: conn, logger: logger}
}

type postingPair struct {
	DebitAccount  string `json:"debit_account"`
	CreditAccount string `json:"credit_account"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Description   string `json:"description,omitempty"`
}

type postJournalEntryReq struct {
	TenantID      string        `json:"tenant_id"`
	EffectiveDate string        `json:"effective_date"`
	Postings      []postingPair `json:"postings"`
	Description   string        `json:"description,omitempty"`
	Reference     string        `json:"reference,omitempty"`
}

type journalEntryMsg struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	EffectiveDate string        `json:"effective_date"`
	Postings      []postingPair `json:"postings"`
	Status        string        `json:"status"`
	Description   string        `json:"description"`
	Reference     string        `json:"reference"`
	Version       int32         `json:"version"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
}

type postJournalEntryResp struct {
	Entry journalEntryMsg `json:"entry"`
}

type getBalanceResp struct {
	AccountCode string `json:"account_code"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	AsOf        string `json:"as_of"`
}

// PostEntry handles POST /api/v1/ledger/entries.
func (p *LedgerProxy) PostEntry(w http.ResponseWriter, r *http.Request) {
	var req postJournalEntryReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.TenantID == "" {
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			req.TenantID = claims.TenantID.String()
		}
	}

	var resp postJournalEntryResp
	err := p.conn.Invoke(r.Context(), "/bib.ledger.v1.LedgerService/PostJournalEntry", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetEntry handles GET /api/v1/ledger/entries/{id}.
func (p *LedgerProxy) GetEntry(w http.ResponseWriter, r *http.Request) {
	entryID := r.PathValue("id")
	if entryID == "" {
		writeError(w, http.StatusBadRequest, "entry id is required")
		return
	}

	req := map[string]string{"id": entryID}
	var resp postJournalEntryResp
	err := p.conn.Invoke(r.Context(), "/bib.ledger.v1.LedgerService/GetJournalEntry", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetBalance handles GET /api/v1/ledger/balances/{account_code}.
func (p *LedgerProxy) GetBalance(w http.ResponseWriter, r *http.Request) {
	accountCode := r.PathValue("account_code")
	if accountCode == "" {
		writeError(w, http.StatusBadRequest, "account_code is required")
		return
	}

	req := map[string]string{
		"account_code": accountCode,
		"as_of":        r.URL.Query().Get("as_of"),
		"currency":     r.URL.Query().Get("currency"),
	}

	var resp getBalanceResp
	err := p.conn.Invoke(r.Context(), "/bib.ledger.v1.LedgerService/GetBalance", &req, &resp)
	if err != nil {
		handleGRPCError(w, err, p.logger)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
