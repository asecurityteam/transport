package transport

import (
	"context"
	"net/http"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
)

func TestRetryAfterNot429(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	rtFunc := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}, nil
	}

	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var resp, e = rt.RoundTrip(req)
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 but got %d", resp.StatusCode)
	}
}

func TestRetryAfter429NoRetryAfter(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	var rtFunc429 = func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       http.NoBody,
		}, nil
	}
	var rtFunc200 = func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		}, nil
	}
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc429).Times(2)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc200).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	start := time.Now()
	var resp, e = rt.RoundTrip(req)
	duration := time.Since(start)
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 but got %d", resp.StatusCode)
	}

	durationInSeconds := int64(duration / time.Second)
	if durationInSeconds < 3 {
		t.Fatalf("expected execution time to take at least 1 + 2 seconds due to use of exponential backoff with initial value of 1 and two replies with 429, but got %d", durationInSeconds)
	}
}

func TestRetryAfter429ComboNoRetryAfterAndWithRetryAfter(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	var rtFunc429NoRetryAfter = func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       http.NoBody,
		}, nil
	}
	var rtFunc429WithRetryAfter = func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       http.NoBody,
			Header: map[string][]string{
				"Retry-After": []string{"1"},
			},
		}, nil
	}
	var rtFunc200 = func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		}, nil
	}
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc429NoRetryAfter).Times(1)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc429WithRetryAfter).Times(1)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc200).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	start := time.Now()
	var resp, e = rt.RoundTrip(req)
	duration := time.Since(start)
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 but got %d", resp.StatusCode)
	}

	durationInSeconds := int64(duration / time.Second)
	if durationInSeconds < 2 {
		t.Fatalf("expected execution time to take at least 1 + 1 seconds due to use of exponential backoff with initial value of 1 and two replies with 429 where the second reply has Retry-After=1 header, but got %d", durationInSeconds)
	}
}

func TestRetryAfter429BadRetryAfter(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	rtFunc := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       http.NoBody,
			Header: map[string][]string{
				"Retry-After": []string{"bogus header value"},
			},
		}, nil
	}

	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var resp, e = rt.RoundTrip(req)
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != 429 {
		t.Fatalf("expected 429 but got %d", resp.StatusCode)
	}
}

func TestRetryAfter429WithRetryAfter(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	rtFunc1 := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       http.NoBody,
			Header: map[string][]string{
				"Retry-After": []string{"1"},
			},
		}, nil
	}

	rtFunc2 := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}, nil
	}

	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc1).Times(1)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc2).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var resp, e = rt.RoundTrip(req)
	if e != nil {
		t.Fatal(e.Error())
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 but got %d", resp.StatusCode)
	}
}

func TestRetryAfter429WithDeadlineExceeded(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	rtFunc1 := func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 429,
			Body:       http.NoBody,
			Header: map[string][]string{
				"Retry-After": []string{"1"},
			},
		}, nil
	}

	rtFunc2 := func(r *http.Request) (*http.Response, error) {
		cancel()
		return nil, context.DeadlineExceeded
	}

	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc1).Times(1)
	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc2).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req.WithContext(ctx))
	if e == nil {
		t.Fatal("expected an error but got nil")
	}
}

func TestRetryContextCanceled(t *testing.T) {
	t.Parallel()

	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var wrapped = NewMockRoundTripper(ctrl)
	var rt = NewRetryAfter()(wrapped)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	rtFunc := func(r *http.Request) (*http.Response, error) {
		cancel()
		return nil, context.DeadlineExceeded
	}

	wrapped.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(rtFunc).Times(1)

	var req, _ = http.NewRequest(http.MethodGet, "/", nil)
	var _, e = rt.RoundTrip(req.WithContext(ctx))
	if e == nil {
		t.Fatal("expected an error but got nil")
	}
}
