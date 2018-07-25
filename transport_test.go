package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
)

type optionTestCase struct {
	Name     string
	Option   Option
	Verifier func(*http.Transport) error
}

func TestTransportOptions(t *testing.T) {
	var testErr = errors.New("")
	var proxyFunc = func(*http.Request) (*url.URL, error) {
		return nil, testErr
	}
	var dialCtxFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, testErr
	}
	var dialFunc = func(network, addr string) (net.Conn, error) {
		return nil, testErr
	}
	var dialTLSFunc = func(network, addr string) (net.Conn, error) {
		return nil, testErr
	}
	var tlsConfig = &tls.Config{}
	var nextProto = map[string]func(authority string, c *tls.Conn) http.RoundTripper{
		"test": nil,
	}
	var header = http.Header{
		http.CanonicalHeaderKey("test"): nil,
	}
	var testCases = []optionTestCase{
		{Name: "OptionProxy", Option: OptionProxy(proxyFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.Proxy(nil)
			if e != testErr {
				return errors.New("proxy function was not set by OptionProxy")
			}
			return nil
		}},
		{Name: "OptionDialContext", Option: OptionDialContext(dialCtxFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.DialContext(context.Background(), "", "")
			if e != testErr {
				return errors.New("dial function was not set by OptionDialContext")
			}
			return nil
		}},
		{Name: "OptionDial", Option: OptionDial(dialFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.Dial("", "") //nolint, Dial func is deprecated
			if e != testErr {
				return errors.New("dial function was not set by OptionDial")
			}
			return nil
		}},
		{Name: "OptionDialTLS", Option: OptionDialTLS(dialTLSFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.DialTLS("", "")
			if e != testErr {
				return errors.New("dial function was not set by OptionDialTLS")
			}
			return nil
		}},
		{Name: "OptionTLSClientConfig", Option: OptionTLSClientConfig(tlsConfig), Verifier: func(tr *http.Transport) error {
			if tr.TLSClientConfig != tlsConfig {
				return errors.New("tls config was not set by OptionTLSClientConfig")
			}
			return nil
		}},
		{Name: "OptionTLSHandshakeTimeout", Option: OptionTLSHandshakeTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.TLSHandshakeTimeout != 1 {
				return errors.New("timeout was not set by OptionTLSHandshakeTimeout")
			}
			return nil
		}},
		{Name: "OptionDisableKeepAlives", Option: OptionDisableKeepAlives(true), Verifier: func(tr *http.Transport) error {
			if !tr.DisableKeepAlives {
				return errors.New("keep alive was not set by OptionDisableKeepAlives")
			}
			return nil
		}},
		{Name: "OptionDisableCompression", Option: OptionDisableCompression(true), Verifier: func(tr *http.Transport) error {
			if !tr.DisableCompression {
				return errors.New("disable compression was not set by OptionDisableCompression")
			}
			return nil
		}},
		{Name: "OptionMaxIdleConns", Option: OptionMaxIdleConns(1), Verifier: func(tr *http.Transport) error {
			if tr.MaxIdleConns != 1 {
				return errors.New("idle conns were not set by OptionMaxIdleConns")
			}
			return nil
		}},
		{Name: "OptionMaxIdleConnsPerHost", Option: OptionMaxIdleConnsPerHost(1), Verifier: func(tr *http.Transport) error {
			if tr.MaxIdleConnsPerHost != 1 {
				return errors.New("idle conns were not set by OptionMaxIdleConnsPerHost")
			}
			return nil
		}},
		{Name: "OptionIdleConnTimeout", Option: OptionIdleConnTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.IdleConnTimeout != 1 {
				return errors.New("timeout was not set by OptionIdleConnTimeout")
			}
			return nil
		}},
		{Name: "OptionResponseHeaderTimeout", Option: OptionResponseHeaderTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.ResponseHeaderTimeout != 1 {
				return errors.New("timeout was not set by OptionResponseHeaderTimeout")
			}
			return nil
		}},
		{Name: "OptionExpectContinueTimeout", Option: OptionExpectContinueTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.ExpectContinueTimeout != 1 {
				return errors.New("timeout was not set by OptionExpectContinueTimeout")
			}
			return nil
		}},
		{Name: "OptionTLSNextProto", Option: OptionTLSNextProto(nextProto), Verifier: func(tr *http.Transport) error {
			if _, ok := tr.TLSNextProto["test"]; !ok {
				return errors.New("next proto was not set by OptionTLSNextProto")
			}
			return nil
		}},
		{Name: "OptionProxyConnectHeader", Option: OptionProxyConnectHeader(header), Verifier: func(tr *http.Transport) error {
			if _, ok := tr.ProxyConnectHeader[http.CanonicalHeaderKey("test")]; !ok {
				return errors.New("header was not set by OptionProxyConnectHeader")
			}
			return nil
		}},
		{Name: "OptionMaxResponseHeaderBytes", Option: OptionMaxResponseHeaderBytes(1), Verifier: func(tr *http.Transport) error {
			if tr.MaxResponseHeaderBytes != 1 {
				return errors.New("limit was not set by OptionMaxResponseHeaderBytes")
			}
			return nil
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(tb *testing.T) {
			var result = New(testCase.Option)
			var e = testCase.Verifier(result)
			if e != nil {
				tb.Fatal(e.Error())
			}
		})
	}
}
