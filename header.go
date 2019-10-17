package transport

import (
	"net/http"
)

// Header is a decorator that injects a header value into every request.
type Header struct {
	wrapped          http.RoundTripper
	requestProvider  HeaderProvider
	responseProvider ResponseHeaderProvider
}

// HeaderProvider is mapping function that generates the required header name
// and value from an outgoing request.
type HeaderProvider func(*http.Request) (headerName string, headerValue string)

// RoundTrip annotates the outgoing request and calls the wrapped Client.
func (c *Header) RoundTrip(r *http.Request) (*http.Response, error) {
	if c.requestProvider != nil {
		var name, value = c.requestProvider(r)
		r.Header.Set(name, value)
	}
	resp, err := c.wrapped.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if c.responseProvider != nil {
		var name, value = c.responseProvider(resp)
		resp.Header.Set(name, value)
	}
	return resp, nil
}

// NewHeader wraps a transport in order to include custom request headers.
func NewHeader(requestProvider HeaderProvider) func(http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		return &Header{
			wrapped:         c,
			requestProvider: requestProvider,
		}
	}
}

// ResponseHeaderProvider is mapping function that generates the required header name
// and value to an outgoing response.
type ResponseHeaderProvider func(*http.Response) (headerName string, headerValue string)

// NewHeaders wraps a transport in order to include custom request and response headers.
func NewHeaders(requestProvider HeaderProvider, responseProvider ResponseHeaderProvider) func(http.RoundTripper) http.RoundTripper {
	return func(c http.RoundTripper) http.RoundTripper {
		return &Header{
			wrapped:          c,
			requestProvider:  requestProvider,
			responseProvider: responseProvider,
		}
	}
}
