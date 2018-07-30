package transport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Factory is any function that takes no arguments and returns a Transport.
type Factory func() http.RoundTripper

// Option is a function that either modifies or generates a new Transport.
type Option func(*http.Transport) *http.Transport

// OptionProxy installs a custom Proxy configuration in the Transport.
func OptionProxy(proxy func(*http.Request) (*url.URL, error)) Option {
	return func(t *http.Transport) *http.Transport {
		t.Proxy = proxy
		return t
	}
}

// OptionDialContext installs a custom DialContext configuration in the Transport.
func OptionDialContext(dialCtx func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(t *http.Transport) *http.Transport {
		t.DialContext = dialCtx
		return t
	}
}

// OptionDial installs a custom Dial configuration in the Transport.
func OptionDial(dial func(network, addr string) (net.Conn, error)) Option {
	return func(t *http.Transport) *http.Transport {
		t.Dial = dial //nolint, Dial func is deprecated
		return t
	}
}

// OptionDialTLS installs a custom DialTLS configuration in the Transport.
func OptionDialTLS(dial func(network, addr string) (net.Conn, error)) Option {
	return func(t *http.Transport) *http.Transport {
		t.DialTLS = dial
		return t
	}
}

// OptionTLSClientConfig installs a custom TLSClientConfig in the Transport.
func OptionTLSClientConfig(config *tls.Config) Option {
	return func(t *http.Transport) *http.Transport {
		t.TLSClientConfig = config
		return t
	}
}

// OptionTLSHandshakeTimeout installs a custom TLSHandshakeTimeout in the Transport.
func OptionTLSHandshakeTimeout(timeout time.Duration) Option {
	return func(t *http.Transport) *http.Transport {
		t.TLSHandshakeTimeout = timeout
		return t
	}
}

// OptionDisableKeepAlives installs a custom DisableKeepAlives option in the Transport.
func OptionDisableKeepAlives(disabled bool) Option {
	return func(t *http.Transport) *http.Transport {
		t.DisableKeepAlives = disabled
		return t
	}
}

// OptionDisableCompression installs a custom DisableCompression option in the Transport.
func OptionDisableCompression(disabled bool) Option {
	return func(t *http.Transport) *http.Transport {
		t.DisableCompression = disabled
		return t
	}
}

// OptionMaxIdleConns installs a custom MaxIdleConns option in the Transport.
func OptionMaxIdleConns(max int) Option {
	return func(t *http.Transport) *http.Transport {
		t.MaxIdleConns = max
		return t
	}
}

// OptionMaxIdleConnsPerHost installs a custom MaxIdleConnsPerHost option in the Transport.
func OptionMaxIdleConnsPerHost(max int) Option {
	return func(t *http.Transport) *http.Transport {
		t.MaxIdleConnsPerHost = max
		return t
	}
}

// OptionIdleConnTimeout installs a custom IdleConnTimeout option in the Transport.
func OptionIdleConnTimeout(timeout time.Duration) Option {
	return func(t *http.Transport) *http.Transport {
		t.IdleConnTimeout = timeout
		return t
	}
}

// OptionResponseHeaderTimeout installs a custom ResponseHeaderTimeout option in the Transport.
func OptionResponseHeaderTimeout(timeout time.Duration) Option {
	return func(t *http.Transport) *http.Transport {
		t.ResponseHeaderTimeout = timeout
		return t
	}
}

// OptionExpectContinueTimeout installs a custom ExpectContinueTimeout option in the Transport.
func OptionExpectContinueTimeout(timeout time.Duration) Option {
	return func(t *http.Transport) *http.Transport {
		t.ExpectContinueTimeout = timeout
		return t
	}
}

// OptionTLSNextProto installs a custom TLSNextProto option in the Transport.
func OptionTLSNextProto(next map[string]func(authority string, c *tls.Conn) http.RoundTripper) Option {
	return func(t *http.Transport) *http.Transport {
		t.TLSNextProto = next
		return t
	}
}

// OptionProxyConnectHeader installs a custom ProxyConnectHeader option in the Transport.
func OptionProxyConnectHeader(header http.Header) Option {
	return func(t *http.Transport) *http.Transport {
		t.ProxyConnectHeader = header
		return t
	}
}

// OptionMaxResponseHeaderBytes installs a custom MaxResponseHeaderBytes option in the Transport.
func OptionMaxResponseHeaderBytes(max int64) Option {
	return func(t *http.Transport) *http.Transport {
		t.MaxResponseHeaderBytes = max
		return t
	}
}

// OptionDefaultTransport configures a transport to match the http.DefaultTransport.
func OptionDefaultTransport(t *http.Transport) *http.Transport {
	t = OptionProxy(http.ProxyFromEnvironment)(t)
	t = OptionDialContext((&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext)(t)
	t = OptionMaxIdleConns(100)(t)
	t = OptionIdleConnTimeout(90 * time.Second)(t)
	t = OptionTLSHandshakeTimeout(10 * time.Second)(t)
	t = OptionExpectContinueTimeout(time.Second)(t)
	return t
}

// New applies the given options to a Transport and returns it.
func New(opts ...Option) *http.Transport {
	var t = &http.Transport{}
	for _, opt := range opts {
		t = opt(t)
	}
	return t
}

// NewFactory returns a Factory that is bound to the given Option set.
func NewFactory(opts ...Option) Factory {
	return func() http.RoundTripper {
		return New(opts...)
	}
}
