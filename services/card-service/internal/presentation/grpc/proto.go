package grpc

// proto.go defines the gRPC server interface derived from bib/card/v1/card.proto.
// This file serves as a stand-in for buf-generated code. Once `buf generate` is run,
// replace this file with the import from github.com/bibbank/bib/api/gen/go/bib/card/v1.

import (
	"context"

	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CardServiceServer is the server API for CardService.
// It mirrors the proto-generated interface from bib.card.v1.CardService.
type CardServiceServer interface {
	IssueCard(context.Context, *IssueCardRequest) (*IssueCardResponse, error)
	AuthorizeTransaction(context.Context, *AuthorizeTransactionRequest) (*AuthorizeTransactionResponse, error)
	GetCard(context.Context, *GetCardRequest) (*GetCardResponse, error)
	mustEmbedUnimplementedCardServiceServer()
}

// UnimplementedCardServiceServer provides forward-compatible default implementations.
type UnimplementedCardServiceServer struct{}

func (UnimplementedCardServiceServer) IssueCard(context.Context, *IssueCardRequest) (*IssueCardResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IssueCard not implemented")
}
func (UnimplementedCardServiceServer) AuthorizeTransaction(context.Context, *AuthorizeTransactionRequest) (*AuthorizeTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AuthorizeTransaction not implemented")
}
func (UnimplementedCardServiceServer) GetCard(context.Context, *GetCardRequest) (*GetCardResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCard not implemented")
}
func (UnimplementedCardServiceServer) mustEmbedUnimplementedCardServiceServer() {}

// RegisterCardServiceServer registers the CardServiceServer with the gRPC server.
func RegisterCardServiceServer(s *grpclib.Server, srv CardServiceServer) {
	s.RegisterService(&_CardService_serviceDesc, srv)
}

var _CardService_serviceDesc = grpclib.ServiceDesc{ //nolint:revive
	ServiceName: "bib.card.v1.CardService",
	HandlerType: (*CardServiceServer)(nil),
	Methods: []grpclib.MethodDesc{
		{MethodName: "IssueCard", Handler: _CardService_IssueCard_Handler},
		{MethodName: "AuthorizeTransaction", Handler: _CardService_AuthorizeTransaction_Handler},
		{MethodName: "GetCard", Handler: _CardService_GetCard_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _CardService_IssueCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(IssueCardRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(CardServiceServer).IssueCard(ctx, req)
}

func _CardService_AuthorizeTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(AuthorizeTransactionRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(CardServiceServer).AuthorizeTransaction(ctx, req)
}

func _CardService_GetCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	req := new(GetCardRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	return srv.(CardServiceServer).GetCard(ctx, req)
}
