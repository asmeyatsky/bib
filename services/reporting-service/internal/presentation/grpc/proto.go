package grpc

// proto.go defines the gRPC server interface derived from bib/reporting/v1/reporting.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/reporting/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReportingServiceServer is the server API for ReportingService.
// It mirrors the proto-generated interface from bib.reporting.v1.ReportingService.
type ReportingServiceServer interface {
	GenerateReport(context.Context, *GenerateReportRequest) (*GenerateReportResponse, error)
	GetReport(context.Context, *GetReportRequest) (*GetReportResponse, error)
	SubmitReport(context.Context, *SubmitReportRequest) (*SubmitReportResponse, error)
	mustEmbedUnimplementedReportingServiceServer()
}

// UnimplementedReportingServiceServer provides forward-compatible default implementations.
type UnimplementedReportingServiceServer struct{}

func (UnimplementedReportingServiceServer) GenerateReport(context.Context, *GenerateReportRequest) (*GenerateReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateReport not implemented")
}
func (UnimplementedReportingServiceServer) GetReport(context.Context, *GetReportRequest) (*GetReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReport not implemented")
}
func (UnimplementedReportingServiceServer) SubmitReport(context.Context, *SubmitReportRequest) (*SubmitReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitReport not implemented")
}
func (UnimplementedReportingServiceServer) mustEmbedUnimplementedReportingServiceServer() {}

// RegisterReportingServiceServer registers the ReportingServiceServer with the gRPC server.
func RegisterReportingServiceServer(s *grpclib.Server, srv ReportingServiceServer) {
	s.RegisterService(&_ReportingService_serviceDesc, srv) //nolint:revive // gRPC handler registration
}

//nolint:revive // gRPC handler registration
var _ReportingService_serviceDesc = grpclib.ServiceDesc{
	ServiceName: "bib.reporting.v1.ReportingService",
	HandlerType: (*ReportingServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "GenerateReport", Handler: _ReportingService_GenerateReport_Handler}, //nolint:revive // gRPC handler registration
		{MethodName: "GetReport", Handler: _ReportingService_GetReport_Handler},           //nolint:revive // gRPC handler registration
		{MethodName: "SubmitReport", Handler: _ReportingService_SubmitReport_Handler},     //nolint:revive // gRPC handler registration
	},
	Streams: []grpclib.StreamDesc{},
}

//nolint:revive,errcheck // gRPC handler registration
func _ReportingService_GenerateReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GenerateReportRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(ReportingServiceServer).GenerateReport(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _ReportingService_GetReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(GetReportRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(ReportingServiceServer).GetReport(ctx, req)
}

//nolint:revive,errcheck // gRPC handler registration
func _ReportingService_SubmitReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) {
	req := new(SubmitReportRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(ReportingServiceServer).SubmitReport(ctx, req)
}
