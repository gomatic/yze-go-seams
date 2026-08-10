package seams

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

// checkedFile type-checks one source string and yields the file with a pass
// carrying its real type info, so the blank-position walk is exercised with
// the checker's own answers rather than hand-built stand-ins.
func checkedFile(t *testing.T, src string) (*analysis.Pass, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "boundary_fixture.go", src, 0)
	require.NoError(t, err)
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{},
		Uses:  map[*ast.Ident]types.Object{},
		Defs:  map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check("p", fset, []*ast.File{file}, info)
	require.NoError(t, err)
	return &analysis.Pass{TypesInfo: info}, file
}

// inertNames flattens the inert set to identifier spellings with counts.
func inertNames(inert map[*ast.Ident]bool) map[string]int {
	names := map[string]int{}
	for ident := range inert {
		names[ident.Name]++
	}
	return names
}

// TestInertIdentsJudgesEveryBlankSpelling pins the evidence rule the value
// walk rests on, across the spellings the adversarial review proved
// equivalent: the untyped blank var, the blank STATEMENT `_ = fn`, and a
// typed blank whose annotation asserts nothing (`var _ any = fn`) are all
// inert — while the function-typed assertion `var _ runner = fn` and the
// named binding `var seam = fn` are real holds and stay live.
func TestInertIdentsJudgesEveryBlankSpelling(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
type runner func() error
func fn() error { return nil }
func gn() error { return nil }
func hn() error { return nil }
var _ = fn
var _ any = gn
var _ runner = fn
var seam = fn
func init() { _ = hn }
`)

	got := inertNames(inertIdents(pass, file))

	assert.Equal(
		t,
		map[string]int{"fn": 1, "gn": 1, "hn": 1},
		got,
		"the untyped blank, the any-typed blank, and the blank statement are inert; the func-typed assertion and the named binding hold",
	)
}

// TestInertIdentsJudgesBlankPositionsNotWholeSpecs pins the per-position rule
// a mixed binding demands: in `var _, keep = a, b` only the blank position's
// value is inert, and a multi-value binding from one call is inert only when
// every target is blank.
func TestInertIdentsJudgesBlankPositionsNotWholeSpecs(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
func a() error { return nil }
func b() error { return nil }
func pair() (func() error, func() error) { return a, b }
var _, keep = a, b
var _, _ = pair()
var x, _ = pair()
var _ = x
var _ = keep
`)

	got := inertNames(inertIdents(pass, file))

	assert.Equal(t, 1, got["a"], "the blank position's value is inert")
	assert.Zero(t, got["b"], "the named sibling's value holds")
	assert.Equal(t, 1, got["pair"], "an all-blank multi-value binding holds nothing")
	assert.Equal(t, 1, got["x"], "a bare blank of a named variable is inert like any other")
}

// TestFuncTypeAnnotationRefusesWhatTheCheckerNeverSaw pins the guard: an
// annotation the type info cannot resolve asserts nothing, and no annotation
// at all asserts nothing.
func TestFuncTypeAnnotationRefusesWhatTheCheckerNeverSaw(t *testing.T) {
	t.Parallel()

	pass := &analysis.Pass{TypesInfo: &types.Info{}}
	assert.False(t, funcTypeAnnotation(pass, nil), "no annotation asserts nothing")
	assert.False(t, funcTypeAnnotation(pass, ast.NewIdent("mystery")),
		"an unresolved annotation asserts nothing")
}
