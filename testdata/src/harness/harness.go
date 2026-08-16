// Package harness is test support: it exists to be imported by other packages'
// tests, and it proves that by importing `testing` from this NON-test file.
// Every reach for the real world below is therefore out of scope, exactly as it
// would be inside a _test.go file.
//
// Its name carries no `test` suffix on purpose. The rule is the import, not the
// spelling — `internal/testutil` and `tests/harness` are test support without
// one, and a name-based rule would report them both.
//
// The reaches are ones the rule reports in scope — a subprocess spawn and a
// clock-bound branch — so deleting the exemption changes this fixture's
// verdict rather than passing vacuously.
package harness

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// Fixture writes a file a test will read back, on the real filesystem.
func Fixture(t *testing.T, at string, content []byte) {
	t.Helper()
	if err := os.WriteFile(at, content, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

// Seed spawns the real git to build a fixture repository.
func Seed(t *testing.T, at string) {
	t.Helper()
	if err := exec.Command("git", "init", at).Run(); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
}

// Expired branches on the real clock, which a test-support package may do
// freely.
func Expired(deadline time.Time) bool { return time.Now().After(deadline) }
