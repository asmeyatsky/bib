package grpc

// proto.go defines the gRPC server interface derived from bib/payment/v1/payment.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/payment/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PaymentServiceServer is the server API for PaymentService.
// It mirrors the proto-generated interface from bib.payment.v1.PaymentService.
type PaymentServiceServer interface {
	InitiatePayment(context.Context, *InitiatePaymentRequest) (*InitiatePaymentResponse, error)
	GetPayment(context.Context, *GetPaymentRequestMsg) (*GetPaymentResponseMsg, error)
	ListPayments(context.Context, *ListPaymentsRequestMsg) (*ListPaymentsResponseMsg, error)
	mustEmbedUnimplementedPaymentServiceServer()
}

// UnimplementedPaymentServiceServer provides forward-compatible default implementations.
type UnimplementedPaymentServiceServer struct{}

func (UnimplementedPaymentServiceServer) InitiatePayment(context.Context, *InitiatePaymentRequest) (*InitiatePaymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitiatePayment not implemented")
}
func (UnimplementedPaymentServiceServer) GetPayment(context.Context, *GetPaymentRequestMsg) (*GetPaymentResponseMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPayment not implemented")
}
func (UnimplementedPaymentServiceServer) ListPayments(context.Context, *ListPaymentsRequestMsg) (*ListPaymentsResponseMsg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPayments not implemented")
}
func (UnimplementedPaymentServiceServer) mustEmbedUnimplementedPaymentServiceServer() {}

// RegisterPaymentServiceServer registers the PaymentServiceServer with the gRPC server.
func RegisterPaymentServiceServer(s *grpclib.Server, srv PaymentServiceServer) {
	s.RegisterService(&_PaymentService_serviceDesc, srv)
}

var _PaymentService_serviceDesc = grpclib.ServiceDesc{ //nolint:revive
	ServiceName: "bib.payment.v1.PaymentService",
	HandlerType: (*PaymentServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "InitiatePayment", Handler: _PaymentService_InitiatePayment_Handler},
		{MethodName: "GetPayment", Handler: _PaymentService_GetPayment_Handler},
		{MethodName: "ListPayments", Handler: _PaymentService_ListPayments_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _PaymentService_InitiatePayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(InitiatePaymentRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(PaymentServiceServer).InitiatePayment(ctx, req) //nolint:errcheck
}

func _PaymentService_GetPayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(GetPaymentRequestMsg)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(PaymentServiceServer).GetPayment(ctx, req) //nolint:errcheck
}

func _PaymentService_ListPayments_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(ListPaymentsRequestMsg)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(PaymentServiceServer).ListPayments(ctx, req) //nolint:errcheck
}
