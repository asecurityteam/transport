package transport

import (
	"net/http"
)

// RequestHeader is a decorator that injects a header value into every request.
type RequestHeader struct {
	wrapped  http.RoundTripper
	provider RequestHeaderProvider
}

// RequestHeaderProvider is mapping function that generates the required header name
// and value from an outgoing request.
type RequestHeaderProvider func(*http.Request) (headerName string, headerValue string)

// RoundTrip annotates the outgoing request and calls the wrapped Client.
func (c *RequestHeader) RoundTrip(r *http.Request) (*http.Response, error) {
	var name, value = c.provider(r)
	r.Header.Set(name, value)
	return c.wrapped.RoundTrip(r)
}

// NewRequestHeader wraps a transport in order to include custom headers.
func NewRequestHeader(provider RequestHeaderProvider) func(http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		return &RequestHeader{
			wrapped:  c,
			provider: provider,
		}
	}
}

// ResponseHeader is a decorator that injects a header value into every request.
type ResponseHeader struct {
	wrapped  http.RoundTripper
	provider ResponseHeaderProvider
}

// ResponseHeaderProvider is mapping function that generates the required header name
// and value to an outgoing response.
type ResponseHeaderProvider func(*http.Response) (headerName string, headerValue string)

// RoundTrip calls the wrapped Client and annotates the outgoing response
func (c *ResponseHeader) RoundTrip(r *http.Request) (*http.Response, error) {

	resp, err := c.wrapped.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	var name, value = c.provider(resp)
	resp.Header.Set(name, value)
	return resp, nil
}

// NewResponseHeader wraps a transport in order to include custom headers.
func NewResponseHeader(provider ResponseHeaderProvider) func(http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		return &ResponseHeader{
			wrapped:  c,
			provider: provider,
		}
	}
}
