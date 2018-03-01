package transport

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

// Requests contain mutable state that is altered on each pass through a
// Transport. In several ways, the state is mutated to the point that it cannot
// be reused. This component was introduced to account for cases where the
// request body was drained partially or completely but we want to re-issue the
// request.
type requestCopier struct {
	original *http.Request
	body     []byte
}

func newRequestCopier(r *http.Request) (*requestCopier, error) {
	var body []byte
	var e error
	if r.Body != nil {
		body, e = ioutil.ReadAll(r.Body)
	}
	return &requestCopier{r, body}, e
}

func (r *requestCopier) Copy() *http.Request {
	var newRequest = new(http.Request)
	*newRequest = *r.original
	newRequest.Body = nil
	if r.body != nil {
		newRequest.Body = ioutil.NopCloser(bytes.NewBuffer(r.body))
		newRequest.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewBuffer(r.body)), nil
		}
	}
	return newRequest
}

type retryFunc func(*http.Request, *http.Response, error) bool

// RetrierOption is a configuration value for the Retrier decorator.
type RetrierOption func(*Retrier) *Retrier

// RetrierOptionLimit applies a maximum retry count for all retry operations.
func RetrierOptionLimit(limit int) RetrierOption {
	return func(r *Retrier) *Retrier {
		r.retryLimit = limit
		return r
	}
}

// RetrierOptionDelay installs a sleep interval in between retries.
func RetrierOptionDelay(delay time.Duration) RetrierOption {
	return func(r *Retrier) *Retrier {
		r.retryDelay = delay
		return r
	}
}

// RetrierOptionDelayJitter installs a random jitter of plus or minus the
// given value when sleeping between retries.
func RetrierOptionDelayJitter(jitter time.Duration) RetrierOption {
	return func(r *Retrier) *Retrier {
		r.retryDelayJitter = jitter
		return r
	}
}

// RetrierOptionTimeout sets a maximum duration for a round trip. If the limit
// is hit then the request is cancelled and retried. This is useful for
// overcoming transient networking issues and delays.
func RetrierOptionTimeout(timeout time.Duration) RetrierOption {
	return func(r *Retrier) *Retrier {
		r.pre = append(r.pre, func(req *http.Request) (*http.Request, func()) {
			var ctx, cancel = context.WithTimeout(req.Context(), timeout)
			return req.WithContext(ctx), cancel
		})
		r.retries = append(r.retries, func(req *http.Request, resp *http.Response, e error) bool {
			if req.Context().Err() == context.DeadlineExceeded {
				return true
			}
			return false
		})
		return r
	}
}

// RetrierOptionResponseCode configures the client to retry requests if certain
// status codes are seen in the response.
func RetrierOptionResponseCode(codes ...int) RetrierOption {
	return func(r *Retrier) *Retrier {
		r.retries = append(r.retries, func(req *http.Request, resp *http.Response, e error) bool {
			for _, code := range codes {
				if resp.StatusCode == code {
					return true
				}
			}
			return false
		})
		return r
	}
}

// Retrier is a wrapper for applying various retry policies to requests.
type Retrier struct {
	wrapped          http.RoundTripper
	retryLimit       int
	retryDelay       time.Duration
	retryDelayJitter time.Duration
	pre              []func(*http.Request) (*http.Request, func())
	retries          []retryFunc
}

// RoundTrip executes a request and applies one or more retry policies.
func (c *Retrier) RoundTrip(r *http.Request) (*http.Response, error) {
	var copier, e = newRequestCopier(r)
	var parentCtx = r.Context()
	if e != nil {
		return nil, e
	}
	var response *http.Response
OUTER:
	for x := 0; x <= c.retryLimit; x = x + 1 {
		r = copier.Copy()
		var finaliser func()
		for _, pre := range c.pre {
			r, finaliser = pre(r)
			if finaliser != nil {
				defer finaliser()
			}
		}
		response, e = c.wrapped.RoundTrip(r)
		for _, retry := range c.retries {
			if retry(r, response, e) {
				select {
				case <-parentCtx.Done():
					return response, parentCtx.Err()
				case <-time.After(c.retryDelay + time.Duration(rand.Float64()*float64(c.retryDelayJitter))):
					continue OUTER
				}
			}
		}
		return response, e
	}
	return response, e
}

// NewRetrier configures a RoundTripper decorator to perform some number of
// retries.
func NewRetrier(opts ...RetrierOption) func(http.RoundTripper) http.RoundTripper {
	return func(wrapped http.RoundTripper) http.RoundTripper {
		var r = &Retrier{wrapped: wrapped}
		for _, opt := range opts {
			r = opt(r)
		}
		if r.retryLimit < 0 {
			r.retryLimit = 0
		}
		return r
	}
}
