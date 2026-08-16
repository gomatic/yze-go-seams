// Package named varies the file NAME in each dimension isTestFile's matcher
// reads, so a widening of that matcher cannot pass unnoticed. Every file here
// but named_test.go is ordinary source the build compiles and ships, and the
// go tool's own test-file rule is the literal suffix `_test.go`,
// case-sensitively — so each of the three near-misses below is judged, and the
// real test file is not.
package named

import "os/exec"

// Control is the ordinary source file's reach, reported.
func Control(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}
