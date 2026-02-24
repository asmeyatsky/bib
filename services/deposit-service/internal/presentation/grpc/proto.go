package grpc

// proto.go defines the gRPC server interface derived from bib/deposit/v1/deposit.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/deposit/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DepositServiceServer is the server API for DepositService.
type DepositServiceServer interface {
	CreateDepositProduct(context.Context, *CreateDepositProductRequest) (*CreateDepositProductResponse, error)
	OpenDepositPosition(context.Context, *OpenDepositPositionRequest) (*OpenDepositPositionResponse, error)
	GetDepositPosition(context.Context, *GetDepositPositionRequest) (*GetDepositPositionResponse, error)
	AccrueInterest(context.Context, *AccrueInterestRequest) (*AccrueInterestResponse, error)
	mustEmbedUnimplementedDepositServiceServer()
}

// UnimplementedDepositServiceServer provides forward-compatible default implementations.
type UnimplementedDepositServiceServer struct{}

func (UnimplementedDepositServiceServer) CreateDepositProduct(context.Context, *CreateDepositProductRequest) (*CreateDepositProductResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateDepositProduct not implemented")
}
func (UnimplementedDepositServiceServer) OpenDepositPosition(context.Context, *OpenDepositPositionRequest) (*OpenDepositPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OpenDepositPosition not implemented")
}
func (UnimplementedDepositServiceServer) GetDepositPosition(context.Context, *GetDepositPositionRequest) (*GetDepositPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDepositPosition not implemented")
}
func (UnimplementedDepositServiceServer) AccrueInterest(context.Context, *AccrueInterestRequest) (*AccrueInterestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AccrueInterest not implemented")
}
func (UnimplementedDepositServiceServer) mustEmbedUnimplementedDepositServiceServer() {}

// RegisterDepositServiceServer registers the DepositServiceServer with the gRPC server.
func RegisterDepositServiceServer(s *grpclib.Server, srv DepositServiceServer) {
	s.RegisterService(&_DepositService_serviceDesc, srv)
}

var _DepositService_serviceDesc = grpclib.ServiceDesc{ //nolint:revive // gRPC handler registration
	ServiceName: "bib.deposit.v1.DepositService",
	HandlerType: (*DepositServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "CreateProduct", Handler: _DepositService_CreateDepositProduct_Handler},
		{MethodName: "OpenPosition", Handler: _DepositService_OpenDepositPosition_Handler},
		{MethodName: "GetPosition", Handler: _DepositService_GetDepositPosition_Handler},
		{MethodName: "AccrueInterest", Handler: _DepositService_AccrueInterest_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _DepositService_CreateDepositProduct_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(CreateDepositProductRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DepositServiceServer).CreateDepositProduct(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.deposit.v1.DepositService/CreateProduct",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DepositServiceServer).CreateDepositProduct(ctx, req.(*CreateDepositProductRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DepositService_OpenDepositPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(OpenDepositPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DepositServiceServer).OpenDepositPosition(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.deposit.v1.DepositService/OpenPosition",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DepositServiceServer).OpenDepositPosition(ctx, req.(*OpenDepositPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DepositService_GetDepositPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(GetDepositPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DepositServiceServer).GetDepositPosition(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.deposit.v1.DepositService/GetPosition",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DepositServiceServer).GetDepositPosition(ctx, req.(*GetDepositPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DepositService_AccrueInterest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(AccrueInterestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DepositServiceServer).AccrueInterest(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.deposit.v1.DepositService/AccrueInterest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DepositServiceServer).AccrueInterest(ctx, req.(*AccrueInterestRequest))
	}
	return interceptor(ctx, in, info, handler)
}
