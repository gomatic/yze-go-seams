// Package direct reaches for the real world at the call site: every LISTED
// call here is the defect this rule exists to catch at authorship.
//
// It carries the list's own boundary with it. The filesystem trio below is
// deliberately off the list and stays silent beside its reported siblings, so
// putting the filesystem back changes this fixture's verdict.
package direct

import (
	crand "crypto/rand"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// started stamps the clock into package state at init. A stamp is not a
// branch, so an initializer's bare time.Now is as silent as any other stamp.
var started = time.Now()

// expiredAtInit BRANCHES on the clock in an initializer, which is the same
// unreachable branch wherever it is written.
var expiredAtInit = time.Now().After(time.Time{}) // want `time.Now is called directly`

// ReadConfig reaches the real filesystem and is NOT reported: t.TempDir gives
// a test a real directory and a bad path reaches the failure branch, so the
// rule's premise does not hold here and the filesystem is off the list.
func ReadConfig(at string) ([]byte, error) {
	return os.ReadFile(at)
}

// WriteConfig writes to the real filesystem, and is silent for the same
// reason.
func WriteConfig(at string, data []byte) error {
	return os.WriteFile(at, data, 0o600)
}

// Discard deletes for real, and is silent for the same reason.
func Discard(at string) error {
	return os.Remove(at)
}

// Stamp reads the clock.
func Stamp() time.Time {
	return time.Now()
}

// Since is the value captured at init, kept so `started` is used.
func Since() time.Duration {
	return time.Since(started)
}

// Pick draws from the hidden global source.
func Pick() int {
	return rand.IntN(10) // want `math/rand/v2.IntN is called directly`
}

// Token reads OS entropy.
func Token() string {
	return crand.Text() // want `crypto/rand.Text is called directly`
}

// Fetch performs a real request.
func Fetch(at string) (*http.Response, error) {
	return http.Get(at) // want `net/http.Get is called directly`
}

// Connect opens a real socket.
func Connect(at string) (net.Conn, error) {
	return net.Dial("tcp", at) // want `net.Dial is called directly`
}

// Run spawns a real process.
func Run(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// Cleanup returns a closure written inside a function body. That is ordinary
// code, not a seam declaration, so the call inside it is reported.
func Cleanup(name string) func() {
	return func() {
		_ = exec.Command(name).Run() // want `os/exec.Command is called directly`
	}
}

// Expired branches on the clock, so a test cannot reach both sides of it
// without controlling what Now returns. This is the shape the rule exists for.
func Expired(deadline time.Time) bool {
	return time.Now().After(deadline) // want `time.Now is called directly`
}

// Elapsed compares in the other direction, with the clock as the ARGUMENT.
func Elapsed(start time.Time) bool {
	return start.Before(time.Now()) // want `time.Now is called directly`
}

// StaleSince spells the same clock-bound branch through time.Since, which is
// time.Now().Sub in the stdlib's own definition — the common spelling must not
// silence the shape the rule was written for.
func StaleSince(start time.Time, ttl time.Duration) bool {
	return time.Since(start) > ttl // want `time.Since is called directly`
}

// OverBudget reaches a comparison through arithmetic.
func OverBudget(start time.Time, budget time.Duration) bool {
	return time.Now().Sub(start) > budget // want `time.Now is called directly`
}

// Bumped reaches the clock inside a binary expression that COMPARES nothing:
// arithmetic is not a branch, so the reading is a stamp and stays silent. Only
// a comparison operator puts a branch out of a test's reach.
func Bumped(base int64) int64 {
	return time.Now().Unix() + base
}

// StampOnly records the clock without testing it: no branch depends on the
// value, so there is nothing a test cannot reach and nothing to report.
func StampOnly() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// UntilDeadline branches on the clock through time.Until, which reads the
// clock exactly as Now does.
func UntilDeadline(deadline time.Time) bool {
	return time.Until(deadline) <= 0 // want `time.Until is called directly`
}

// nothing binds this literal, so it is not a seam a test can rebind and its
// body is ordinary code.
var _ = func(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// cached memoises a spawn. The literal is buried inside the initializer's
// expression rather than being the initializer, so it is not a seam either: a
// test can rebind cached, but not the closure sync.OnceValue closed over.
var cached = sync.OnceValue(func() []byte {
	out, _ := exec.Command("git", "rev-parse", "HEAD").Output() // want `os/exec.Command is called directly`
	return out
})

// Cached reads the memoised value.
func Cached() []byte { return cached() }
