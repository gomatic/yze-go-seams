// Package tool is a composition root by import path: it lives beneath a cmd
// element, where handing the real world to the packages that took it as a
// parameter is precisely the job.
//
// Both reaches below are shapes the rule reports in scope, so the exemption is
// what makes this fixture silent.
package tool

import (
	"os/exec"
	"time"
)

// Run spawns a real process, and is not reported.
func Run(name string) error { return exec.Command(name).Run() }

// Expired branches on the real clock, and is not reported.
func Expired(deadline time.Time) bool { return time.Now().After(deadline) }
