// Command root is a composition root by package clause: a main package wires
// the real world in, so nothing here is reported.
package main

import (
	"os"
	"time"
)

func main() {
	_, _ = os.ReadFile("config")
	_ = time.Now()
}
