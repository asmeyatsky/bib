package grpc

// proto.go defines the gRPC server interface derived from bib/account/v1/account.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/account/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AccountServiceServer is the server API for AccountService.
// It mirrors the proto-generated interface from bib.account.v1.AccountService.
type AccountServiceServer interface {
	OpenAccount(context.Context, *OpenAccountRequest) (*OpenAccountResponse, error)
	GetAccount(context.Context, *GetAccountRequest) (*GetAccountResponse, error)
	FreezeAccount(context.Context, *FreezeAccountRequest) (*FreezeAccountResponse, error)
	CloseAccount(context.Context, *CloseAccountRequest) (*CloseAccountResponse, error)
	ListAccounts(context.Context, *ListAccountsRequest) (*ListAccountsResponse, error)
	mustEmbedUnimplementedAccountServiceServer()
}

// UnimplementedAccountServiceServer provides forward-compatible default implementations.
type UnimplementedAccountServiceServer struct{}

func (UnimplementedAccountServiceServer) OpenAccount(context.Context, *OpenAccountRequest) (*OpenAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OpenAccount not implemented")
}
func (UnimplementedAccountServiceServer) GetAccount(context.Context, *GetAccountRequest) (*GetAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAccount not implemented")
}
func (UnimplementedAccountServiceServer) FreezeAccount(context.Context, *FreezeAccountRequest) (*FreezeAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FreezeAccount not implemented")
}
func (UnimplementedAccountServiceServer) CloseAccount(context.Context, *CloseAccountRequest) (*CloseAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CloseAccount not implemented")
}
func (UnimplementedAccountServiceServer) ListAccounts(context.Context, *ListAccountsRequest) (*ListAccountsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListAccounts not implemented")
}
func (UnimplementedAccountServiceServer) mustEmbedUnimplementedAccountServiceServer() {}

// RegisterAccountServiceServer registers the AccountServiceServer with the gRPC server.
func RegisterAccountServiceServer(s *grpclib.Server, srv AccountServiceServer) {
	s.RegisterService(&_AccountService_serviceDesc, srv) //nolint:revive // gRPC handler registration
}

//nolint:revive // gRPC handler registration
var _AccountService_serviceDesc = grpclib.ServiceDesc{
	ServiceName: "bib.account.v1.AccountService",
	HandlerType: (*AccountServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "OpenAccount", Handler: _AccountService_OpenAccount_Handler},     //nolint:revive // gRPC handler registration
		{MethodName: "GetAccount", Handler: _AccountService_GetAccount_Handler},       //nolint:revive // gRPC handler registration
		{MethodName: "FreezeAccount", Handler: _AccountService_FreezeAccount_Handler}, //nolint:revive // gRPC handler registration
		{MethodName: "CloseAccount", Handler: _AccountService_CloseAccount_Handler},   //nolint:revive // gRPC handler registration
		{MethodName: "ListAccounts", Handler: _AccountService_ListAccounts_Handler},   //nolint:revive // gRPC handler registration
	},
	Streams: []grpclib.StreamDesc{},
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_OpenAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(OpenAccountRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(AccountServiceServer).OpenAccount(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_GetAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GetAccountRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(AccountServiceServer).GetAccount(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_FreezeAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(FreezeAccountRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(AccountServiceServer).FreezeAccount(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_CloseAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(CloseAccountRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(AccountServiceServer).CloseAccount(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_ListAccounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(ListAccountsRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(AccountServiceServer).ListAccounts(ctx, req)
}
