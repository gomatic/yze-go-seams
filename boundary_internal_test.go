package seams

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseFile parses one source string for the boundary walks.
func parseFile(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "boundary_fixture.go", src, 0)
	require.NoError(t, err)
	return file
}

// TestInertIdentsMarksOnlyUntypedBlankSpecs pins the evidence rule the value
// walk rests on: a bare `var _ = fn` holds nothing a test could replace, so
// its references are inert — while the typed assertion `var _ runner = fn`
// and the named binding `var seam = fn` are real holds and stay live.
func TestInertIdentsMarksOnlyUntypedBlankSpecs(t *testing.T) {
	t.Parallel()

	file := parseFile(t, `package p
type runner func() error
func fn() error { return nil }
var _ = fn
var _ runner = fn
var seam = fn
`)

	inert := inertIdents(file)

	names := map[string]int{}
	for ident := range inert {
		names[ident.Name]++
	}
	assert.Equal(t, map[string]int{"fn": 1}, names,
		"only the untyped blank spec's reference is inert; the typed assertion and the named binding hold")
}

// TestIsUntypedBlankSpecRequiresEveryNameBlankAndNoType pins the guard's exact
// boundary: a type annotation, any non-blank name, or an empty name list each
// disqualify the spec.
func TestIsUntypedBlankSpecRequiresEveryNameBlankAndNoType(t *testing.T) {
	t.Parallel()

	file := parseFile(t, `package p
type runner func() error
func fn() error { return nil }
var _ = fn
var _ runner = fn
var _, keep = fn, fn
`)

	var specs []*ast.ValueSpec
	ast.Inspect(file, func(n ast.Node) bool {
		if spec, ok := n.(*ast.ValueSpec); ok {
			specs = append(specs, spec)
		}
		return true
	})
	require.Len(t, specs, 3)

	assert.True(t, isUntypedBlankSpec(specs[0]), "bare blank, no type")
	assert.False(t, isUntypedBlankSpec(specs[1]), "a type annotation is a compiler-checked assertion")
	assert.False(t, isUntypedBlankSpec(specs[2]), "a non-blank sibling name is a real hold")
}
