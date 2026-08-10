// Package direct reaches for the real world at the call site: every call here
// is the defect this rule exists to catch at authorship.
package direct

import (
	crand "crypto/rand"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// started stamps the clock into package state at init. A stamp is not a
// branch, so an initializer's bare time.Now is as silent as any other stamp.
var started = time.Now()

// expiredAtInit BRANCHES on the clock in an initializer, which is the same
// unreachable branch wherever it is written.
var expiredAtInit = time.Now().After(time.Time{}) // want `time.Now is called directly`

// ReadConfig has an error branch no test can enter.
func ReadConfig(at string) ([]byte, error) {
	return os.ReadFile(at)
}

// WriteConfig writes to the real filesystem.
func WriteConfig(at string, data []byte) error {
	return os.WriteFile(at, data, 0o600)
}

// Discard deletes for real.
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

// StampOnly records the clock without testing it: no branch depends on the
// value, so there is nothing a test cannot reach and nothing to report.
func StampOnly() string {
	return time.Now().UTC().Format(time.RFC3339)
}
