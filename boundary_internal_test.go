package seams

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAFuncTypeDefaultIsMarkedOnlyOnAnIdenticalSignature pins the third
// boundary shape's exact contract: a package function is the declared function
// type's implementation only when the signatures are identical — a
// near-miss signature is ordinary code, and a non-function object is never
// marked at all.
func TestAFuncTypeDefaultIsMarkedOnlyOnAnIdenticalSignature(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("example.com/p", "p")
	str := types.Typ[types.String]
	errType := types.Universe.Lookup("error").Type()
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, pkg, "name", str)),
		types.NewTuple(types.NewVar(0, pkg, "", errType)), false)
	nearMiss := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, pkg, "n", types.Typ[types.Int])),
		types.NewTuple(types.NewVar(0, pkg, "", errType)), false)
	matching := types.NewFunc(0, pkg, "DefaultImpl", sig)
	differing := types.NewFunc(0, pkg, "Other", nearMiss)

	boundary := exemptions{}
	sigs := []*types.Signature{sig}
	markFuncTypeDefault(matching, sigs, boundary)
	markFuncTypeDefault(differing, sigs, boundary)
	markFuncTypeDefault(types.NewTypeName(0, pkg, "NotAFunc", str), sigs, boundary)

	assert.True(t, boundary[matching], "an identical signature is the seam's implementation")
	assert.False(t, boundary[differing], "a near-miss signature is ordinary code")
	assert.Len(t, boundary, 1, "nothing else is marked")
}
