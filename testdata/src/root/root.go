// Command root is a composition root by package clause: a main package wires
// the real world in, so nothing here is reported.
//
// Every reach below is a shape the rule DOES report elsewhere — a subprocess
// spawn and a clock-bound branch — so deleting the exemption changes this
// fixture's verdict. A filesystem read or a bare clock stamp would be silent
// in scope as well, and would prove nothing about the exemption.
package main

import (
	"os/exec"
	"time"
)

func main() {
	_ = exec.Command("git", "status").Run()
	if time.Now().After(time.Time{}) {
		return
	}
}
