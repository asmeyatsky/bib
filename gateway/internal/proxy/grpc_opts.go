package proxy

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

// init registers the JSON codec with the gRPC encoding registry.
func init() {
	encoding.RegisterCodec(jsonCodec{})
}

// grpcCallOption returns the gRPC call option that forces JSON encoding
// on the wire. This lets the gateway invoke backend methods without
// proto-generated stubs. When proto-generated client code is available,
// this helper can be removed and typed client calls used instead.
func grpcCallOption() grpc.CallOption {
	return grpc.ForceCodecCallOption{Codec: jsonCodec{}}
}
