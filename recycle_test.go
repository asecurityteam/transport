package transport

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

type roundTripperForRecycleTests struct {
	v string // Need a value here to make instances unique values
}

func (roundTripperForRecycleTests) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("")
}

func TestRecycleOptionTTL(t *testing.T) {
	var factory = func() http.RoundTripper {
		return &roundTripperForRecycleTests{"string"}
	}
	var r = NewRecycler(factory)
	if r.ttl != 0 {
		t.Fatal("ttl defaulted to non-zero")
	}
	if r.ttlJitter != 0 {
		t.Fatal("ttlJitter defaulted to non-zero")
	}
	r = NewRecycler(factory, RecycleOptionTTL(time.Second), RecycleOptionTTLJitter(10*time.Millisecond))
	if r.ttl != time.Second {
		t.Fatal("ttl did not set correctly")
	}
	if r.ttlJitter != 10*time.Millisecond {
		t.Fatal("ttlJitter did not set correctly")
	}

	var result = r.getTransport()
	if r.nextTTL.Before(time.Now().Add(time.Second-11*time.Millisecond)) || r.nextTTL.After(time.Now().Add(time.Second+11*time.Millisecond)) {
		t.Fatalf("ttl was not generated with the correct jitter")
	}
	if r.getTransport() != result {
		t.Fatal("regenerated transport before ttl")
	}
	time.Sleep(r.nextTTL.Sub(time.Now()) + 5*time.Millisecond)
	if r.getTransport() == result {
		t.Fatal("did not regenerated transport after ttl")
	}
}

func TestRecycleOptionMaxusage(t *testing.T) {
	var factory = func() http.RoundTripper {
		return &roundTripperForRecycleTests{"string2"}
	}
	var r = NewRecycler(factory)
	if r.maxUsage != 0 {
		t.Fatal("maxUsage defaulted to non-zero")
	}
	r = NewRecycler(factory, RecycleOptionMaxUsage(2))
	if r.maxUsage != 2 {
		t.Fatal("maxUsage did not set correctly")
	}

	var result = r.getTransport()
	if r.currentUsage != 1 {
		t.Fatal("did not track transport usage")
	}
	if r.getTransport() != result {
		t.Fatal("regenerated transport too soon")
	}
	if r.currentUsage != 2 {
		t.Fatal("did not track transport usage")
	}
	if r.getTransport() == result {
		t.Fatal("did not regenerated transport after max usage")
	}
	if r.currentUsage != 0 {
		t.Fatal("did not reset transport usage")
	}
}

func TestRecycleOptionChannel(t *testing.T) {
	var factory = func() http.RoundTripper {
		return &roundTripperForRecycleTests{"string3"}
	}
	var r = NewRecycler(factory)
	if len(r.signals) != 0 {
		t.Fatal("channel count was not zero")
	}
	var signal = make(chan struct{}, 1)
	r = NewRecycler(factory, RecycleOptionChannel(signal))
	if len(r.signals) != 1 {
		t.Fatal("channel count was not one")
	}
	var result = r.getTransport()
	if r.getTransport() != result {
		t.Fatal("regenerated transport before getting a signal")
	}
	signal <- struct{}{}
	time.Sleep(time.Millisecond) // Wait for the background listener to activate
	if r.getTransport() == result {
		t.Fatal("did not regenerate transport after getting a signal")
	}
}
