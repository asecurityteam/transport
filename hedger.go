package transport

import (
	"context"
	"net/http"
	"time"
)

// Hedger is a wrapper that fans out a new request at each time interval defined
// by the backoff policy, and returns the first response received. For
// latency-based retries, this will often be a better approach than a
// "stop-and-retry" policy (such as the TimeoutRetrier). The hedging decorator
// allows for a worst case request to take up to a maximum configurable timeout,
// while pessimistically creating new requests before the timeout is reached.
type Hedger struct {
	wrapped       http.RoundTripper
	backoffPolicy BackoffPolicy
}

type hedgedResponse struct {
	Response *http.Response
	Err      error
}

func (c *Hedger) hedgedRoundTrip(doneCtx context.Context, requestCtx context.Context, r *http.Request, resp chan *hedgedResponse) { // nolint
	// Create a local context to manage the request cancellation. Because these
	// are all children of the source parentCtx they will eventually be
	// canceled when the parent is canceled even if we do not call the cancel
	// method returned here. The implication is that the source parent context
	// _must_ end at some point. That is, a background context with no end of
	// life would cause resources and memory to leak over time.
	ctx, cancel := context.WithCancel(requestCtx) // nolint
	// Create a local channel for accepting the results. This allows us to
	// sink the result and close the goroutine under all conditions including
	// if the context is canceled because it has a buffer space of one. If it is
	// never read from then it will eventually be GC'd after the method exits.
	localResp := make(chan *hedgedResponse, 1)
	go func() {
		var response, err = c.wrapped.RoundTrip(r.WithContext(ctx))
		localResp <- &hedgedResponse{Response: response, Err: err}
	}()

	select {
	case resp <- <-localResp:
	case <-doneCtx.Done():
		// End work in flight if the parent signals that it needs no more
		// responses. Because the response channel is unbuffered, all responses
		// that complete will block on this select until they are read. The
		// hedger will read only one of them and then trigger the Done() case
		// for all other
		cancel()
	}
} // nolint

// RoundTrip executes a new request at each time interval defined
// by the backoff policy, and returns the first response received.
func (c *Hedger) RoundTrip(r *http.Request) (*http.Response, error) {
	var copier, e = newRequestCopier(r)
	if e != nil {
		return nil, e
	}
	var parentCtx = r.Context()
	// doneCtx is used to indicate that the RoundTrip is complete and any
	// outstanding work should be canceled.
	var doneCtx, done = context.WithCancel(parentCtx)
	defer done()
	// requestCtx is a copy of parentCtx without any modifications. This could
	// likely just be parentCtx directly. Making a child out of habit.
	requestCtx, _ := context.WithCancel(parentCtx) // nolint

	var backoffer = c.backoffPolicy()
	var respChan = make(chan *hedgedResponse)
	var request = copier.Copy()

	go c.hedgedRoundTrip(doneCtx, requestCtx, request, respChan)

	for {
		select {
		case resp := <-respChan:
			return resp.Response, resp.Err
		case <-parentCtx.Done():
			return nil, parentCtx.Err()
		case <-time.After(backoffer.Backoff(r, nil, nil)):
			request = copier.Copy()
			go c.hedgedRoundTrip(doneCtx, requestCtx, request, respChan)
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
