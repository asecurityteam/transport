package transport

import (
	"errors"
	"net/http"
	"testing"
)

type roundTripperForRotatorTests struct {
	v string // Need a value here to make instances unique values
}

func (roundTripperForRotatorTests) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("")
}

func TestRotatorOptionInstances(t *testing.T) {
	var factory = func() http.RoundTripper {
		return &roundTripperForRotatorTests{"string"}
	}
	var r = NewRotator(factory)
	if len(r.instances) != 1 {
		t.Fatal("did not default to one instance")
	}
	r = NewRotator(factory, RotatorOptionInstances(2))
	if len(r.instances) != 2 {
		t.Fatal("did not create the right number of instances")
	}
	_, _ = r.RoundTrip(nil)
	if r.currentOffset != 1 {
		t.Fatal("did not rotate through instances after using")
	}
	_, _ = r.RoundTrip(nil)
	if r.currentOffset != 0 {
		t.Fatal("did not rotate back through the beginning")
	}
}
