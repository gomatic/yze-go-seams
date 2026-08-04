// Package tested is ordinary production code that has a test. Its _test.go
// file imports `testing` — as every tested package's does — and that must not
// exempt the package: only a NON-test file importing `testing` marks test
// support, so the reach for the real clock below is still reported.
//
// The reach inside tested_test.go itself stays silent, because what a test does
// with a real resource is the test's own business.
package tested

import "time"

// Name is the value the package's test reads back.
const Name = "tested"

// Stamped reads the real clock from production code, which is reported: the
// package is in scope despite having a test that imports `testing`.
func Stamped() time.Time {
	return time.Now()
}
