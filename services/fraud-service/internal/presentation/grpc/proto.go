package grpc

// proto.go defines the gRPC server interface derived from bib/fraud/v1/fraud.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/fraud/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FraudServiceServer is the server API for FraudService.
type FraudServiceServer interface {
	AssessTransaction(context.Context, *AssessTransactionRequest) (*AssessTransactionResponse, error)
	GetAssessment(context.Context, *GetAssessmentRequest) (*GetAssessmentResponse, error)
	mustEmbedUnimplementedFraudServiceServer()
}

// UnimplementedFraudServiceServer provides forward-compatible default implementations.
type UnimplementedFraudServiceServer struct{}

func (UnimplementedFraudServiceServer) AssessTransaction(context.Context, *AssessTransactionRequest) (*AssessTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AssessTransaction not implemented")
}
func (UnimplementedFraudServiceServer) GetAssessment(context.Context, *GetAssessmentRequest) (*GetAssessmentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAssessment not implemented")
}
func (UnimplementedFraudServiceServer) mustEmbedUnimplementedFraudServiceServer() {}

// RegisterFraudServiceServer registers the FraudServiceServer with the gRPC server.
func RegisterFraudServiceServer(s *grpclib.Server, srv FraudServiceServer) {
	s.RegisterService(&_FraudService_serviceDesc, srv)
}

var _FraudService_serviceDesc = grpclib.ServiceDesc{
	ServiceName: "bib.fraud.v1.FraudService",
	HandlerType: (*FraudServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "AssessTransaction", Handler: _FraudService_AssessTransaction_Handler},
		{MethodName: "GetAssessment", Handler: _FraudService_GetAssessment_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _FraudService_AssessTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(AssessTransactionRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(FraudServiceServer).AssessTransaction(ctx, req)
}

func _FraudService_GetAssessment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GetAssessmentRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(FraudServiceServer).GetAssessment(ctx, req)
}
