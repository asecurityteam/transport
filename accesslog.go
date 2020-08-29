package transport

import (
	"net/http"
	"strconv"
	"time"

	"github.com/asecurityteam/logevent"
)

type accessLog struct {
	Time                   string `logevent:"@timestamp"`
	Schema                 string `logevent:"schema,default=access"`
	Host                   string `logevent:"host"`
	Site                   string `logevent:"site"`
	HTTPRequestContentType string `logevent:"http_request_content_type"`
	HTTPMethod             string `logevent:"http_method"`
	HTTPReferrer           string `logevent:"http_referrer"`
	HTTPUserAgent          string `logevent:"http_user_agent"`
	URIPath                string `logevent:"uri_path"`
	URIQuery               string `logevent:"uri_query"`
	Scheme                 string `logevent:"scheme"`
	Port                   int    `logevent:"port"`
	Duration               int    `logevent:"duration"`
	HTTPContentType        string `logevent:"http_content_type"`
	Status                 int    `logevent:"status"`
	Message                string `logevent:"message,default=access"`
}

type loggingTransport struct {
	Wrapped http.RoundTripper
}

// RoundTrip writes structured access logs for the request.
func (c *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var dstPortStr = r.URL.Port()
	var dstPort, _ = strconv.Atoi(dstPortStr)
	var a = accessLog{
		Time:                   time.Now().UTC().Format(time.RFC3339Nano),
		Host:                   r.URL.Hostname(),
		Port:                   dstPort,
		Site:                   r.Host,
		HTTPRequestContentType: r.Header.Get("Content-Type"),
		HTTPMethod:             r.Method,
		HTTPReferrer:           r.Referer(),
		HTTPUserAgent:          r.UserAgent(),
		URIPath:                r.URL.Path,
		URIQuery:               r.URL.Query().Encode(),
		Scheme:                 r.URL.Scheme,
	}
	var start = time.Now()
	var resp, e = c.Wrapped.RoundTrip(r)
	a.Duration = int(time.Since(start).Nanoseconds() / 1e6)
	if e == nil {
		a.Status = resp.StatusCode
		a.HTTPContentType = resp.Header.Get("Content-Type")
	} else {
		a.Status = ErrorToStatusCode(e)
	}
	logevent.FromContext(r.Context()).Info(a)
	return resp, e
}

// NewAccessLog configures a RoundTripper decorator that generates log
// details for each request.
func NewAccessLog() func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return &loggingTransport{Wrapped: next}
	}
}
