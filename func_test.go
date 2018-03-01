package transport

import (
	"net/http"
	"testing"
)

func TestRoundTripperFunc(t *testing.T) {
	var called bool
	var found *http.Request
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var c http.RoundTripper = RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		found = r
		return nil, nil
	})

	c.RoundTrip(req)
	if !called || found != req {
		t.Fatal("did not call wrapped function with request")
	}
}
