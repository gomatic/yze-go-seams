// Package cmdutil is ordinary production code whose import path CONTAINS the
// letters "cmd" without carrying a `cmd` ELEMENT. The composition-root
// exemption matches a path element, so this package stays in scope — a
// substring match would silence it, and every other package whose path merely
// spells the letters.
package cmdutil

import "time"

// Expired branches on the real clock and is reported: cmdutil is not beneath a
// cmd element.
func Expired(deadline time.Time) bool {
	return time.Now().After(deadline) // want `time.Now is called directly`
}
