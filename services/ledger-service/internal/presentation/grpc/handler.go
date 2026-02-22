package grpc

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	postEntry   *usecase.PostJournalEntry
	getEntry    *usecase.GetJournalEntry
	getBalance  *usecase.GetBalance
	listEntries *usecase.ListJournalEntries
	backvalue   *usecase.BackvalueEntry
	periodClose *usecase.PeriodClose
}

func NewLedgerHandler(
	postEntry *usecase.PostJournalEntry,
	getEntry *usecase.GetJournalEntry,
	getBalance *usecase.GetBalance,
	listEntries *usecase.ListJournalEntries,
	backvalue *usecase.BackvalueEntry,
	periodClose *usecase.PeriodClose,
) *LedgerHandler {
	return &LedgerHandler{
		postEntry:   postEntry,
		getEntry:    getEntry,
		getBalance:  getBalance,
		listEntries: listEntries,
		backvalue:   backvalue,
		periodClose: periodClose,
	}
}

// PostJournalEntry handles gRPC PostJournalEntry calls.
// Since we don't have generated proto code yet, we define a manual interface.
// This will be updated when proto generation is wired.

// PostJournalEntryRequest/Response are temporary types until proto gen is wired.
type PostJournalEntryRequest struct {
	TenantID      string
	EffectiveDate string
	Postings      []*PostingPairMsg
	Description   string
	Reference     string
}

type PostingPairMsg struct {
	DebitAccount  string
	CreditAccount string
	Amount        string
	Currency      string
	Description   string
}

type JournalEntryMsg struct {
	ID            string
	TenantID      string
	EffectiveDate string
	Postings      []*PostingPairMsg
	Status        string
	Description   string
	Reference     string
	Version       int32
	CreatedAt     *timestamppb.Timestamp
	UpdatedAt     *timestamppb.Timestamp
}

type PostJournalEntryResponse struct {
	Entry *JournalEntryMsg
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
		amount, err := decimal.NewFromString(p.Amount)
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
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &PostJournalEntryResponse{
		Entry: toJournalEntryMsg(result),
	}, nil
}

type GetBalanceRequest struct {
	AccountCode string
	AsOf        string
	Currency    string
}

type GetBalanceResponse struct {
	AccountCode string
	Amount      string
	Currency    string
	AsOf        string
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
		// TODO: log original error server-side: err
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetBalanceResponse{
		AccountCode: result.AccountCode,
		Amount:      result.Amount.String(),
		Currency:    result.Currency,
		AsOf:        result.AsOf.Format("2006-01-02"),
	}, nil
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
		Version:       int32(r.Version),
		CreatedAt:     timestamppb.New(r.CreatedAt),
		UpdatedAt:     timestamppb.New(r.UpdatedAt),
	}
}
