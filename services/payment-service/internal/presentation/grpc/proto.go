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

func _PaymentService_InitiatePayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(InitiatePaymentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).InitiatePayment(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.payment.v1.PaymentService/InitiatePayment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).InitiatePayment(ctx, req.(*InitiatePaymentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_GetPayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(GetPaymentRequestMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).GetPayment(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.payment.v1.PaymentService/GetPayment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).GetPayment(ctx, req.(*GetPaymentRequestMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_ListPayments_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(ListPaymentsRequestMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).ListPayments(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.payment.v1.PaymentService/ListPayments",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).ListPayments(ctx, req.(*ListPaymentsRequestMsg))
	}
	return interceptor(ctx, in, info, handler)
}
