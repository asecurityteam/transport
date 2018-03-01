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
	rt.Calls = rt.Calls + 1
	rt.lock.Unlock()
	time.Sleep(latency)
	return result, e
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
		RetrierOptionLimit(3),
		RetrierOptionDelay(time.Millisecond),
		RetrierOptionDelayJitter(time.Millisecond),
		RetrierOptionTimeout(time.Millisecond),
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
		{Error: context.DeadlineExceeded, Response: nil, Sleep: 2 * time.Millisecond},
		{Error: context.DeadlineExceeded, Response: nil, Sleep: 2 * time.Millisecond},
		{Error: context.DeadlineExceeded, Response: nil, Sleep: 2 * time.Millisecond},
		{Error: context.DeadlineExceeded, Response: nil, Sleep: 2 * time.Millisecond},
		{Error: context.DeadlineExceeded, Response: nil, Sleep: 2 * time.Millisecond},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		RetrierOptionLimit(3),
		RetrierOptionDelay(time.Millisecond),
		RetrierOptionDelayJitter(time.Millisecond),
		RetrierOptionTimeout(time.Millisecond),
	)(wrapped)
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req)
	if wrapped.Calls != 4 {
		t.Fatalf("called %d times instead of 4", wrapped.Calls)
	}
	if e == nil {
		t.Fatal("did not get an error response from the RoundTripper")
	}
}

func TestRetryTimeRespectsParentContextDeadline(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 2 * time.Millisecond},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 2 * time.Millisecond},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		RetrierOptionLimit(3),
		RetrierOptionDelay(time.Millisecond),
		RetrierOptionDelayJitter(time.Millisecond),
		RetrierOptionTimeout(time.Millisecond),
	)(wrapped)
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var ctx, cancel = context.WithTimeout(req.Context(), time.Millisecond)
	defer cancel()
	var _, e = rt.RoundTrip(req.WithContext(ctx))
	if wrapped.Calls != 1 {
		t.Fatalf("called %d times instead of 1", wrapped.Calls)
	}
	if e != context.DeadlineExceeded {
		t.Fatal("did not get an error response from the RoundTripper")
	}
}

func TestRetryCodes(t *testing.T) {
	var responses = []retryRoundTipperFixtureResponse{
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusInternalServerError, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
		{Error: nil, Response: &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewBufferString(``))}, Sleep: 0},
	}
	var wrapped = newRetryRoundTripperFixture(responses...)
	var rt = NewRetrier(
		RetrierOptionLimit(3),
		RetrierOptionDelay(time.Millisecond),
		RetrierOptionDelayJitter(time.Millisecond),
		RetrierOptionResponseCode(http.StatusInternalServerError),
	)(wrapped)
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var resp, e = rt.RoundTrip(req)
	if wrapped.Calls != 3 {
		t.Fatalf("called %d times instead of 3", wrapped.Calls)
	}
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("did not retry until 200 OK: %d", resp.StatusCode)
	}
}
