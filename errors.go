package transport

import (
	"context"
	"errors"
	"net/http"
)

// ErrorToStatusCode attempts to translate an HTTP request error to a meaningful
// HTTP status code.
//
// There can be many reasons why we couldn't get a proper response from the upstream server.
// This includes timeouts, inability to connect, or the client canceling a request. If the
// error appears to indicate a timeout occurred, then return 504 Gateway Timeout, otherwise,
// return 502 Bad Gateway.
func ErrorToStatusCode(err error) int {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}
