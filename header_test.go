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

func TestRequestHeaderAddsHeaders(t *testing.T) {
	const value string = "VALUE"
	t.Parallel()
	var provider = func(*http.Request) (string, string) {
		return "KEY", value
	}

	var fixture = &fixtureHeaderTransport{}
	var client = NewHeader(provider)(fixture)
	var r, _ = http.NewRequest("GET", "/", nil)
	_, _ = client.RoundTrip(r)
	if fixture.Request.Header.Get("KEY") != value {
		t.Fatal("Decorator did not add headers to the request.")
	}
}

func TestResponseHeaderAddsHeaders(t *testing.T) {
	const value string = "VALUE"
	t.Parallel()
	var provider = func(*http.Response) (string, string) {
		return "KEY", value
	}

	resp := &http.Response{Header: http.Header{}}
	var fixture = &fixtureHeaderTransport{Response: resp}
	var client = NewHeaders(nil, provider)(fixture)
	var r, _ = http.NewRequest("GET", "/", nil)
	modifiedResp, _ := client.RoundTrip(r)
	if fixture.Request.Header.Get("KEY") == value {
		t.Fatal("Decorator should not add headers to the request.")
	}
	if modifiedResp.Header.Get("KEY") != value {
		t.Fatal("Decorator did not add headers to the response.")
	}
}
