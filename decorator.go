package transport

import "net/http"

// Decorator is a named type for any function that takes a RoundTripper and
// returns a RoundTripper.
type Decorator func(http.RoundTripper) http.RoundTripper

// Chain is an ordered collection of Decorators.
type Chain []Decorator

// Apply wraps the given RoundTripper with the Decorator chain.
func (c Chain) Apply(base http.RoundTripper) http.RoundTripper {
	for x := len(c) - 1; x >= 0; x = x - 1 {
		base = c[x](base)
	}
	return base
}

// ApplyFactory wraps the given Factory such that all new instances produced
// will be decorated with the contents of the chain.
func (c Chain) ApplyFactory(base Factory) Factory {
	return func() http.RoundTripper {
		return c.Apply(base())
	}
}
