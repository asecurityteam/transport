package transport

import "net/http"

// RoundTripperFunc is modelled after the http.HandlerFunc and converts a
// function matching the RoundTrip signature to a RoundTripper implementation.
type RoundTripperFunc func(r *http.Request) (*http.Response, error)

// RoundTrip calls the wrapped function.
func (d RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return d(r)
}
