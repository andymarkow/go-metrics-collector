package httpclient

import (
	"github.com/go-resty/resty/v2"
)

// HTTPClient is a wrapper for resty.Client.
type HTTPClient struct {
	*resty.Client
}

// NewHTTPClient returns a new HTTPClient.
//
// The underlying resty client is created with default settings.
func NewHTTPClient() *HTTPClient {
	client := resty.New()

	return &HTTPClient{
		Client: client,
	}
}
