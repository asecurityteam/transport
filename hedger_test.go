package transport

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
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
