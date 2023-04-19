package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRequestCopier(t *testing.T) {
	var bodyContent = "TEST"
	var e error
	var request, _ = http.NewRequest(http.MethodGet, "/", io.NopCloser(bytes.NewBufferString(bodyContent)))
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

func newRoundTripWithLatencyFunc(resp *http.Response, latency time.Duration) func(r *http.Request) (*http.Response, error) {
	return func(r *http.Request) (*http.Response, error) {
		resp.Request = r
		select {
		case <-time.After(latency):
			return resp, nil
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
}

func TestRetryTimeIfNoLatency(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Minute),
	)(wrapped)

	var roundTripFunc = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, 0)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFunc).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var resp, e = rt.RoundTrip(req)
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 but got %d", resp.StatusCode)
	}
}

func TestRetryTimeIfLatency(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Millisecond),
	)(wrapped)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	var roundTripWithLatencyFunc = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, time.Minute)
	var roundTripDeadlineExceededFunc = func(r *http.Request) (*http.Response, error) {
		cancel()
		return nil, context.DeadlineExceeded
	}
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripWithLatencyFunc).Times(2)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripDeadlineExceededFunc).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req.WithContext(ctx))
	if e == nil {
		t.Fatal("expected an error but got nil")
	}
}

func TestContextKeepAlive(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Minute),
	)(wrapped)

	var roundTripFunc = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, 0)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFunc).Times(1)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var res, e = rt.RoundTrip(req.WithContext(ctx))
	if e != nil {
		t.Fatalf("expected a nil error but got %s", e.Error())
	}
	if res.Request.Context().Err() != nil {
		t.Fatal("context should not have been canceled but was")
	}
}

func TestContextKeepAliveWithRetries(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewTimeoutRetryPolicy(time.Millisecond),
	)(wrapped)

	var roundTripFuncWithLatency = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, time.Minute)
	var roundTripFuncWithoutLatency = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, 0)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFuncWithLatency).Times(1)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFuncWithoutLatency).Times(1)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var res, e = rt.RoundTrip(req.WithContext(ctx))
	if e != nil {
		t.Fatalf("expected a nil error but got %s", e.Error())
	}
	if res.Request.Context().Err() != nil {
		t.Fatal("context should not have been canceled but was")
	}
}

func TestRetryCode(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewStatusCodeRetryPolicy(http.StatusInternalServerError),
	)(wrapped)

	var roundTripFuncWithError = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       http.NoBody,
	}, 0)
	var roundTripFunc = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, time.Second)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFuncWithError).Times(4)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFunc).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req)
	if e != nil {
		t.Fatalf("expected a success but got: %s", e.Error())
	}
}

func TestRetryLimit(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetrier(
		NewFixedBackoffPolicy(0),
		NewLimitedRetryPolicy(3, NewStatusCodeRetryPolicy(http.StatusInternalServerError)),
	)(wrapped)

	var roundTripFuncWithError = newRoundTripWithLatencyFunc(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       http.NoBody,
	}, 0)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(roundTripFuncWithError).Times(4)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req)
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

func TestNewExponentialBackofferPolicy(t *testing.T) {
	exponentialBackoffPolicy := NewExponentialBackoffPolicy(time.Second)
	backoffer1 := exponentialBackoffPolicy()
	backoffer2 := exponentialBackoffPolicy()

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var res = &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       http.NoBody,
	}

	backoffer1DurationRound1 := backoffer1.Backoff(req, res, nil)
	backoffer1DurationRound2 := backoffer1.Backoff(req, res, nil)

	backoffer2DurationRound1 := backoffer2.Backoff(req, res, nil)

	assert.Equal(t, backoffer1DurationRound2, backoffer1DurationRound1*2)
	assert.Equal(t, backoffer1DurationRound1, backoffer2DurationRound1)

}
