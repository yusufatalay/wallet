package interceptors

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientAuthInterceptor, extracts metadata from incoming context and adds it to gRPC.
func ClientAuthInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Println("[Client Interceptor] could not found metadata from incoming context, proceeding...")

			return invoker(ctx, method, req, reply, cc, opts...)
		}

		log.Printf("[Client Interceptor] forwarding metadata: %v", md)

		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
