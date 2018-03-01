package transport

import (
	"errors"
	"net/http"
	"testing"
)

func TestChainAppliesReverseOrder(t *testing.T) {
	var annotations []string
	var annotator = func(annotation string) func(wrapped http.RoundTripper) http.RoundTripper {
		return func(wrapped http.RoundTripper) http.RoundTripper {
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				annotations = append(annotations, annotation)
				return wrapped.RoundTrip(r)
			})
		}
	}
	var base = RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("")
	})

	var chain = Chain{
		annotator("one"),
		annotator("two"),
		annotator("three"),
	}
	var result = chain.Apply(base)
	result.RoundTrip(nil)
	if len(annotations) != 3 {
		t.Fatal("did not apply decorators")
	}
	if annotations[0] != "one" {
		t.Fatalf("decorators applied out of order: %v", annotations)
	}
	if annotations[1] != "two" {
		t.Fatalf("decorators applied out of order: %v", annotations)
	}
	if annotations[2] != "three" {
		t.Fatalf("decorators applied out of order: %v", annotations)
	}
}

func TestChainAppliesFactoryReverseOrder(t *testing.T) {
	var annotations []string
	var annotator = func(annotation string) func(wrapped http.RoundTripper) http.RoundTripper {
		return func(wrapped http.RoundTripper) http.RoundTripper {
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				annotations = append(annotations, annotation)
				return wrapped.RoundTrip(r)
			})
		}
	}
	var base = func() http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("")
		})
	}

	var chain = Chain{
		annotator("one"),
		annotator("two"),
		annotator("three"),
	}
	var result = chain.ApplyFactory(base)
	result().RoundTrip(nil)
	if len(annotations) != 3 {
		t.Fatal("did not apply decorators")
	}
	if annotations[0] != "one" {
		t.Fatalf("decorators applied out of order: %v", annotations)
	}
	if annotations[1] != "two" {
		t.Fatalf("decorators applied out of order: %v", annotations)
	}
	if annotations[2] != "three" {
		t.Fatalf("decorators applied out of order: %v", annotations)
	}
}
