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
		"test": nil,
	}
	var testCases = []optionTestCase{
		{Name: "OptionProxy", Option: OptionProxy(proxyFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.Proxy(nil)
			if e != testErr {
				return errors.New("OptionProxy did not set proxy function")
			}
			return nil
		}},
		{Name: "OptionDialContext", Option: OptionDialContext(dialCtxFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.DialContext(nil, "", "")
			if e != testErr {
				return errors.New("OptionDialContext did not set dial function")
			}
			return nil
		}},
		{Name: "OptionDial", Option: OptionDial(dialFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.Dial("", "")
			if e != testErr {
				return errors.New("OptionDial did not set dial function")
			}
			return nil
		}},
		{Name: "OptionDialTLS", Option: OptionDialTLS(dialTLSFunc), Verifier: func(tr *http.Transport) error {
			var _, e = tr.DialTLS("", "")
			if e != testErr {
				return errors.New("OptionDialTLS did not set dial function")
			}
			return nil
		}},
		{Name: "OptionTLSClientConfig", Option: OptionTLSClientConfig(tlsConfig), Verifier: func(tr *http.Transport) error {
			if tr.TLSClientConfig != tlsConfig {
				return errors.New("OptionTLSClientConfig did not set tls config")
			}
			return nil
		}},
		{Name: "OptionTLSHandshakeTimeout", Option: OptionTLSHandshakeTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.TLSHandshakeTimeout != 1 {
				return errors.New("OptionTLSHandshakeTimeout did not set timeout")
			}
			return nil
		}},
		{Name: "OptionDisableKeepAlives", Option: OptionDisableKeepAlives(true), Verifier: func(tr *http.Transport) error {
			if tr.DisableKeepAlives != true {
				return errors.New("OptionDisableKeepAlives did not set keep alive")
			}
			return nil
		}},
		{Name: "OptionDisableCompression", Option: OptionDisableCompression(true), Verifier: func(tr *http.Transport) error {
			if tr.DisableCompression != true {
				return errors.New("OptionDisableCompression did not set disable compression")
			}
			return nil
		}},
		{Name: "OptionMaxIdleConns", Option: OptionMaxIdleConns(1), Verifier: func(tr *http.Transport) error {
			if tr.MaxIdleConns != 1 {
				return errors.New("OptionMaxIdleConns did not set idle conns")
			}
			return nil
		}},
		{Name: "OptionMaxIdleConnsPerHost", Option: OptionMaxIdleConnsPerHost(1), Verifier: func(tr *http.Transport) error {
			if tr.MaxIdleConnsPerHost != 1 {
				return errors.New("OptionMaxIdleConnsPerHost did not set idle conns")
			}
			return nil
		}},
		{Name: "OptionIdleConnTimeout", Option: OptionIdleConnTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.IdleConnTimeout != 1 {
				return errors.New("OptionIdleConnTimeout did not set timeout")
			}
			return nil
		}},
		{Name: "OptionResponseHeaderTimeout", Option: OptionResponseHeaderTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.ResponseHeaderTimeout != 1 {
				return errors.New("OptionResponseHeaderTimeout did not set timeout")
			}
			return nil
		}},
		{Name: "OptionExpectContinueTimeout", Option: OptionExpectContinueTimeout(1), Verifier: func(tr *http.Transport) error {
			if tr.ExpectContinueTimeout != 1 {
				return errors.New("OptionExpectContinueTimeout did not set timeout")
			}
			return nil
		}},
		{Name: "OptionTLSNextProto", Option: OptionTLSNextProto(nextProto), Verifier: func(tr *http.Transport) error {
			if _, ok := tr.TLSNextProto["test"]; !ok {
				return errors.New("OptionTLSNextProto did not set next proto")
			}
			return nil
		}},
		{Name: "OptionProxyConnectHeader", Option: OptionProxyConnectHeader(header), Verifier: func(tr *http.Transport) error {
			if _, ok := tr.ProxyConnectHeader["test"]; !ok {
				return errors.New("OptionProxyConnectHeader did not set header")
			}
			return nil
		}},
		{Name: "OptionMaxResponseHeaderBytes", Option: OptionMaxResponseHeaderBytes(1), Verifier: func(tr *http.Transport) error {
			if tr.MaxResponseHeaderBytes != 1 {
				return errors.New("OptionMaxResponseHeaderBytes did not set limit")
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
