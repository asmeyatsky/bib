package grpc

import (
	"context"
	"log/slog"
	"math"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bibbank/bib/pkg/auth"
	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
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

var currencyCodeRE = regexp.MustCompile(`^[A-Z]{3}$`)

// tenantIDFromContext extracts the tenant ID from JWT claims in the context.
func tenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "authentication required")
	}
	return claims.TenantID, nil
}

// LedgerHandler implements the gRPC LedgerService server.
type LedgerHandler struct {
	UnimplementedLedgerServiceServer
	postEntry   *usecase.PostJournalEntry
	getEntry    *usecase.GetJournalEntry
	getBalance  *usecase.GetBalance
	listEntries *usecase.ListJournalEntries
	backvalue   *usecase.BackvalueEntry
	periodClose *usecase.PeriodClose

	logger *slog.Logger
}

func NewLedgerHandler(
	postEntry *usecase.PostJournalEntry,
	getEntry *usecase.GetJournalEntry,
	getBalance *usecase.GetBalance,
	listEntries *usecase.ListJournalEntries,
	backvalue *usecase.BackvalueEntry,
	periodClose *usecase.PeriodClose,
	logger *slog.Logger,
) *LedgerHandler {
	return &LedgerHandler{
		postEntry:   postEntry,
		getEntry:    getEntry,
		getBalance:  getBalance,
		listEntries: listEntries,
		backvalue:   backvalue,
		periodClose: periodClose,

		logger: logger}
}

// PostJournalEntry handles gRPC PostJournalEntry calls.
// Since we don't have generated proto code yet, we define a manual interface.
// This will be updated when proto generation is wired.

// PostJournalEntryRequest/Response are temporary types until proto gen is wired.
type PostJournalEntryRequest struct {
	TenantID      string            `json:"tenant_id"`
	EffectiveDate string            `json:"effective_date"`
	Description   string            `json:"description,omitempty"`
	Reference     string            `json:"reference,omitempty"`
	Postings      []*PostingPairMsg `json:"postings"`
}

type PostingPairMsg struct {
	DebitAccount  string `json:"debit_account"`
	CreditAccount string `json:"credit_account"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Description   string `json:"description,omitempty"`
}

type JournalEntryMsg struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	EffectiveDate string            `json:"effective_date"`
	Status        string            `json:"status"`
	Description   string            `json:"description"`
	Reference     string            `json:"reference"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
	Postings      []*PostingPairMsg `json:"postings"`
	Version       int32             `json:"version"`
}

type PostJournalEntryResponse struct {
	Entry *JournalEntryMsg `json:"entry"`
}

func (h *LedgerHandler) HandlePostJournalEntry(ctx context.Context, req *PostJournalEntryRequest) (*PostJournalEntryResponse, error) {
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

	effectiveDate, err := time.Parse("2006-01-02", req.EffectiveDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid effective_date: %v", err)
	}

	if len(req.Postings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one posting is required")
	}

	var postings []dto.PostingPairDTO
	for i, p := range req.Postings {
		if p.DebitAccount == "" {
			return nil, status.Errorf(codes.InvalidArgument, "posting[%d]: debit_account is required", i)
		}
		if p.CreditAccount == "" {
			return nil, status.Errorf(codes.InvalidArgument, "posting[%d]: credit_account is required", i)
		}
		var amount decimal.Decimal
		amount, err = decimal.NewFromString(p.Amount)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "posting[%d]: invalid amount: %v", i, err)
		}
		if !amount.IsPositive() {
			return nil, status.Errorf(codes.InvalidArgument, "posting[%d]: amount must be positive", i)
		}
		if p.Currency == "" {
			return nil, status.Errorf(codes.InvalidArgument, "posting[%d]: currency is required", i)
		}
		if !currencyCodeRE.MatchString(p.Currency) {
			return nil, status.Errorf(codes.InvalidArgument, "posting[%d]: currency must be a 3-letter uppercase ISO code", i)
		}
		postings = append(postings, dto.PostingPairDTO{
			DebitAccount:  p.DebitAccount,
			CreditAccount: p.CreditAccount,
			Amount:        amount,
			Currency:      p.Currency,
			Description:   p.Description,
		})
	}

	result, err := h.postEntry.Execute(ctx, dto.PostJournalEntryRequest{
		TenantID:      tenantID,
		EffectiveDate: effectiveDate,
		Postings:      postings,
		Description:   req.Description,
		Reference:     req.Reference,
	})
	if err != nil {
		h.logger.Error("handler error", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &PostJournalEntryResponse{
		Entry: toJournalEntryMsg(result),
	}, nil
}

type GetBalanceRequest struct {
	AccountCode string `json:"account_code"`
	AsOf        string `json:"as_of"`
	Currency    string `json:"currency"`
}

type GetBalanceResponse struct {
	AccountCode string `json:"account_code"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	AsOf        string `json:"as_of"`
}

func (h *LedgerHandler) HandleGetBalance(ctx context.Context, req *GetBalanceRequest) (*GetBalanceResponse, error) {
	if err := requireRole(ctx, auth.RoleAdmin, auth.RoleOperator, auth.RoleAuditor, auth.RoleCustomer, auth.RoleAPIClient); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if req.AccountCode == "" {
		return nil, status.Error(codes.InvalidArgument, "account_code is required")
	}
	if req.Currency != "" && !currencyCodeRE.MatchString(req.Currency) {
		return nil, status.Error(codes.InvalidArgument, "currency must be a 3-letter uppercase ISO code")
	}

	var asOf time.Time
	if req.AsOf != "" {
		var err error
		asOf, err = time.Parse("2006-01-02", req.AsOf)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid as_of date: %v", err)
		}
	}

	result, err := h.getBalance.Execute(ctx, dto.GetBalanceRequest{
		AccountCode: req.AccountCode,
		Currency:    req.Currency,
		AsOf:        asOf,
	})
	if err != nil {
		h.logger.Error("handler error", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetBalanceResponse{
		AccountCode: result.AccountCode,
		Amount:      result.Amount.String(),
		Currency:    result.Currency,
		AsOf:        result.AsOf.Format("2006-01-02"),
	}, nil
}

// GetJournalEntryRequest represents the proto GetJournalEntryRequest message.
type GetJournalEntryRequest struct {
	ID string `json:"id"`
}

// GetJournalEntryResponse represents the proto GetJournalEntryResponse message.
type GetJournalEntryResponse struct {
	Entry *JournalEntryMsg `json:"entry"`
}

// GetJournalEntry retrieves a journal entry by ID.
func (h *LedgerHandler) GetJournalEntry(_ context.Context, _ *GetJournalEntryRequest) (*GetJournalEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetJournalEntry not implemented")
}

// PostJournalEntry delegates to HandlePostJournalEntry for gRPC interface compatibility.
func (h *LedgerHandler) PostJournalEntry(ctx context.Context, req *PostJournalEntryRequest) (*PostJournalEntryResponse, error) {
	return h.HandlePostJournalEntry(ctx, req)
}

// GetBalance delegates to HandleGetBalance for gRPC interface compatibility.
func (h *LedgerHandler) GetBalance(ctx context.Context, req *GetBalanceRequest) (*GetBalanceResponse, error) {
	return h.HandleGetBalance(ctx, req)
}

func toJournalEntryMsg(r dto.JournalEntryResponse) *JournalEntryMsg {
	var postings []*PostingPairMsg
	for _, p := range r.Postings {
		postings = append(postings, &PostingPairMsg{
			DebitAccount:  p.DebitAccount,
			CreditAccount: p.CreditAccount,
			Amount:        p.Amount.String(),
			Currency:      p.Currency,
			Description:   p.Description,
		})
	}
	return &JournalEntryMsg{
		ID:            r.ID.String(),
		TenantID:      r.TenantID.String(),
		EffectiveDate: r.EffectiveDate.Format("2006-01-02"),
		Postings:      postings,
		Status:        r.Status,
		Description:   r.Description,
		Reference:     r.Reference,
		Version:       int32(min(r.Version, math.MaxInt32)), // #nosec G115
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     r.UpdatedAt.Format(time.RFC3339),
	}
}
