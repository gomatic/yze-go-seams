// Package functype declares a function-type seam and its default
// implementation. The named function type is the seam — a caller holds one, a
// test substitutes one — and the package function with exactly that signature
// is the real implementation behind it, bound at a composition root outside
// this package. It is silent for the same reason an interface adapter's
// methods are.
package functype

import "os/exec"

// Runner is the seam: consumers hold one, and a test substitutes one.
type Runner func(name string, args ...string) ([]byte, error)

// ExecRun is Runner's real implementation, bound by composition roots outside
// this package. The in-package value walk cannot see that binding; the type
// identity is what proves the seam exists.
func ExecRun(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// Probe matches no declared function type, so its direct call is the defect
// the rule exists to catch.
func Probe(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}
