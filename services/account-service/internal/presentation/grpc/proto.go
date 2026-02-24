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
func _AccountService_OpenAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(OpenAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountServiceServer).OpenAccount(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.account.v1.AccountService/OpenAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountServiceServer).OpenAccount(ctx, req.(*OpenAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_GetAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountServiceServer).GetAccount(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.account.v1.AccountService/GetAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountServiceServer).GetAccount(ctx, req.(*GetAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_FreezeAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(FreezeAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountServiceServer).FreezeAccount(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.account.v1.AccountService/FreezeAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountServiceServer).FreezeAccount(ctx, req.(*FreezeAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_CloseAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(CloseAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountServiceServer).CloseAccount(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.account.v1.AccountService/CloseAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountServiceServer).CloseAccount(ctx, req.(*CloseAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

//nolint:revive,errcheck // gRPC handler registration
func _AccountService_ListAccounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListAccountsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountServiceServer).ListAccounts(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.account.v1.AccountService/ListAccounts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountServiceServer).ListAccounts(ctx, req.(*ListAccountsRequest))
	}
	return interceptor(ctx, in, info, handler)
}
