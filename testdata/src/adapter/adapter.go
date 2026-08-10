// Package adapter is the shape every dependency-injected design bottoms out
// in: the one place that touches the real world, behind a seam that already
// exists. The boundary is silent; everything outside it is not — and every
// exemption here wraps a LISTED impure call, so deleting the exemption breaks
// this fixture rather than passing vacuously.
package adapter

import (
	"net/http"
	"os/exec"
	"time"
)

// runner runs a named command. It is the seam.
type runner func(string) ([]byte, error)

// runCommand is the seam, defaulting to the real implementation and replaced
// by tests. Naming execRun here — as a value, not a call — is what makes
// execRun the boundary.
var runCommand runner = execRun

// execRun is the real implementation behind the seam: the subprocess spawn
// lives here and nowhere else.
func execRun(name string) ([]byte, error) { return exec.Command(name).Output() }

// Run runs through the seam.
func Run(name string) ([]byte, error) { return runCommand(name) }

// Fetcher is the injected abstraction for the network.
type Fetcher interface {
	Fetch(url string) (*http.Response, error)
	Head(url string) (*http.Response, error)
}

// Web is the Fetcher backed by the real network.
type Web struct{}

// Fetch is the interface's method, implemented against the real network.
func (Web) Fetch(url string) (*http.Response, error) { return http.Get(url) }

// Head is the interface's other method, and is silent for the same reason.
func (Web) Head(url string) (*http.Response, error) { return http.Head(url) }

// Probe is a method on the same type that the interface does NOT name, so it
// is ordinary code and the rule still applies to it.
func (Web) Probe(url string) (*http.Response, error) {
	return http.Get(url) // want `net/http.Get is called directly`
}

// Anything is an interface with no methods. Every type satisfies it, and it
// exempts nothing: an exemption is granted per interface method, and this one
// names none.
type Anything interface{}

// Loose satisfies Anything and nothing else.
type Loose struct{}

// Spawn is not behind any abstraction, whatever Anything accepts.
func (Loose) Spawn(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// Purge is called by name elsewhere in the package. A method NAMED as the
// callee of a call is not a method handed around as a value, so it is not a
// boundary and the rule still applies to it.
func (Loose) Purge(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// Clean calls Purge, which is a call and not a reference.
func Clean(l Loose, name string) error { return l.Purge(name) }

// Buffered is the Fetcher implemented with a pointer receiver: it satisfies
// the interface only through *Buffered, which is the shape a stateful adapter
// takes and must be recognised the same way.
type Buffered struct{ last time.Time }

// Fetch reads the real network behind the interface, stamping when.
func (b *Buffered) Fetch(url string) (*http.Response, error) {
	b.last = time.Now()
	return http.Get(url)
}

// Head is the pointer adapter's other interface method.
func (b *Buffered) Head(url string) (*http.Response, error) { return http.Head(url) }

// helper is never handed around as a value, so it is not a boundary.
func helper(name string) ([]byte, error) {
	return exec.Command(name).Output() // want `os/exec.Command is called directly`
}

// Open calls the helper, which is a call and not a reference.
func Open(name string) ([]byte, error) { return helper(name) }
