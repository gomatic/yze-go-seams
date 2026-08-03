// Package seam writes every conforming form: the package-level seam variable
// the standard blesses, injected collaborators, and the pure siblings of the
// impure entry points. Nothing here is reported.
package seam

import (
	"math/rand/v2"
	"net/http"
	"os"
	"time"
)

// readFile is the seam the standard blesses explicitly — a reference to the
// stdlib function, never a call, so a test can replace it and reach the error
// branch behind it.
var readFile = os.ReadFile

// now is the clock seam.
var now = time.Now

// openExclusive is the same seam written as a closure, because the seam's
// signature is not the stdlib's. A function literal in a package-level var
// initializer is a seam declaration, exactly like the two above.
var openExclusive = func(at string) (*os.File, error) {
	return os.OpenFile(at, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

// ReadConfig reads through the seam, so its qualifier is an identifier rather
// than a package.
func ReadConfig(at string) ([]byte, error) { return readFile(at) }

// Stamp reads the clock through the seam.
func Stamp() time.Time { return now() }

// Elapsed is pure arithmetic over its argument, not a reading of the clock.
func Elapsed(from time.Time) time.Duration { return time.Since(from) }

// Deadline is pure arithmetic over its argument.
func Deadline(from time.Time) time.Time { return from.Add(5 * time.Second) }

// Pick draws from a generator the caller supplied.
func Pick(from *rand.Rand) int { return from.IntN(10) }

// Source builds a generator from a seed the caller chose — reproducible, and
// injectable by construction.
func Source(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed)) }

// Clock is a collaborator whose method is spelled exactly like the stdlib
// entry point it replaces, so the rule cannot be resolving names by spelling.
type Clock struct{}

// Now is the collaborator's reading.
func (Clock) Now() time.Time { return time.Time{} }

// StampInjected reads the clock through the collaborator.
func StampInjected(clock Clock) time.Time { return clock.Now() }

// Store took its collaborators as fields.
type Store struct {
	read   func(string) ([]byte, error)
	client *http.Client
}

// Read reads through the injected function.
func (s Store) Read(at string) ([]byte, error) { return s.read(at) }

// Fetch sends through the injected client, one selector deep on a value.
func (s Store) Fetch(req *http.Request) (*http.Response, error) { return s.client.Do(req) }

// Inject hands the real implementations to the constructor. Passing the
// function and the client is the injection this rule exists to encourage — a
// reference, not a call.
func Inject() Store { return Store{read: os.ReadFile, client: http.DefaultClient} }
