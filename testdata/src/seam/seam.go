// Package seam writes every conforming form: the package-level seam variable
// the standard blesses, injected collaborators, and the pure siblings of the
// impure entry points. Nothing here is reported.
package seam

import (
	"math/rand/v2"
	"net/http"
	"os/exec"
	"time"
)

// spawn is the seam the standard blesses explicitly — a reference to a LISTED
// stdlib function, never a call, so a test can replace it and reach the branch
// behind it. It is listed on purpose: a seam over an unlisted surface would be
// silent whether the seam worked or not.
var spawn = exec.Command

// now is the clock seam.
var now = time.Now

// runQuiet is the same seam written as a closure, because the seam's
// signature is not the stdlib's. A function literal in a NAMED package-level
// var initializer is a seam declaration, exactly like the two above — the
// subprocess spawn inside it is the seam's own implementation, not a call
// site that lacks one.
var runQuiet = func(name string) error {
	return exec.Command(name).Run()
}

// fetchOnce keeps the closure-seam shape over another listed surface, so the
// two spellings are pinned on more than one entry point.
var fetchOnce = func(at string) (*http.Response, error) {
	return http.Get(at)
}

// RunConfig runs through the seam, so its qualifier is an identifier rather
// than a package.
func RunConfig(name string) error { return spawn(name).Run() }

// FetchOnce goes through the closure seam.
func FetchOnce(at string) (*http.Response, error) { return fetchOnce(at) }

// Stamp reads the clock through the seam.
func Stamp() time.Time { return now() }

// Elapsed reads the clock through time.Since but only STAMPS the result —
// nothing here branches on it — so it is silent for the same reason a bare
// time.Now stamp is, not because Since is pure (it is not: it reads the
// clock on the caller's behalf).
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
	run    func(name string, args ...string) *exec.Cmd
	client *http.Client
}

// Run runs through the injected function.
func (s Store) Run(name string) error { return s.run(name).Run() }

// Fetch sends through the injected client, one selector deep on a value.
func (s Store) Fetch(req *http.Request) (*http.Response, error) { return s.client.Do(req) }

// Inject hands the real implementations to the constructor. Passing the
// function and the client is the injection this rule exists to encourage — a
// reference, not a call — and exec.Command is on the list, so the silence is
// the reference rule's rather than the symbol's.
func Inject() Store { return Store{run: exec.Command, client: http.DefaultClient} }

// Remaining stamps how long is left, through time.Until: a returned
// measurement is a stamp, exactly like Elapsed's.
func Remaining(deadline time.Time) time.Duration { return time.Until(deadline) }

// Measured stamps the same measurement through the Sub spelling Since is
// defined to equal: the two spellings draw one silent verdict.
func Measured(start time.Time) time.Duration { return time.Now().Sub(start) }

// Ordered stamps an ordering: a returned Compare is a value, not a branch.
func Ordered(a, b time.Time) int { return a.Compare(b) }

// Recorded hands a clock reading to an injected recorder and branches on the
// RECORDER's error: the comparison judges the sink, not the clock, and the
// stamp doctrine keeps the reading silent.
func Recorded(sink func(time.Duration) error, start time.Time) bool {
	return sink(time.Since(start)) != nil
}
