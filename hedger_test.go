package transport

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func TestHedgerSuccess(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var backoffTime = time.Hour
	var decorator = NewHedger(NewFixedBackoffPolicy(backoffTime))
	var client = &http.Client{
		Transport: decorator(wrapped),
	}
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	req = req.WithContext(context.Background())
	wrapped.EXPECT().RoundTrip(gomock.Any()).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		},
		nil,
	)
	var done = make(chan interface{})
	var errChan = make(chan string)
	go func() {
		var resp, err = client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			errChan <- fmt.Sprintf("Got status code %d and err %v, expected status code %d and err %v",
				resp.StatusCode, err, http.StatusOK, nil)
		}
		close(done)
	}()

	select {
	case e := <-errChan:
		t.Fatal(e)
	case <-done:
		break
	case <-time.After(100 * time.Millisecond):
		t.Fatal("roundtrip took too long to exit")
	}
}

func TestHedgerMultipleCalls(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var backoffTime = time.Millisecond
	var decorator = NewHedger(NewFixedBackoffPolicy(backoffTime))
	var client = &http.Client{
		Transport: decorator(wrapped),
	}
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	req = req.WithContext(context.Background())

	wrapped.EXPECT().RoundTrip(gomock.Any()).Do(
		func(...interface{}) {
			time.Sleep(time.Hour)
		}).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		},
		nil,
	).Times(2)
	wrapped.EXPECT().RoundTrip(gomock.Any()).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		},
		nil,
	)
	var done = make(chan interface{})
	var errChan = make(chan string)
	go func() {
		var resp, err = client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			errChan <- fmt.Sprintf("Got status code %d and err %v, expected status code %d and err %v",
				resp.StatusCode, err, http.StatusOK, nil)
			return
		}
		close(done)
	}()

	select {
	case e := <-errChan:
		t.Fatal(e)
	case <-done:
		break
	case <-time.After(100 * time.Millisecond):
		t.Fatal("roundtrip took too long to exit")
	}
}

func TestHedgerContextTimeout(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var backoffTime = time.Hour
	var decorator = NewHedger(NewFixedBackoffPolicy(backoffTime))
	var client = &http.Client{
		Transport: decorator(wrapped),
	}
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))

	var timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Nanosecond)
	time.Sleep(time.Nanosecond)
	defer cancel()
	req = req.WithContext(timeoutCtx)

	wrapped.EXPECT().RoundTrip(gomock.Any()).Do(func(...interface{}) { time.Sleep(time.Hour) }).Return(nil, nil).AnyTimes()
	var done = make(chan interface{})
	var errChan = make(chan string)
	go func() {
		var resp, err = client.Do(req)
		if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			errChan <- fmt.Sprintf("Expected err %v, but got %v", context.DeadlineExceeded, err)
			return
		}
		if resp != nil {
			errChan <- fmt.Sprintf("Expected resp nil, but got %v", resp)
			return
		}
		close(done)
	}()

	select {
	case e := <-errChan:
		t.Fatal(e)
	case <-done:
		break
	case <-time.After(100 * time.Millisecond):
		t.Fatal("roundtrip took too long to exit")
	}
}

func TestHedgerResponseContextNotCanceled(t *testing.T) {
	// This test is intentionally not using mocks because the condition we're
	// testing only shows up when the std lib http.Transport is being used. It
	// wraps the Response.Body in a context aware reader that starts emitting
	// errors as soon as the context used to make the request is canceled. The
	// net result is that it is possible to get a success response from the
	// hedger that then has an unreadable response body.
	t.Parallel()

	tsrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		_, _ = w.Write([]byte(`response data`))
	}))
	defer tsrv.Close()

	client := tsrv.Client()
	var backoffTime = 5 * time.Millisecond
	var decorator = NewHedger(NewFixedBackoffPolicy(backoffTime))
	client.Transport = decorator(client.Transport)
	var req, _ = http.NewRequest("GET", tsrv.URL, http.NoBody)

	var done = make(chan interface{})
	var errChan = make(chan string)
	go func() {
		var resp, err = client.Do(req)
		if err != nil {
			errChan <- fmt.Sprintf("Got err %v but expected no error", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errChan <- fmt.Sprintf("Got status code %d but expected status code %d",
				resp.StatusCode, http.StatusOK)
			return
		}
		if resp.Request.Context().Err() != nil {
			errChan <- "the request context is canceled too soon. this prevents reading the response body"
		}
		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			errChan <- fmt.Sprintf("could not read the response body: %s", err)
		}
		close(done)
	}()

	select {
	case e := <-errChan:
		t.Fatal(e)
	case <-done:
		break
	case <-time.After(100 * time.Millisecond):
		t.Fatal("roundtrip took too long to exit")
	}
}

func TestHedgerConcurrentHeaderModifications(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var backoffTime = time.Millisecond
	var decorator = NewHedger(NewFixedBackoffPolicy(backoffTime))
	var client = &http.Client{
		Transport: decorator(wrapped),
	}
	var req, _ = http.NewRequest("GET", "/", ioutil.NopCloser(bytes.NewReader([]byte(``))))
	req = req.WithContext(context.Background())

	wrapped.EXPECT().RoundTrip(gomock.Any()).Do(
		func(r *http.Request) {
			for x := 0; x < 100; x = x + 1 {
				r.Header.Set("key", "value")
			}
			time.Sleep(time.Hour)
		}).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		},
		nil,
	).Times(5)
	wrapped.EXPECT().RoundTrip(gomock.Any()).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		},
		nil,
	)
	var done = make(chan interface{})
	var errChan = make(chan string)
	go func() {
		var resp, err = client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			errChan <- fmt.Sprintf("Got status code %d and err %v, expected status code %d and err %v",
				resp.StatusCode, err, http.StatusOK, nil)
			return
		}
		close(done)
	}()

	select {
	case e := <-errChan:
		t.Fatal(e)
	case <-done:
		break
	case <-time.After(100 * time.Millisecond):
		t.Fatal("roundtrip took too long to exit")
	}
}
