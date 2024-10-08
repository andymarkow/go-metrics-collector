// Package metricclient provides a metric client implementation.
package metricclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/andymarkow/go-metrics-collector/internal/cryptutils"
	"github.com/andymarkow/go-metrics-collector/internal/httpclient"
)

// MetricClient represents a metric client.
type MetricClient struct {
	log         *zap.Logger
	client      *httpclient.HTTPClient
	rateLimiter *rate.Limiter
	cryptoKey   *rsa.PublicKey
}

// NewMetricClient returns a new metric client.
func NewMetricClient(opts ...Option) *MetricClient {
	mc := &MetricClient{
		log:         zap.NewNop(),
		client:      httpclient.NewHTTPClient(),
		rateLimiter: rate.NewLimiter(rate.Limit(10), 10),
	}

	for _, opt := range opts {
		opt(mc)
	}

	return mc
}

// Option represents a metric client option.
type Option func(c *MetricClient)

// WithLogger sets the logger for the metric client.
func WithLogger(log *zap.Logger) Option {
	return func(c *MetricClient) {
		c.log = log
	}
}

// WithRateLimiter sets the rate limiter for the metric client.
func WithRateLimiter(rateLimiter *rate.Limiter) Option {
	return func(c *MetricClient) {
		c.rateLimiter = rateLimiter
	}
}

// WithServerAddr sets the server address for the metric client.
func WithServerAddr(addr string) Option {
	return func(c *MetricClient) {
		c.client.SetBaseURL(addr)
	}
}

// WithCryptoKey sets the crypto key for the metric client.
func WithCryptoKey(key *rsa.PublicKey) Option {
	return func(c *MetricClient) {
		c.cryptoKey = key
	}
}

// UpdateMetrics updates metrics.
func (c *MetricClient) UpdateMetrics(ctx context.Context, hashsum string, data []byte) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rateLimiter.Wait: %w", err)
	}

	// Compress payload data with gzip compression method.
	body, err := compressDataGzip(data)
	if err != nil {
		return fmt.Errorf("failed to compress payload data with gzip: %w", err)
	}

	// If crypto public key is set, encrypt payload data with a public RSA key.
	if c.cryptoKey != nil {
		body, err = encryptData(c.cryptoKey, body)
		if err != nil {
			return fmt.Errorf("failed to encrypt payload data with a public RSA key: %w", err)
		}

		c.log.Debug("payload encrypted")
	}

	ip, err := getIPAddress()
	if err != nil {
		return fmt.Errorf("failed to get IP address: %w", err)
	}

	req := c.client.R().
		SetHeader("X-Real-IP", ip.String()).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(body)

	if hashsum != "" {
		req.SetHeader("HashSHA256", hashsum)
	}

	// Send payload data to the remote server.
	resp, err := req.Post("/updates")
	if err != nil {
		return fmt.Errorf("client.PostRequest: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to send data: %d - %s", resp.StatusCode(), resp.String())
	}

	return nil
}

func encryptData(key *rsa.PublicKey, data []byte) ([]byte, error) {
	cryptoHash := sha256.New()

	// Encrypt data with a public RSA key.
	encryptedData, err := cryptutils.EncryptOAEP(cryptoHash, rand.Reader, key, data, nil)
	if err != nil {
		return nil, fmt.Errorf("cryptutils.EncryptOAEP: %w", err)
	}

	return encryptedData, nil
}

// compressDataGzip compresses the given data using gzip.
//
// The function writes the given data to a gzip writer and then closes the writer.
// If any error occurs while writing or closing, the function returns the error.
//
// If no error occurs, the function returns the compressed data as a byte slice.
func compressDataGzip(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	zbuf := gzip.NewWriter(buf)

	if _, err := zbuf.Write(data); err != nil {
		return nil, fmt.Errorf("zbuf.Write: %w", err)
	}

	if err := zbuf.Close(); err != nil {
		return nil, fmt.Errorf("zbuf.Close: %w", err)
	}

	return buf.Bytes(), nil
}

// getIPAddress returns the first non-loopback IPv4 address of the system.
func getIPAddress() (net.IP, error) {
	// Get a addresses list of all network interfaces.
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("net.InterfaceAddrs: %w", err)
	}

	for _, addr := range addrs {
		// Get the IP address network.
		ipNet, ok := addr.(*net.IPNet)
		// If the IP address is IPv4 and not a loopback address.
		if ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
			// Return first valid non-loopback IPv4 address.
			return ipNet.IP, nil
		}
	}

	// If no valid non-loopback IPv4 address is found, return 127.0.0.1.
	return net.IPv4(127, 0, 0, 1), nil
}
