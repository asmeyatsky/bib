package grpc

// proto.go defines the gRPC server interface derived from bib/fx/v1/fx.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/fx/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FXServiceServer is the server API for FXService.
type FXServiceServer interface {
	GetExchangeRate(context.Context, *GetExchangeRateRequest) (*GetExchangeRateResponse, error)
	ConvertAmount(context.Context, *ConvertAmountRequest) (*ConvertAmountResponse, error)
	ListExchangeRates(context.Context, *ListExchangeRatesRequest) (*ListExchangeRatesResponse, error)
	Revaluate(context.Context, *RevaluateRequest) (*RevaluateResponse, error)
	mustEmbedUnimplementedFXServiceServer()
}

// UnimplementedFXServiceServer provides forward-compatible default implementations.
type UnimplementedFXServiceServer struct{}

func (UnimplementedFXServiceServer) GetExchangeRate(context.Context, *GetExchangeRateRequest) (*GetExchangeRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetExchangeRate not implemented")
}
func (UnimplementedFXServiceServer) ConvertAmount(context.Context, *ConvertAmountRequest) (*ConvertAmountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConvertAmount not implemented")
}
func (UnimplementedFXServiceServer) ListExchangeRates(context.Context, *ListExchangeRatesRequest) (*ListExchangeRatesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListExchangeRates not implemented")
}
func (UnimplementedFXServiceServer) Revaluate(context.Context, *RevaluateRequest) (*RevaluateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Revaluate not implemented")
}
func (UnimplementedFXServiceServer) mustEmbedUnimplementedFXServiceServer() {}

// RegisterFXServiceServer registers the FXServiceServer with the gRPC server.
func RegisterFXServiceServer(s *grpclib.Server, srv FXServiceServer) {
	s.RegisterService(&_FXService_serviceDesc, srv)
}

var _FXService_serviceDesc = grpclib.ServiceDesc{
	ServiceName: "bib.fx.v1.FXService",
	HandlerType: (*FXServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "GetExchangeRate", Handler: _FXService_GetExchangeRate_Handler},
		{MethodName: "ConvertAmount", Handler: _FXService_ConvertAmount_Handler},
		{MethodName: "ListExchangeRates", Handler: _FXService_ListExchangeRates_Handler},
		{MethodName: "Revaluate", Handler: _FXService_Revaluate_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _FXService_GetExchangeRate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GetExchangeRateRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(FXServiceServer).GetExchangeRate(ctx, req)
}

func _FXService_ConvertAmount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(ConvertAmountRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(FXServiceServer).ConvertAmount(ctx, req)
}

func _FXService_ListExchangeRates_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(ListExchangeRatesRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(FXServiceServer).ListExchangeRates(ctx, req)
}

func _FXService_Revaluate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(RevaluateRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(FXServiceServer).Revaluate(ctx, req)
}
