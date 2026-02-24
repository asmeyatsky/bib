package grpc

// proto.go defines the gRPC server interface derived from bib/lending/v1/lending.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/lending/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LendingServiceServer is the server API for LendingService.
// It mirrors the proto-generated interface from bib.lending.v1.LendingService.
type LendingServiceServer interface {
	SubmitApplication(context.Context, *SubmitApplicationRequest) (*SubmitApplicationResponse, error)
	GetApplication(context.Context, *GetApplicationRequest) (*GetApplicationResponse, error)
	DisburseLoan(context.Context, *DisburseLoanRequest) (*DisburseLoanResponse, error)
	GetLoan(context.Context, *GetLoanRequest) (*GetLoanResponse, error)
	MakePayment(context.Context, *MakePaymentRequest) (*MakePaymentResponse, error)
	mustEmbedUnimplementedLendingServiceServer()
}

// UnimplementedLendingServiceServer provides forward-compatible default implementations.
type UnimplementedLendingServiceServer struct{}

func (UnimplementedLendingServiceServer) SubmitApplication(context.Context, *SubmitApplicationRequest) (*SubmitApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitApplication not implemented")
}
func (UnimplementedLendingServiceServer) GetApplication(context.Context, *GetApplicationRequest) (*GetApplicationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetApplication not implemented")
}
func (UnimplementedLendingServiceServer) DisburseLoan(context.Context, *DisburseLoanRequest) (*DisburseLoanResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DisburseLoan not implemented")
}
func (UnimplementedLendingServiceServer) GetLoan(context.Context, *GetLoanRequest) (*GetLoanResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLoan not implemented")
}
func (UnimplementedLendingServiceServer) MakePayment(context.Context, *MakePaymentRequest) (*MakePaymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MakePayment not implemented")
}
func (UnimplementedLendingServiceServer) mustEmbedUnimplementedLendingServiceServer() {}

// RegisterLendingServiceServer registers the LendingServiceServer with the gRPC server.
func RegisterLendingServiceServer(s *grpclib.Server, srv LendingServiceServer) {
	s.RegisterService(&_LendingService_serviceDesc, srv) //nolint:revive // gRPC handler registration
}

//nolint:revive // gRPC handler registration
var _LendingService_serviceDesc = grpclib.ServiceDesc{
	ServiceName: "bib.lending.v1.LendingService",
	HandlerType: (*LendingServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "SubmitApplication", Handler: _LendingService_SubmitApplication_Handler}, //nolint:revive // gRPC handler registration
		{MethodName: "GetApplication", Handler: _LendingService_GetApplication_Handler},       //nolint:revive // gRPC handler registration
		{MethodName: "DisburseLoan", Handler: _LendingService_DisburseLoan_Handler},           //nolint:revive // gRPC handler registration
		{MethodName: "GetLoan", Handler: _LendingService_GetLoan_Handler},                     //nolint:revive // gRPC handler registration
		{MethodName: "MakePayment", Handler: _LendingService_MakePayment_Handler},             //nolint:revive // gRPC handler registration
	},
	Streams: []grpclib.StreamDesc{},
}

//nolint:revive,errcheck // gRPC handler registration
func _LendingService_SubmitApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(SubmitApplicationRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(LendingServiceServer).SubmitApplication(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _LendingService_GetApplication_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GetApplicationRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(LendingServiceServer).GetApplication(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _LendingService_DisburseLoan_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(DisburseLoanRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(LendingServiceServer).DisburseLoan(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _LendingService_GetLoan_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GetLoanRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(LendingServiceServer).GetLoan(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _LendingService_MakePayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(MakePaymentRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(LendingServiceServer).MakePayment(ctx, req)
}
