// Package tool is a composition root by import path: it lives beneath a cmd
// element, where handing the real world to the packages that took it as a
// parameter is precisely the job.
package tool

import (
	"os"
	"time"
)

// Read reaches for the real filesystem, and is not reported.
func Read(at string) ([]byte, error) { return os.ReadFile(at) }

// Stamp reads the real clock, and is not reported.
func Stamp() time.Time { return time.Now() }
