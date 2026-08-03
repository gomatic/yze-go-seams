// Package adapter is the shape every dependency-injected design bottoms out
// in: the one place that touches the real world, behind a seam that already
// exists. The boundary is silent; everything outside it is not.
package adapter

import (
	"io"
	"os"
	"time"
)

// dirReader lists a directory. It is the seam.
type dirReader func(string) ([]os.DirEntry, error)

// readDir is the seam, defaulting to the real implementation and replaced by
// tests. Naming osReadDir here — as a value, not a call — is what makes
// osReadDir the boundary.
var readDir dirReader = osReadDir

// osReadDir is the real implementation behind the seam.
func osReadDir(at string) ([]os.DirEntry, error) { return os.ReadDir(at) }

// List reads through the seam.
func List(at string) ([]os.DirEntry, error) { return readDir(at) }

// Clock is the injected abstraction for the clock.
type Clock interface {
	Now() time.Time
	Stamp() string
}

// System is the Clock backed by the real clock.
type System struct{}

// Now is the interface's method, implemented against the real clock.
func (System) Now() time.Time { return time.Now().UTC() }

// Stamp is the interface's other method, and is silent for the same reason.
func (System) Stamp() string { return time.Now().Format(time.RFC3339) }

// Snapshot is a method on the same type that the interface does NOT name, so
// it is ordinary code and the rule still applies to it.
func (System) Snapshot() time.Time {
	return time.Now() // want `time.Now is called directly`
}

// Anything is an interface with no methods. Every type satisfies it, and it
// exempts nothing: an exemption is granted per interface method, and this one
// names none.
type Anything interface{}

// Loose satisfies Anything and nothing else.
type Loose struct{}

// Read is not behind any abstraction, whatever Anything accepts.
func (Loose) Read(at string) ([]byte, error) {
	return os.ReadFile(at) // want `os.ReadFile is called directly`
}

// Purge is called by name elsewhere in the package. A method NAMED as the
// callee of a call is not a method handed around as a value, so it is not a
// boundary and the rule still applies to it.
func (Loose) Purge(at string) error {
	return os.Remove(at) // want `os.Remove is called directly`
}

// Clean calls Purge, which is a call and not a reference.
func Clean(l Loose, at string) error { return l.Purge(at) }

// Buffered is the Clock implemented with a pointer receiver: it satisfies the
// interface only through *Buffered, which is the shape a stateful adapter
// takes and must be recognised the same way.
type Buffered struct{ last time.Time }

// Now reads the real clock behind the interface.
func (b *Buffered) Now() time.Time {
	b.last = time.Now().UTC()
	return b.last
}

// Stamp formats what Now last read.
func (b *Buffered) Stamp() string { return b.last.Format(time.RFC3339) }

// helper is never handed around as a value, so it is not a boundary.
func helper(at string) (io.ReadCloser, error) {
	return os.Open(at) // want `os.Open is called directly`
}

// Open calls the helper, which is a call and not a reference.
func Open(at string) (io.ReadCloser, error) { return helper(at) }
