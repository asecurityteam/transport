package transport

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// RetryAfter determines whether or not the transport will automatically retry
// a request based on configured behaviors for 429 responses with Retry-After header.
type RetryAfter struct {
	wrapped http.RoundTripper
}

// RoundTrip executes a request and applies one or more retry policies.
func (c *RetryAfter) RoundTrip(r *http.Request) (*http.Response, error) {
	var copier, e = newRequestCopier(r)
	var parentCtx = r.Context()
	if e != nil {
		return nil, e
	}
	var response *http.Response
	var requestCtx, cancel = context.WithCancel(parentCtx)
	var req = copier.Copy().WithContext(requestCtx)

	retryAfter := 0
	for {
		if retryAfter > 0 {
			select {
			case <-parentCtx.Done():
				cancel()
				return nil, parentCtx.Err()
			case <-time.After(time.Duration(retryAfter) * time.Millisecond):
			}
			requestCtx, cancel = context.WithCancel(parentCtx) // nolint
			req = copier.Copy().WithContext(requestCtx)
		}
		response, e = c.wrapped.RoundTrip(req)
		if e != nil {
			break
		}
		if response.StatusCode != 429 {
			break
		} else {
			retryAfterString := response.Header.Get("Retry-After")
			if retryAfterString == "" {
				break
			}
			var err error
			if retryAfter, err = strconv.Atoi(retryAfterString); err != nil {
				break
			}
		}
	}
	if e != nil {
		cancel()
	}
	return response, e // nolint
}

// NewRetryAfter configures a RoundTripper decorator to perform some number of
// retries.
func NewRetryAfter() func(http.RoundTripper) http.RoundTripper {
	return func(wrapped http.RoundTripper) http.RoundTripper {
		return &RetryAfter{wrapped: wrapped}
	}
}
