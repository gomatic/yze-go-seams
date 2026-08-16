// Package boundary pins the silences the package comment promises, each beside
// a reported sibling so the silence is the rule's and never the fixture being
// unreachable: the process environment stays off the list, an adapter for
// ANOTHER package's interface is reported like any other method, and a thin
// public wrapper over a non-clock entry point is reported while the clock's
// stamp/branch narrowing keeps its wrapper silent.
package boundary

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// Home reads the process environment, which is deliberately not on the list:
// configuration read once at the edge would be reported far more often than a
// testability defect.
func Home() string { return os.Getenv("HOME") }

// Spawn is the listed sibling that proves the package is judged at all.
func Spawn(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// Piped satisfies io.Reader, an interface from ANOTHER package. Whether an
// exported type is someone else's collaborator cannot be decided from this
// package's syntax, so the method is reported like any other.
type Piped struct{}

// Read spawns the real process behind the foreign interface.
func (Piped) Read(into []byte) (int, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output() // want `os/exec.Command is called directly`
	if err != nil {
		return 0, err
	}
	return copy(into, out), nil
}

// The foreign interface Piped satisfies, asserted so the shape is real.
var _ io.Reader = Piped{}

// generateAt is the fully-tested implementation the wrapper below pushes the
// impurity out of.
func generateAt(seed int, at time.Time) string { return strconv.Itoa(seed) + at.String() }

// Generate is the thin public wrapper over generateAt. The clock reading is
// consumed as an ARGUMENT, which is a stamp under the branching narrowing, so
// it is silent — the wrapper shape itself is not what decides it.
func Generate(seed int) string { return generateAt(seed, time.Now()) }

// status is the fully-tested implementation behind the other wrapper.
func status(res *http.Response, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	return res.StatusCode, nil
}

// Fetch is the identical wrapper shape over a non-clock entry point, and it IS
// reported: a one-line wrapper is not distinguishable from a domain function
// that reaches for the network.
func Fetch(url string) (int, error) {
	return status(http.Get(url)) // want `net/http.Get is called directly`
}
