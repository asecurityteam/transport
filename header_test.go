package transport

import (
	"net/http"
	"testing"
)

type fixtureHeaderTransport struct {
	Response *http.Response
	Request  *http.Request
	Err      error
}

func (c *fixtureHeaderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.Request = r
	return c.Response, c.Err
}

func TestHeaderAddsHeaders(t *testing.T) {
	t.Parallel()
	var provider = func(*http.Request) (string, string) {
		return "KEY", "VALUE"
	}

	var fixture = &fixtureHeaderTransport{}
	var client = NewHeader(provider)(fixture)
	var r, _ = http.NewRequest("GET", "/", nil)
	_, _ = client.RoundTrip(r)
	if fixture.Request.Header.Get("KEY") != "VALUE" {
		t.Fatal("Decorator did not add headers to the request.")
	}
}
