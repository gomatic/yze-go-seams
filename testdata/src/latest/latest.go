// Package latest is ordinary production code whose NAME ends in the letters
// "test". It imports no testing package, so it is in scope and every reach for
// the real world below is reported.
//
// This is the fixture that forbids the tempting rule. `latest` and `pgtest`
// are the same string shape — a suffix match on "test" cannot separate a word
// from a test-support package, so the spelling is never consulted.
package latest

import (
	"os"
	"time"
)

// Released is the moment the newest release was cut, read from the real clock.
func Released() time.Time {
	return time.Now()
}

// Manifest reads the newest manifest off the real filesystem.
func Manifest(at string) ([]byte, error) {
	return os.ReadFile(at)
}
