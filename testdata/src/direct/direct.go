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

// started captures the clock into package state at init — a direct call is a
// direct call wherever it is written.
var started = time.Now() // want `time.Now is called directly`

// ReadConfig has an error branch no test can enter.
func ReadConfig(at string) ([]byte, error) {
	return os.ReadFile(at) // want `os.ReadFile is called directly`
}

// WriteConfig writes to the real filesystem.
func WriteConfig(at string, data []byte) error {
	return os.WriteFile(at, data, 0o600) // want `os.WriteFile is called directly`
}

// Discard deletes for real.
func Discard(at string) error {
	return os.Remove(at) // want `os.Remove is called directly`
}

// Stamp reads the clock.
func Stamp() time.Time {
	return time.Now() // want `time.Now is called directly`
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
func Cleanup(at string) func() {
	return func() {
		_ = os.RemoveAll(at) // want `os.RemoveAll is called directly`
	}
}
