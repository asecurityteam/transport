package transport

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestRequestCopier(t *testing.T) {
	var bodyContent = "TEST"
	var e error
	var request, _ = http.NewRequest(http.MethodGet, "/", ioutil.NopCloser(bytes.NewBufferString(bodyContent)))
	var copier *requestCopier
	copier, e = newRequestCopier(request)
	if e != nil {
		t.Fatal(e.Error())
	}

	var r1 = copier.Copy()
	var buf1 bytes.Buffer
	var n int64
	n, e = io.Copy(&buf1, r1.Body)
	if e != nil {
		t.Fatal(e.Error())
	}
	if n != int64(len([]byte(bodyContent))) {
		t.Fatalf("expected to copy %d bytes but only found %d.", len([]byte(bodyContent)), n)
	}

	var r2 = copier.Copy()
	var buf2 bytes.Buffer
	n, e = io.Copy(&buf2, r2.Body)
	if e != nil {
		t.Fatal(e.Error())
	}
	if n != int64(len([]byte(bodyContent))) {
		t.Fatalf("expected to copy %d bytes but only found %d.", len([]byte(bodyContent)), n)
	}

	var regenerated, _ = r2.GetBody()
	n, e = io.Copy(&buf2, regenerated)
	if e != nil {
		t.Fatal(e.Error())
	}
	if n != int64(len([]byte(bodyContent))) {
		t.Fatalf("expected to copy %d bytes but only found %d.", len([]byte(bodyContent)), n)
	}
}

type retryRoundTipperFixtureResponse struct {
	Error    error
	Response *http.Response
	Sleep    time.Duration
}
type retryRoundTipperFixture struct {
	Calls     int
	responses []retryRoundTipperFixtureResponse
	lock      *sync.Mutex
}

func (rt *retryRoundTipperFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.lock.Lock()
	var result, e, latency = rt.responses[rt.Calls].Response, rt.responses[rt.Calls].Error, rt.responses[rt.Calls].Sleep
	result.Request = r
	rt.Calls = rt.Calls + 1
	rt.lock.Unlock()
	select {
	case <-time.After(latency):
		return result, e
	case <-r.Context().Done():
		return nil, r.Context().Err()
	}
}

func newRetryRoundTripperFixture(responses ...retryRoundTipperFixtureResponse) *retryRoundTipperFixture {
	return &retryRoundTipperFixture{
		Calls:     0,
		lock:      &sync.Mutex{},
		responses: responses,
	}
}

func TestRetryTimeIfNoLatency(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Millisecond),
	)(wrapped)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var resp, e = rt.RoundTrip(req)
	if wrapped.Calls != 1 {
		t.Fatal("retry engaged before timeout condition")
	}
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 but got %d", resp.StatusCode)
	}
}

func TestRetryTimeIfLatency(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Millisecond),
	)(wrapped)

	var ctx, cancel = context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req.WithContext(ctx))
	if wrapped.Calls != 3 {
		t.Fatalf("retried the wrong number of times: %d", wrapped.Calls)
	}
	if e == nil {
		t.Fatal("expected an error but got nil")
	}
}

func TestContextKeepAlive(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Millisecond),
	)(wrapped)

	var ctx, cancel = context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var res, e = rt.RoundTrip(req.WithContext(ctx))
	if wrapped.Calls != 1 {
		t.Fatalf("retried the wrong number of times: %d", wrapped.Calls)
	}
	if e != nil {
		t.Fatalf("expected a nil error but got %s", e.Error())
	}
	if res.Request.Context().Err() != nil {
		t.Fatal("context should not have been cancelled but was")
	}
}

func TestContextKeepAliveWithRetries(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Millisecond),
	)(wrapped)

	var ctx, cancel = context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var res, e = rt.RoundTrip(req.WithContext(ctx))
	if wrapped.Calls != 2 {
		t.Fatalf("retried the wrong number of times: %d", wrapped.Calls)
	}
	if e != nil {
		t.Fatalf("expected a nil error but got %s", e.Error())
	}
	if res.Request.Context().Err() != nil {
		t.Fatal("context should not have been cancelled but was")
	}
}

func TestRetryCode(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewStatusCodeRetryPolicy(http.StatusInternalServerError),
	)(wrapped)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req)
	if wrapped.Calls != 5 {
		t.Fatalf("retried the wrong number of times: %d", wrapped.Calls)
	}
	if e != nil {
		t.Fatalf("expected a success but got: %s", e.Error())
	}
}

func TestRetryLimit(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: time.Second},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewLimitedRetryPolicy(3, NewStatusCodeRetryPolicy(http.StatusInternalServerError)),
	)(wrapped)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req)
	if wrapped.Calls != 4 {
		t.Fatalf("retried the wrong number of times: %d", wrapped.Calls)
	}
	if e != nil {
		t.Fatalf("expected a response with bad code but got: %s", e.Error())
	}
}

func TestJitterCalculation(t *testing.T) {
	var jitterValue = float64(1)
	var original = time.Millisecond
	var percentage = .1
	var randomCalled = false
	var random = func() float64 {
		if randomCalled {
			return 0
		}
		randomCalled = true
		return jitterValue
	}
	var result = calculateJitteredBackoff(original, percentage, random)
	if result != (time.Millisecond + 100*time.Microsecond) {
		t.Fatal(result)
	}
	randomCalled = false
	random = func() float64 {
		if randomCalled {
			return 1
		}
		randomCalled = true
		return jitterValue
	}
	result = calculateJitteredBackoff(original, percentage, random)
	if result != (time.Millisecond - 100*time.Microsecond) {
		t.Fatal(result)
	}
}
