package grpc

// proto.go defines the gRPC server interface derived from bib/identity/v1/identity.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/identity/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IdentityServiceServer is the server API for IdentityService.
// It mirrors the proto-generated interface from bib.identity.v1.IdentityService.
type IdentityServiceServer interface {
	InitiateVerification(context.Context, *InitiateVerificationRequest) (*InitiateVerificationResponse, error)
	GetVerification(context.Context, *GetVerificationRequest) (*GetVerificationResponse, error)
	CompleteCheck(context.Context, *CompleteCheckRequest) (*CompleteCheckResponse, error)
	mustEmbedUnimplementedIdentityServiceServer()
}

// UnimplementedIdentityServiceServer provides forward-compatible default implementations.
type UnimplementedIdentityServiceServer struct{}

func (UnimplementedIdentityServiceServer) InitiateVerification(context.Context, *InitiateVerificationRequest) (*InitiateVerificationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitiateVerification not implemented")
}
func (UnimplementedIdentityServiceServer) GetVerification(context.Context, *GetVerificationRequest) (*GetVerificationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVerification not implemented")
}
func (UnimplementedIdentityServiceServer) CompleteCheck(context.Context, *CompleteCheckRequest) (*CompleteCheckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CompleteCheck not implemented")
}
func (UnimplementedIdentityServiceServer) mustEmbedUnimplementedIdentityServiceServer() {}

// RegisterIdentityServiceServer registers the IdentityServiceServer with the gRPC server.
func RegisterIdentityServiceServer(s *grpclib.Server, srv IdentityServiceServer) {
	s.RegisterService(&_IdentityService_serviceDesc, srv)
}

var _IdentityService_serviceDesc = grpclib.ServiceDesc{ //nolint:revive
	ServiceName: "bib.identity.v1.IdentityService",
	HandlerType: (*IdentityServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "InitiateVerification", Handler: _IdentityService_InitiateVerification_Handler},
		{MethodName: "GetVerification", Handler: _IdentityService_GetVerification_Handler},
		{MethodName: "CompleteCheck", Handler: _IdentityService_CompleteCheck_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _IdentityService_InitiateVerification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(InitiateVerificationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityServiceServer).InitiateVerification(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.identity.v1.IdentityService/InitiateVerification",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityServiceServer).InitiateVerification(ctx, req.(*InitiateVerificationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityService_GetVerification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(GetVerificationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityServiceServer).GetVerification(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.identity.v1.IdentityService/GetVerification",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityServiceServer).GetVerification(ctx, req.(*GetVerificationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _IdentityService_CompleteCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(CompleteCheckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IdentityServiceServer).CompleteCheck(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.identity.v1.IdentityService/CompleteCheck",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IdentityServiceServer).CompleteCheck(ctx, req.(*CompleteCheckRequest))
	}
	return interceptor(ctx, in, info, handler)
}
