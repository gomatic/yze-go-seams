// Package functype pins how a function-type seam earns its exemption: through
// EVIDENCE of binding, never through signature shape alone. The typed blank
// assertion `var _ Runner = ExecRun` is the package stating, checked by the
// compiler, that the function backs the declared seam type a composition root
// binds — and it exempts. A signature that merely matches a declared type
// proves nothing (a `type task func()` would otherwise silence every niladic
// function), and a bare `var _ = fn` holds nothing a test could replace.
package functype

import "os/exec"

// Runner is the seam: consumers hold one, and a test substitutes one.
type Runner func(name string, args ...string) ([]byte, error)

// Asserted backs Runner, and says so.
var _ Runner = ExecRun

// ExecRun is Runner's real implementation, bound by composition roots outside
// this package; the typed assertion above is the in-package evidence.
func ExecRun(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// ShapeOnly matches Runner's signature exactly but nothing binds it or asserts
// it: a matching shape is not a seam, and its direct call is reported.
func ShapeOnly(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output() // want `os/exec.Command is called directly`
}

// inert is referenced only by a bare untyped blank var, which holds nothing a
// test could replace: it stays reported.
func inert(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The bare blank reference that must NOT exempt inert.
var _ = inert
