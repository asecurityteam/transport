package transport

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Recycler is a decorator that discards and regenerates the transport after
// a given set of triggers.
type Recycler struct {
	wrapped      http.RoundTripper
	ttl          time.Duration
	ttlJitter    time.Duration
	nextTTL      time.Time
	maxUsage     int
	currentUsage int
	signals      []chan struct{}
	signal       chan struct{}
	lock         *sync.Mutex
	factory      Factory
}

// RecycleOption is a configuration for the Recycler decorator
type RecycleOption func(*Recycler) *Recycler

// RecycleOptionTTL configures the recycler to rotate Transports on an interval.
func RecycleOptionTTL(ttl time.Duration) RecycleOption {
	return func(r *Recycler) *Recycler {
		r.ttl = ttl
		return r
	}
}

// RecycleOptionTTLJitter adds a randomised jitter to the TTL that is plus or
// minus the duration value given.
func RecycleOptionTTLJitter(jitter time.Duration) RecycleOption {
	return func(r *Recycler) *Recycler {
		r.ttlJitter = jitter
		return r
	}
}

// RecycleOptionMaxUsage configures the recycler to rotate Transports after a number
// of uses.
func RecycleOptionMaxUsage(max int) RecycleOption {
	return func(r *Recycler) *Recycler {
		r.maxUsage = max
		return r
	}
}

// RecycleOptionChannel configures the recycler to rotate based on input from a
// channel.
func RecycleOptionChannel(signal chan struct{}) RecycleOption {
	return func(r *Recycler) *Recycler {
		r.signals = append(r.signals, signal)
		return r
	}
}

// NewRecycler uses the given factory as a source and recycles the transport
// based on the options given.
func NewRecycler(factory Factory, opts ...RecycleOption) *Recycler {
	var r = &Recycler{wrapped: factory(), lock: &sync.Mutex{}, factory: factory, signal: make(chan struct{})}
	for _, opt := range opts {
		r = opt(r)
	}
	r.listen()
	return r
}

// NewRecyclerFactory is a counterpart for NewRecycler that generates a Factory
// function for use with other decorators.
func NewRecyclerFactory(factory Factory, opts ...RecycleOption) Factory {
	return func() http.RoundTripper {
		return NewRecycler(factory, opts...)
	}
}

func (c *Recycler) resetTransport() http.RoundTripper {
	c.wrapped = c.factory()
	c.currentUsage = 0
	var renderedJitter = time.Duration(rand.Float64() * float64(c.ttlJitter))
	if rand.Float64()*100 > 50 {
		renderedJitter = -renderedJitter
	}
	c.nextTTL = time.Now().Add(c.ttl + renderedJitter)
	return c.wrapped
}

func (c *Recycler) listen() {
	for _, signal := range c.signals {
		go c.listenOne(signal)
	}
}

func (c *Recycler) listenOne(s chan struct{}) {
	for range s {
		c.signal <- struct{}{}
	}
}

func (c *Recycler) getTransport() http.RoundTripper {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.maxUsage > 0 {
		c.currentUsage = c.currentUsage + 1
		if c.currentUsage > c.maxUsage {
			return c.resetTransport()
		}
	}
	if c.ttl > 0 && time.Now().After(c.nextTTL) {
		return c.resetTransport()
	}
	select {
	case <-c.signal:
		return c.resetTransport()
	default:
		break
	}
	return c.wrapped
}

// RoundTrip applies the discard and regenerate policy.
func (c *Recycler) RoundTrip(r *http.Request) (*http.Response, error) {
	var rt = c.getTransport()
	return rt.RoundTrip(r)
}
