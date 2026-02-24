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

func _IdentityService_InitiateVerification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(InitiateVerificationRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(IdentityServiceServer).InitiateVerification(ctx, req) //nolint:errcheck
}

func _IdentityService_GetVerification_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(GetVerificationRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(IdentityServiceServer).GetVerification(ctx, req) //nolint:errcheck
}

func _IdentityService_CompleteCheck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(CompleteCheckRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(IdentityServiceServer).CompleteCheck(ctx, req) //nolint:errcheck
}
