// Package global calls methods on the stdlib's own process-wide state, which
// no test can replace.
package global

import "net/http"

// Fetch sends through the package-global client — the request goes out for
// real, where an injected *http.Client would not.
func Fetch(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req) // want `net/http.DefaultClient is called directly`
}

// Route registers on the package-global mux. That is process-wide state too,
// but it is not on the rule's list: routing is not a resource whose failure
// branch a test cannot reach, and the rule claims only what it can defend.
func Route(pattern string, handler http.Handler) {
	http.DefaultServeMux.Handle(pattern, handler)
}

// RoundTrip sends through the package-global transport: the same wire as
// DefaultClient, one layer down.
func RoundTrip(req *http.Request) (*http.Response, error) {
	return http.DefaultTransport.RoundTrip(req) // want `net/http.DefaultTransport is called directly`
}
