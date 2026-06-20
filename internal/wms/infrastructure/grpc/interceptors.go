package grpc

import (
	"context"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryLimitInterceptor(rps, burst int) grpc.UnaryServerInterceptor {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if limiter.Allow() {
			return handler(ctx, req)
		}

		return nil, status.Error(codes.ResourceExhausted, "too many requests")
	}
}
