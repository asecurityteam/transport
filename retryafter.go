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
	wrapped       http.RoundTripper
	backoffPolicy BackoffPolicy
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

	var backoffer = c.backoffPolicy()
	var retryAfter time.Duration
	for {
		if retryAfter > 0 {
			select {
			case <-parentCtx.Done():
				cancel()
				return nil, parentCtx.Err()
			case <-time.After(retryAfter):
			}
			cancel()
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
				retryAfter = backoffer.Backoff(r, response, e)
			} else {
				var retryAfterInt int
				var err error
				if retryAfterInt, err = strconv.Atoi(retryAfterString); err != nil {
					break
				}
				retryAfter = time.Duration(retryAfterInt) * time.Millisecond
			}
		}
	}
	if e != nil {
		cancel()
	}
	return response, e // nolint
}

// NewRetryAfter configures a RoundTripper decorator to honor a status code 429 response,
// using the Retry-After header directive when present, or the backoffPolicy if not present.
func NewRetryAfter() func(http.RoundTripper) http.RoundTripper {
	return func(wrapped http.RoundTripper) http.RoundTripper {
		return &RetryAfter{wrapped: wrapped, backoffPolicy: NewExponentialBackoffPolicy(20 * time.Millisecond)}
	}
}
