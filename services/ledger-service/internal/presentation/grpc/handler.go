package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bibbank/bib/services/ledger-service/internal/application/dto"
	"github.com/bibbank/bib/services/ledger-service/internal/application/usecase"
)

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
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	effectiveDate, err := time.Parse("2006-01-02", req.EffectiveDate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid effective_date: %v", err)
	}

	var postings []dto.PostingPairDTO
	for _, p := range req.Postings {
		amount, err := decimal.NewFromString(p.Amount)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to post journal entry: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to get balance: %v", err)
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
