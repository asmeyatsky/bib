package grpc

// proto.go defines the gRPC server interface derived from bib/ledger/v1/ledger.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/ledger/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LedgerServiceServer is the server API for LedgerService.
// It mirrors the proto-generated interface from bib.ledger.v1.LedgerService.
type LedgerServiceServer interface {
	PostJournalEntry(context.Context, *PostJournalEntryRequest) (*PostJournalEntryResponse, error)
	GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error)
	GetJournalEntry(context.Context, *GetJournalEntryRequest) (*GetJournalEntryResponse, error)
	mustEmbedUnimplementedLedgerServiceServer()
}

// UnimplementedLedgerServiceServer provides forward-compatible default implementations.
type UnimplementedLedgerServiceServer struct{}

func (UnimplementedLedgerServiceServer) PostJournalEntry(context.Context, *PostJournalEntryRequest) (*PostJournalEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostJournalEntry not implemented")
}
func (UnimplementedLedgerServiceServer) GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBalance not implemented")
}
func (UnimplementedLedgerServiceServer) GetJournalEntry(context.Context, *GetJournalEntryRequest) (*GetJournalEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetJournalEntry not implemented")
}
func (UnimplementedLedgerServiceServer) mustEmbedUnimplementedLedgerServiceServer() {}

// RegisterLedgerServiceServer registers the LedgerServiceServer with the gRPC server.
func RegisterLedgerServiceServer(s *grpclib.Server, srv LedgerServiceServer) {
	s.RegisterService(&_LedgerService_serviceDesc, srv) //nolint:revive // gRPC handler registration
}

//nolint:revive // gRPC handler registration
var _LedgerService_serviceDesc = grpclib.ServiceDesc{
	ServiceName: "bib.ledger.v1.LedgerService",
	HandlerType: (*LedgerServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "PostJournalEntry", Handler: _LedgerService_PostJournalEntry_Handler}, //nolint:revive // gRPC handler registration
		{MethodName: "GetBalance", Handler: _LedgerService_GetBalance_Handler},             //nolint:revive // gRPC handler registration
		{MethodName: "GetJournalEntry", Handler: _LedgerService_GetJournalEntry_Handler},   //nolint:revive // gRPC handler registration
	},
	Streams: []grpclib.StreamDesc{},
}

//nolint:revive,errcheck // gRPC handler registration
func _LedgerService_PostJournalEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(PostJournalEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).PostJournalEntry(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.ledger.v1.LedgerService/PostJournalEntry",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).PostJournalEntry(ctx, req.(*PostJournalEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

//nolint:revive,errcheck // gRPC handler registration
func _LedgerService_GetBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).GetBalance(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.ledger.v1.LedgerService/GetBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).GetBalance(ctx, req.(*GetBalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

//nolint:revive,errcheck // gRPC handler registration
func _LedgerService_GetJournalEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetJournalEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LedgerServiceServer).GetJournalEntry(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.ledger.v1.LedgerService/GetJournalEntry",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LedgerServiceServer).GetJournalEntry(ctx, req.(*GetJournalEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}
