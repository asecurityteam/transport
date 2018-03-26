package transport

import (
	"net/http"
	"sync"
)

// Rotator contains multiple instances of a RoundTripper and rotates through
// them per-request. This is useful when dealing with HTTP/2 in situations where
// more than one TCP connection per host is required.
type Rotator struct {
	numberOfInstances int
	currentOffset     int
	instances         []http.RoundTripper
	factory           Factory
	lock              *sync.Mutex
}

// RotatorOption is a configuration for the Rotator decorator
type RotatorOption func(*Rotator) *Rotator

// RotatorOptionInstances configurs a rotator with the number of internal
// RoundTripper instances it should maintain for the rotation.
func RotatorOptionInstances(number int) RotatorOption {
	return func(r *Rotator) *Rotator {
		r.numberOfInstances = number
		return r
	}
}

// NewRotator uses the given factory as a source and generates a number of
// instances based on the options given. The instances are called in a naive,
// round-robin manner.
func NewRotator(factory Factory, opts ...RotatorOption) *Rotator {
	var r = &Rotator{factory: factory, lock: &sync.Mutex{}}
	for _, opt := range opts {
		r = opt(r)
	}
	for x := 0; x < r.numberOfInstances; x = x + 1 {
		r.instances = append(r.instances, r.factory())
	}
	// Defensively maintain at least one in the set at all times.
	if len(r.instances) < 1 {
		r.instances = append(r.instances, r.factory())
		r.numberOfInstances = 1
	}
	return r
}

// NewRotatorFactory is a counterpart for NewRotator that generates a Factory
// function for use with other decorators.
func NewRotatorFactory(factory Factory, opts ...RotatorOption) Factory {
	return func() http.RoundTripper {
		return NewRotator(factory, opts...)
	}
}

// RoundTrip round-robins the outgoing requests against all of the internal
// instances.
func (c *Rotator) RoundTrip(r *http.Request) (*http.Response, error) {
	c.lock.Lock()
	c.currentOffset = (c.currentOffset + 1) % c.numberOfInstances
	var offset = c.currentOffset
	c.lock.Unlock()
	return c.instances[offset].RoundTrip(r)
}
