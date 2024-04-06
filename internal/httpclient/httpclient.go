package httpclient

import (
	"github.com/go-resty/resty/v2"
)

type HTTPClient struct {
	*resty.Client
}

func NewHTTPClient() *HTTPClient {
	client := resty.New()

	return &HTTPClient{
		Client: client,
	}
}
