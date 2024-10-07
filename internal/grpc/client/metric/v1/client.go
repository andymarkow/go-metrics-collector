// Package client provides a gRPC client implementation.
package client

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pbv1 "github.com/andymarkow/go-metrics-collector/internal/grpc/api/metric/v1"
	"github.com/andymarkow/go-metrics-collector/internal/grpc/interceptor"
)

// Client represents a gRPC client.
type Client struct {
	cfg         *config
	log         *zap.Logger
	conn        *grpc.ClientConn
	client      pbv1.MetricServiceClient
	rateLimiter *rate.Limiter
}

// config represents a gRPC client configuration.
type config struct {
	addr string
}

// NewClient creates a new gRPC client.
func NewClient(opts ...Option) (*Client, error) {
	cfg := defaultConfig()

	client := &Client{
		cfg:         cfg,
		log:         zap.NewNop(),
		rateLimiter: rate.NewLimiter(rate.Limit(10), 10),
	}

	for _, opt := range opts {
		opt(client)
	}

	logInterc := interceptor.NewLogInterceptor(interceptor.WithLogInterceptorLogger(client.log))

	conn, err := grpc.NewClient(cfg.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithChainUnaryInterceptor(logInterc.ClientLogInterceptor),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient: %w", err)
	}

	metricsClient := pbv1.NewMetricServiceClient(conn)

	client.conn = conn
	client.client = metricsClient

	return client, nil
}

func defaultConfig() *config {
	return &config{
		addr: ":50051",
	}
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("conn.Close: %w", err)
	}

	return nil
}

// Option represents a gRPC client option.
type Option func(c *Client)

// WithServerAddr sets the gRPC server address.
func WithServerAddr(addr string) Option {
	return func(c *Client) {
		c.cfg.addr = addr
	}
}

// WithLogger sets the logger for the gRPC client.
func WithLogger(log *zap.Logger) Option {
	return func(c *Client) {
		c.log = log
	}
}

// WithRateLimiter sets the rate limiter for the gRPC client.
func WithRateLimiter(rateLimiter *rate.Limiter) Option {
	return func(c *Client) {
		c.rateLimiter = rateLimiter
	}
}

// UpdateMetricsV1 updates metrics.
func (c *Client) UpdateMetricsV1(ctx context.Context, hashsum string, data []byte) (string, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rateLimiter.Wait: %w", err)
	}

	if hashsum != "" {
		md := metadata.New(map[string]string{"hashsum": hashsum})

		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	resp, err := c.client.UpdateMetrics(ctx, &pbv1.UpdateMetricsRequest{
		Payload: &pbv1.Payload{
			Data: &wrapperspb.BytesValue{Value: data},
		},
	})
	if err != nil {
		if e, ok := status.FromError(err); ok {
			return "", fmt.Errorf("CODE: %s, MESSAGE: %s", e.Code(), e.Message())
		}

		return "", fmt.Errorf("client.UpdateMetrics: %w", err)
	}

	if err := resp.GetError().GetMsg().GetValue(); err != "" {
		return "", fmt.Errorf("logical error received: %s", err)
	}

	return resp.GetStatus().GetMsg().GetValue(), nil
}
