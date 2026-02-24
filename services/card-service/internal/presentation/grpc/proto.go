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
	FreezeCard(context.Context, *FreezeCardGRPCRequest) (*FreezeCardGRPCResponse, error)
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
func (UnimplementedCardServiceServer) FreezeCard(context.Context, *FreezeCardGRPCRequest) (*FreezeCardGRPCResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FreezeCard not implemented")
}
func (UnimplementedCardServiceServer) mustEmbedUnimplementedCardServiceServer() {}

// FreezeCardGRPCRequest represents the proto FreezeCardRequest message.
type FreezeCardGRPCRequest struct {
	CardID string `json:"card_id"`
}

// FreezeCardGRPCResponse represents the proto FreezeCardResponse message.
type FreezeCardGRPCResponse struct {
	CardID string `json:"card_id"`
	Status string `json:"status"`
}

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
		{MethodName: "FreezeCard", Handler: _CardService_FreezeCard_Handler},
	},
	Streams: []grpclib.StreamDesc{},
}

func _CardService_IssueCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(IssueCardRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServiceServer).IssueCard(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.card.v1.CardService/IssueCard",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServiceServer).IssueCard(ctx, req.(*IssueCardRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardService_AuthorizeTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(AuthorizeTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServiceServer).AuthorizeTransaction(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.card.v1.CardService/AuthorizeTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServiceServer).AuthorizeTransaction(ctx, req.(*AuthorizeTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardService_GetCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(GetCardRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServiceServer).GetCard(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.card.v1.CardService/GetCard",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServiceServer).GetCard(ctx, req.(*GetCardRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardService_FreezeCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpclib.UnaryServerInterceptor) (interface{}, error) { //nolint:revive,errcheck // gRPC handler registration
	in := new(FreezeCardGRPCRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServiceServer).FreezeCard(ctx, in)
	}
	info := &grpclib.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/bib.card.v1.CardService/FreezeCard",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServiceServer).FreezeCard(ctx, req.(*FreezeCardGRPCRequest))
	}
	return interceptor(ctx, in, info, handler)
}
