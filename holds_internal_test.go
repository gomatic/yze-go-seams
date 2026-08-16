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
// carrying its real type info and package, so the inert walk is exercised with
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
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	require.NoError(t, err)
	return &analysis.Pass{TypesInfo: info, Pkg: pkg}, file
}

// identNames flattens a set of identifiers to spellings with counts, so a
// claim about WHICH references were judged can be made without depending on
// their order.
func identNames(idents map[*ast.Ident]bool) map[string]int {
	names := map[string]int{}
	for ident := range idents {
		names[ident.Name]++
	}
	return names
}

// TestInertIdentsSubtractsWhatATestCannotReplace names the whole content of
// the value-function exemption, in the direction the exemption is written: a
// reference is a hold unless it binds the function where a test can reach
// nothing.
//
// Three bind nothing — the blank identifier, a local variable, and an argument
// at a parameter that is not a function type. Everything else is a hold,
// including the shapes an enumeration of "good" positions kept getting wrong:
// a package-level var written in an `init`, a registry entry, a field, and an
// argument at a function-typed parameter.
func TestInertIdentsSubtractsWhatATestCannotReplace(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
type runner func() error
type store struct{ run runner }
func blanked() error { return nil }
func captured() error { return nil }
func reassigned() error { return nil }
func loose() error { return nil }
func packaged() error { return nil }
func inited() error { return nil }
func registered() error { return nil }
func fielded() error { return nil }
func injected() error { return nil }
var _ = blanked
var packagedSeam runner = packaged
var initedSeam runner
var registry = map[string]runner{}
func init() { initedSeam = inited; registry["a"] = registered }
func Capture() error { f := captured; return f() }
func Reassign() error { var local runner; local = reassigned; return local() }
func Loose() { record(loose) }
func record(any) {}
func Field(s *store) { s.run = fielded }
func Inject() error { return apply(injected) }
func apply(with runner) error { return with() }
`)

	got := identNames(inertIdents(pass, file))

	assert.Equal(
		t,
		map[string]int{"blanked": 1, "captured": 1, "reassigned": 1, "loose": 1},
		got,
		"a blank, a local binding and an any-parameter bind nothing a test can reach; a package var, an init assignment, a registry entry, a field and a function-typed parameter are all holds",
	)
}

// TestInertIdentsJudgesBlankPositionsNotWholeSpecs pins the per-position rule
// a mixed binding demands — in `var _, keep = a, b` only the blank position's
// value is inert — and the rule for a binding whose values do not align with
// its names: it takes several values from ONE expression and names no function
// at any position, so it marks nothing either way.
func TestInertIdentsJudgesBlankPositionsNotWholeSpecs(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
type runner func() error
func a() error { return nil }
func b() error { return nil }
func pair() (runner, runner) { return a, b }
var _, keep = a, b
var _, _ = pair()
`)

	got := identNames(inertIdents(pass, file))

	assert.Equal(t, 1, got["a"], "the blank position's value is inert")
	assert.Zero(t, got["b"], "the named sibling's value holds")
	assert.Zero(t, got["pair"], "an unaligned binding names no function at any position")
}

// TestInertIdentsHonoursOnlyACompilerCheckedBlank pins the one blank that is
// evidence. `var _ = fn` and `var _ any = fn` bind nothing a test could
// replace — every value satisfies any, so the second asserts as little as the
// first — while `var _ runner = fn` is the package stating, checked by the
// compiler, that fn backs a declared seam shape.
func TestInertIdentsHonoursOnlyACompilerCheckedBlank(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
type runner func() error
func bare() error { return nil }
func loose() error { return nil }
func asserted() error { return nil }
var _ = bare
var _ any = loose
var _ runner = asserted
`)

	got := identNames(inertIdents(pass, file))

	assert.Equal(t, map[string]int{"bare": 1, "loose": 1}, got,
		"only the function-typed blank assertion escapes the blank rule")
}

// TestCalleeIdentsNamesEveryCallSpelling pins what counts as calling rather
// than holding: a plain name, a method, a parenthesised callee, and an
// explicit generic instantiation in either spelling. An instantiation is a
// CALL to the function it instantiates, so reading the identifier underneath
// it as a value reference would let one unused type parameter silence a body.
func TestCalleeIdentsNamesEveryCallSpelling(t *testing.T) {
	t.Parallel()

	_, file := checkedFile(t, `package p
func plain() error { return nil }
func method() error { return nil }
func parens() error { return nil }
func one[T any]() error { return nil }
func two[T, U any]() error { return nil }
type holder struct{}
func (holder) method() error { return nil }
func Call(h holder) { _ = plain(); _ = h.method(); _ = (parens)(); _ = one[int](); _ = two[int, string]() }
`)

	got := identNames(calleeIdents(file))

	assert.Equal(t, 1, got["plain"], "a plain callee is a call")
	assert.Equal(t, 1, got["method"], "so is a method")
	assert.Equal(t, 1, got["parens"], "parentheses do not make a callee a value")
	assert.Equal(t, 1, got["one"], "nor does an explicit instantiation")
	assert.Equal(t, 1, got["two"], "nor a two-argument one")
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

// TestIsFuncParamRefusesACallWithNoParameterToBind pins the one call shape
// that carries an argument and declares no parameter at all: a conversion to a
// niladic function type, `(func())(fn)`. There is nothing for the argument to
// bind, so the position is not a function-typed parameter.
func TestIsFuncParamRefusesACallWithNoParameterToBind(t *testing.T) {
	t.Parallel()

	niladic := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	assert.False(t, isFuncParam(niladic, 0), "a call declaring no parameter binds nothing")
}
