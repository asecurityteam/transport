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
	return &requestCopier{original: r, body: body}, e
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

// Retrier determines whether or not the transport will automatically retry
// a request.
type Retrier interface {
	Retry(*http.Request, *http.Response, error) bool
}

// Requester can be implemented if the Retrier needs to manipulate the request
// or request context before it is executed.
type Requester interface {
	Request(*http.Request) *http.Request
}

// RetryPolicy is a factory that generates a Retrier.
type RetryPolicy func() Retrier

// Backoffer determines how much time to wait in between automated retires.
type Backoffer interface {
	Backoff(*http.Request, *http.Response, error) time.Duration
}

// BackoffPolicy is a factory that generates a Backoffer.
type BackoffPolicy func() Backoffer

// LimitedRetrier wraps a series of retry policies in a hard upper limit.
type LimitedRetrier struct {
	limit    int
	attempts int
	retries  []Retrier
}

// NewLimitedRetryPolicy wraps a series of retry policies in an upper limit.
func NewLimitedRetryPolicy(limit int, policies ...RetryPolicy) RetryPolicy {
	return func() Retrier {
		var retries = make([]Retrier, 0, len(policies))
		for _, policy := range policies {
			retries = append(retries, policy())
		}
		return &LimitedRetrier{
			limit:    limit,
			attempts: 0,
			retries:  retries,
		}
	}
}

// Request implements Requester by calling the wrapped Request methods where
// needed.
func (r *LimitedRetrier) Request(req *http.Request) *http.Request {
	for _, retry := range r.retries {
		if requester, ok := retry.(Requester); ok {
			req = requester.Request(req)
		}
	}
	return req
}

// Retry the request based on the wrapped policies until the limit is reached.
// Once the limit is reached then this method always returns false.
func (r *LimitedRetrier) Retry(req *http.Request, resp *http.Response, e error) bool {
	if r.attempts >= r.limit {
		return false
	}
	r.attempts = r.attempts + 1
	for _, retry := range r.retries {
		if retry.Retry(req, resp, e) {
			return true
		}
	}
	return false
}

// StatusCodeRetrier retries based on HTTP status codes.
type StatusCodeRetrier struct {
	codes []int
}

// Retry the request if the response has a valid code that matches on of the
// codes given in the retry set.
func (r *StatusCodeRetrier) Retry(req *http.Request, resp *http.Response, e error) bool {
	for _, code := range r.codes {
		if resp != nil && resp.StatusCode == code {
			return true
		}
	}
	return false
}

// NewStatusCodeRetryPolicy generates a RetryPolicy that retries on specified
// status codes in the HTTP response.
func NewStatusCodeRetryPolicy(codes ...int) RetryPolicy {
	var retrier = &StatusCodeRetrier{codes: codes}
	return func() Retrier {
		return retrier
	}
}

// TimeoutRetrier applies a timeout to requests and retries if the request
// took longer than the timeout duration.
type TimeoutRetrier struct {
	timeout time.Duration
}

// NewTimeoutRetryPolicy generates a RetryPolicy that ends a request after a
// given timeout duration and tries again.
func NewTimeoutRetryPolicy(timeout time.Duration) RetryPolicy {
	var retrier = &TimeoutRetrier{timeout: timeout}
	return func() Retrier {
		return retrier
	}
}

// Retry the request if the context exceeded the deadline.
func (r *TimeoutRetrier) Retry(req *http.Request, resp *http.Response, e error) bool {
	return e == context.DeadlineExceeded
}

// Request adds a timeout to the request context.
func (r *TimeoutRetrier) Request(req *http.Request) *http.Request {
	var ctx, _ = context.WithTimeout(req.Context(), r.timeout) // nolint
	return req.WithContext(ctx)
}

// FixedBackoffer signals the client to wait for a static amount of time.
type FixedBackoffer struct {
	wait time.Duration
}

// NewFixedBackoffPolicy generates a BackoffPolicy that always returns the
// same value.
func NewFixedBackoffPolicy(wait time.Duration) BackoffPolicy {
	var backoffer = &FixedBackoffer{wait: wait}
	return func() Backoffer {
		return backoffer
	}
}

// Backoff for a static amount of time.
func (b *FixedBackoffer) Backoff(*http.Request, *http.Response, error) time.Duration {
	return b.wait
}

// PercentJitteredBackoffer adjusts the backoff time by a random amount within
// N percent of the duration to help with thundering herds.
type PercentJitteredBackoffer struct {
	wrapped Backoffer
	jitter  float64
	random  func() float64
}

// NewPercentJitteredBackoffPolicy wraps any backoff policy and applies a
// percentage based jitter to the original policy's value. The percentage float
// should be between 0 and 1. The jitter will be applied as a positive and
// negative value equally.
func NewPercentJitteredBackoffPolicy(wrapped BackoffPolicy, jitterPercent float64) BackoffPolicy {
	return func() Backoffer {
		return &PercentJitteredBackoffer{
			wrapped: wrapped(),
			jitter:  jitterPercent,
			random:  rand.Float64,
		}
	}
}

func calculateJitteredBackoff(original time.Duration, percentage float64, random func() float64) time.Duration {
	var jitterWindow = time.Duration(percentage * float64(original))
	var jitter = time.Duration(random() * float64(jitterWindow))
	if random() > .5 {
		jitter = -jitter
	}
	return original + jitter
}

// Backoff for a jittered amount.
func (b *PercentJitteredBackoffer) Backoff(r *http.Request, response *http.Response, e error) time.Duration {
	var d = b.wrapped.Backoff(r, response, e)
	return calculateJitteredBackoff(d, b.jitter, b.random)
}

// Retry is a wrapper for applying various retry policies to requests.
type Retry struct {
	wrapped       http.RoundTripper
	backoffPolicy BackoffPolicy
	retryPolicies []RetryPolicy
}

// RoundTrip executes a request and applies one or more retry policies.
func (c *Retry) RoundTrip(r *http.Request) (*http.Response, error) {
	var copier, e = newRequestCopier(r)
	var parentCtx = r.Context()
	if e != nil {
		return nil, e
	}
	var response *http.Response
	var requestCtx, cancel = context.WithCancel(parentCtx)
	defer cancel()
	var req = copier.Copy().WithContext(requestCtx)

	var retriers = make([]Retrier, 0, len(c.retryPolicies))
	var backoffer = c.backoffPolicy()
	for _, retryPolicy := range c.retryPolicies {
		retriers = append(retriers, retryPolicy())
	}
	for _, retrier := range retriers {
		if requester, ok := retrier.(Requester); ok {
			req = requester.Request(req)
		}
	}

	response, e = c.wrapped.RoundTrip(req)
	for c.shouldRetry(r, response, e, retriers) {
		select {
		case <-parentCtx.Done():
			return nil, parentCtx.Err()
		case <-time.After(backoffer.Backoff(r, response, e)):
		}
		requestCtx, cancel = context.WithCancel(parentCtx)
		defer cancel()
		var req = copier.Copy().WithContext(requestCtx)
		for _, retrier := range retriers {
			if requester, ok := retrier.(Requester); ok {
				req = requester.Request(req)
			}
		}
		response, e = c.wrapped.RoundTrip(req)
	}
	return response, e
}

func (c *Retry) shouldRetry(r *http.Request, response *http.Response, e error, retriers []Retrier) bool {
	for _, retrier := range retriers {
		if retrier.Retry(r, response, e) {
			return true
		}
	}
	return false
}

// NewRetrier configures a RoundTripper decorator to perform some number of
// retries.
func NewRetrier(backoffPolicy BackoffPolicy, retryPolicies ...RetryPolicy) func(http.RoundTripper) http.RoundTripper {
	return func(wrapped http.RoundTripper) http.RoundTripper {
		return &Retry{wrapped: wrapped, backoffPolicy: backoffPolicy, retryPolicies: retryPolicies}
	}
}
