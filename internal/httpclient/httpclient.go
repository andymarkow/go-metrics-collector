// Package httpclient provides a HTTP client implementation.
package httpclient

import (
	"errors"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// HTTPClient represents a HTTP client.
type HTTPClient struct {
	*resty.Client
}

// NewHTTPClient returns a new HTTPClient.
//
// The underlying resty client is created with default settings.
func NewHTTPClient(opts ...Option) *HTTPClient {
	cl := &HTTPClient{resty.New()}

	setDefaultConfig(cl)

	for _, opt := range opts {
		opt(cl)
	}

	return cl
}

// Option is a HTTP client option.
type Option func(c *HTTPClient)

// WithLogger sets the logger for the HTTP client.
func WithLogger(log *zap.Logger) Option {
	return func(c *HTTPClient) {
		c.SetLogger(log.Sugar())
	}
}

func WithBaseURL(baseURL string) Option {
	// Check if the URL does not start with "http://" or "https://".
	if !strings.HasPrefix(baseURL, "http://") &&
		!strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	return func(c *HTTPClient) {
		c.SetBaseURL(baseURL)
	}
}

func setDefaultConfig(c *HTTPClient) {
	c.SetRetryCount(3)
	c.SetRetryWaitTime(1 * time.Second)
	c.SetRetryMaxWaitTime(10 * time.Second)
	c.SetRetryAfter(retryAfterWithInterval(2))
	c.AddRetryCondition(func(_ *resty.Response, err error) bool {
		// Retry for retryable errors.
		return isRetryableError(err)
	})
}

// retryAfterWithInterval returns duration intervals between retries.
func retryAfterWithInterval(retryWaitInterval int) resty.RetryAfterFunc {
	return func(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
		return time.Duration((resp.Request.Attempt*retryWaitInterval - 1)) * time.Second, nil
	}
}

// isRetryableError checks if the error is a retryable error.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		// Connection refused error
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			// Connection timeout error
			return true
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// DNS error
		return true
	}

	var addrErr *net.AddrError
	if errors.As(err, &addrErr) {
		// Address error
		return true
	}

	// Operational error
	var opErr *net.OpError

	return errors.As(err, &opErr)
}
