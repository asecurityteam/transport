package transport

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"
)

type hedgerFixtureRoundTripper struct {
	Response *http.Response
	Err      error
	Sleep    time.Duration
	Counter  int
	l        sync.RWMutex
}

func (c *hedgerFixtureRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	c.incrCounter()
	time.Sleep(c.Sleep)
	return c.Response, c.Err
}

func (c *hedgerFixtureRoundTripper) incrCounter() {
	c.l.Lock()
	defer c.l.Unlock()
	c.Counter = c.Counter + 1
}

func (c *hedgerFixtureRoundTripper) Count() int {
	c.l.Lock()
	defer c.l.Unlock()
	return c.Counter
}

func TestHedger(t *testing.T) {
	t.Parallel()

	var successResponse = &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{},
	}
	var testCases = []struct {
		name           string
		roundTripper   *hedgerFixtureRoundTripper
		expectedResp   *http.Response
		err            error
		backoffTime    time.Duration
		contextTimeout time.Duration
		called         int
	}{
		{
			name: "Single successful call",
			roundTripper: &hedgerFixtureRoundTripper{
				Response: successResponse,
				Err:      nil,
				Sleep:    0},
			expectedResp:   successResponse,
			err:            nil,
			backoffTime:    5 * time.Millisecond,
			contextTimeout: 3 * time.Millisecond,
			called:         1,
		},
		{
			name: "Single failed call due to context timeout",
			roundTripper: &hedgerFixtureRoundTripper{
				Response: successResponse,
				Err:      nil,
				Sleep:    4 * time.Millisecond},
			expectedResp:   nil,
			err:            context.DeadlineExceeded,
			backoffTime:    5 * time.Millisecond,
			contextTimeout: 3 * time.Millisecond,
			called:         1,
		},
		{
			name: "Three calls",
			roundTripper: &hedgerFixtureRoundTripper{
				Response: successResponse,
				Err:      nil,
				Sleep:    9 * time.Millisecond},
			expectedResp:   successResponse,
			err:            nil,
			backoffTime:    5 * time.Millisecond,
			contextTimeout: 12 * time.Millisecond,
			called:         2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			var transport = tc.roundTripper
			var backoffTime = tc.backoffTime
			var decorator = NewHedger(NewFixedBackoffPolicy(backoffTime))
			var client = &http.Client{
				Transport: decorator(transport),
			}
			var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
			var timeoutCtx, cancel = context.WithTimeout(context.Background(), tc.contextTimeout)
			defer cancel()
			req = req.WithContext(timeoutCtx)
			var resp, er = client.Transport.RoundTrip(req)
			if resp != tc.expectedResp || er != tc.err {
				t.Fatalf("Got resp %v and err %v, expected resp %v and err %v", resp, er, tc.expectedResp, tc.err)
			}
			if transport.Count() != tc.called {
				t.Fatalf("Called decorator %d times, expected %d", transport.Count(), tc.called)
			}
		})
	}

}
