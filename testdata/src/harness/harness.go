// Package harness is test support: it exists to be imported by other packages'
// tests, and it proves that by importing `testing` from this NON-test file.
// Every reach for the real world below is therefore out of scope, exactly as it
// would be inside a _test.go file.
//
// Its name carries no `test` suffix on purpose. The rule is the import, not the
// spelling — `internal/testutil` and `tests/harness` are test support without
// one, and a name-based rule would report them both.
package harness

import (
	"os"
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

// Stamp reads the real clock, which a test-support package may do freely.
func Stamp() time.Time {
	return time.Now()
}

// Read pulls a fixture back off the real filesystem.
func Read(at string) ([]byte, error) {
	return os.ReadFile(at)
}
