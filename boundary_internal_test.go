package seams

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInjectableWantsSomethingThatCanHoldTheInterface pins the interface half
// of the same doctrine. An exported name is part of the package's API, so any
// importer can hold one; an unexported name has to be written as the type of
// something before an implementation can be handed through it. An unexported
// interface nobody writes down gives a test nothing to substitute, so
// declaring one exempts nothing.
func TestInjectableWantsSomethingThatCanHoldTheInterface(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	exported := types.NewTypeName(0, types.NewPackage("p", "p"), "Fetcher", nil)
	unexported := types.NewTypeName(0, types.NewPackage("p", "p"), "doer", nil)

	want.True(injectable(exported, map[types.Object]bool{}), "an importer can hold an exported interface")
	want.False(injectable(unexported, map[types.Object]bool{}), "nothing can hold an interface nobody writes down")
	want.True(injectable(unexported, map[types.Object]bool{unexported: true}),
		"an unexported interface written as a type is one the package is handed through")
}
