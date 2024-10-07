// Package interceptor provides gRPC interceptors.
package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// LogInterceptor represents a gRPC interceptor.
type LogInterceptor struct {
	log *zap.Logger
}

// NewLogInterceptor creates a new LogInterceptor instance.
func NewLogInterceptor(opts ...LogInterceptorOpt) *LogInterceptor {
	i := &LogInterceptor{
		log: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(i)
	}

	return i
}

// LogInterceptorOpt is an LogInterceptor option.
type LogInterceptorOpt func(i *LogInterceptor)

// WithLogInterceptorLogger is an LogInterceptor option that sets logger.
func WithLogInterceptorLogger(log *zap.Logger) LogInterceptorOpt {
	return func(i *LogInterceptor) {
		i.log = log
	}
}

func (i *LogInterceptor) ServerLogInterceptor(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (interface{}, error) {
	i.log.Info("incoming grpc request", zap.String("method", info.FullMethod))

	return handler(ctx, req)
}

func (i *LogInterceptor) ClientLogInterceptor(
	ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
) error {
	startTime := time.Now()

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		i.log.Error("outgoing grpc request", zap.String("method", method), zap.Error(err))
	} else {
		i.log.Info("outgoing grpc request", zap.String("method", method), zap.Duration("time", time.Since(startTime)))
	}

	return err
}
