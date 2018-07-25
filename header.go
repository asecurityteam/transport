package transport

import (
	"net/http"
)

// Header is a decorator that injects a header value into every request.
type Header struct {
	wrapped  http.RoundTripper
	provider HeaderProvider
}

// HeaderProvider is mapping function that generates the required header name
// and value from an outgoing request.
type HeaderProvider func(*http.Request) (headerName string, headerValue string)

// RoundTrip annotates the outgoing request and calls the wrapped Client.
func (c *Header) RoundTrip(r *http.Request) (*http.Response, error) {
	var name, value = c.provider(r)
	r.Header.Set(name, value)
	return c.wrapped.RoundTrip(r)
}

// NewHeader wraps a transport in order to include custom headers.
func NewHeader(provider HeaderProvider) func(http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		return &Header{
			wrapped:  c,
			provider: provider,
		}
	}
}
