// Package latest is ordinary production code whose NAME ends in the letters
// "test". It imports no testing package, so it is in scope and its reaches for
// the listed real-world entry points below are reported.
//
// This is the fixture that forbids the tempting rule. `latest` and `pgtest`
// are the same string shape — a suffix match on "test" cannot separate a word
// from a test-support package, so the spelling is never consulted.
package latest

import (
	"os/exec"
	"time"
)

// Stale branches on the real clock, which is reported: the package is in
// scope whatever its name ends in.
func Stale(cut time.Time, ttl time.Duration) bool {
	return time.Since(cut) > ttl // want `time.Since is called directly`
}

// Manifest reads the newest manifest by spawning the real git.
func Manifest() ([]byte, error) {
	return exec.Command("git", "show", "HEAD:manifest.yaml").Output() // want `os/exec.Command is called directly`
}
