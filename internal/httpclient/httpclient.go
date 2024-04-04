package httpclient

import "net/http"

type HTTPClient struct {
	http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{}
}
