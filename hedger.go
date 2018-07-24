package transport

import (
	"context"
	"net/http"
	"time"
)

// Hedger is a wrapper that fans out a new request at each time interval defined
// by the backoff policy, and returns the first response received.
type Hedger struct {
	wrapped       http.RoundTripper
	backoffPolicy BackoffPolicy
}

type hedgedResponse struct {
	Response *http.Response
	Err      error
}

func (c *Hedger) hedgedRoundTrip(ctx context.Context, r *http.Request, resp chan *hedgedResponse) {
	var response, err = c.wrapped.RoundTrip(r)
	select {
	case resp <- &hedgedResponse{Response: response, Err: err}:
	case <-ctx.Done():
	}
}

// RoundTrip executes a new request at each time interval defined
// by the backoff policy, and returns the first response received.
func (c *Hedger) RoundTrip(r *http.Request) (*http.Response, error) {
	var copier, e = newRequestCopier(r)
	if e != nil {
		return nil, e
	}
	var parentCtx = r.Context()
	var backoffer = c.backoffPolicy()
	var respChan = make(chan *hedgedResponse)
	var requestCtx, cancel = context.WithCancel(parentCtx)
	defer cancel()
	var request = copier.Copy().WithContext(requestCtx)

	go c.hedgedRoundTrip(requestCtx, request, respChan)

	for {
		select {
		case resp := <-respChan:
			return resp.Response, resp.Err
		case <-parentCtx.Done():
			return nil, parentCtx.Err()
		case <-time.After(backoffer.Backoff(r, nil, nil)):
			request = copier.Copy().WithContext(requestCtx)
			go c.hedgedRoundTrip(requestCtx, request, respChan)
		}
	}
}

// NewHedger configures a RoundTripper decorator to perform some number of
// hedged requests.
func NewHedger(backoffPolicy BackoffPolicy) func(http.RoundTripper) http.RoundTripper {
	return func(wrapped http.RoundTripper) http.RoundTripper {
		return &Hedger{wrapped: wrapped, backoffPolicy: backoffPolicy}
	}
}
