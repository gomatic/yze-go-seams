// OS entropy through the package-global reader, in the method spelling the
// rule claims. The io.ReadFull spelling hands the global to a collaborator-
// taking function, which the value-passing boundary leaves silent — a
// documented limit, pinned here so a widening is a choice.
package global

import (
	"crypto/rand"
	"io"
)

// Fill reads entropy through the global reader's own method.
func Fill(buf []byte) (int, error) {
	return rand.Reader.Read(buf) // want `crypto/rand.Reader is called directly`
}

// FillFull hands the global reader to io.ReadFull: passing a collaborator is
// the injection shape, and the rule stays silent by that boundary.
func FillFull(buf []byte) (int, error) {
	return io.ReadFull(rand.Reader, buf)
}
