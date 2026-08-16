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
// carrying its real type info, so the hold walk is exercised with the
// checker's own answers rather than hand-built stand-ins.
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

// heldNames flattens the held expressions to the identifier spellings they
// name, with counts, so a claim about WHICH holds were recognised can be made
// without depending on their order.
func heldNames(held []ast.Expr) map[string]int {
	names := map[string]int{}
	for _, expr := range held {
		if ident, ok := heldIdent(expr); ok {
			names[ident.Name]++
		}
	}
	return names
}

// TestHeldExprsWantsAHoldATestCanReplace names the whole content of the
// value-function exemption: a reference marks only when it binds the function
// somewhere a test can write over.
//
// The four that qualify are the four a test can substitute — a package-level
// var, a composite-literal element, an assignment to a field, and an argument
// at a parameter declared with a FUNCTION type, which is the constructor
// injection the rule exists to encourage. A local capture, a return value, and
// an argument at an `any` parameter bind the function nowhere a test can
// reach, so each stays reported; four lines of one of them would otherwise
// silence any function in the package.
func TestHeldExprsWantsAHoldATestCanReplace(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
type runner func() error
type store struct{ run runner }
func packaged() error { return nil }
func element() error { return nil }
func fielded() error { return nil }
func assigned() error { return nil }
func captured() error { return nil }
func passed() error { return nil }
func returned() error { return nil }
func loose() error { return nil }
var seam runner = packaged
var chain = []runner{element}
var built = store{run: fielded}
func Bind(s *store) { s.run = assigned }
func Capture() error { f := captured; return f() }
func Pass() error { return apply(passed) }
func apply(with runner) error { return with() }
func Loose() { record(loose) }
func record(what any) { _ = what }
func Return() runner { return returned }
`)

	got := heldNames(heldExprs(pass, file))

	assert.Equal(
		t,
		map[string]int{"packaged": 1, "element": 1, "fielded": 1, "assigned": 1, "passed": 1},
		got,
		"a package-level var, a composite-literal element, a field assignment and a function-typed parameter are holds; a local, an any-parameter and a return are not",
	)
}

// TestHeldExprsJudgesBlankPositionsNotWholeSpecs pins the per-position rule a
// mixed binding demands — in `var _, keep = a, b` only the named position
// holds — and the rule for a binding whose values do not align with its names:
// a multi-value binding from one expression holds the CALL'S results and names
// no function, in a declaration and in an assignment alike.
func TestHeldExprsJudgesBlankPositionsNotWholeSpecs(t *testing.T) {
	t.Parallel()

	pass, file := checkedFile(t, `package p
type runner func() error
type store struct{ first, second runner }
func a() error { return nil }
func b() error { return nil }
func pair() (runner, runner) { return a, b }
var _, keep = a, b
var one, two = pair()
func Rebind(s *store) { s.first, s.second = pair() }
`)

	got := heldNames(heldExprs(pass, file))

	assert.Zero(t, got["a"], "the blank position holds nothing")
	assert.Equal(t, 1, got["b"], "the named sibling in the same spec holds")
	assert.Zero(t, got["pair"], "an unaligned multi-value binding holds the results, not the function that made them")
}

// TestHeldExprsHonoursOnlyACompilerCheckedBlank pins the one blank that is
// evidence. `var _ = fn` and `var _ any = fn` hold nothing a test could
// replace — every value satisfies any, so the second asserts as little as the
// first — while `var _ runner = fn` is the package stating, checked by the
// compiler, that fn backs a declared seam shape.
func TestHeldExprsHonoursOnlyACompilerCheckedBlank(t *testing.T) {
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

	got := heldNames(heldExprs(pass, file))

	assert.Equal(t, map[string]int{"asserted": 1}, got,
		"only the function-typed blank assertion is evidence of a seam")
}

// TestHeldIdentNamesEveryFunctionSpelling pins what a hold can name: a plain
// function, a method or another package's function through a selector, and an
// explicit generic instantiation in either spelling — an instantiation is the
// same function, so holding one is holding it. A call names no function to
// hold, which is what keeps `var cached = sync.OnceValue(func…)` from marking.
func TestHeldIdentNamesEveryFunctionSpelling(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	plain, ok := heldIdent(ast.NewIdent("execRun"))
	want.True(ok)
	want.Equal("execRun", plain.Name)

	through, ok := heldIdent(&ast.SelectorExpr{X: ast.NewIdent("os"), Sel: ast.NewIdent("ReadFile")})
	want.True(ok)
	want.Equal("ReadFile", through.Name)

	one, ok := heldIdent(&ast.IndexExpr{X: ast.NewIdent("spawn"), Index: ast.NewIdent("int")})
	want.True(ok)
	want.Equal("spawn", one.Name, "an instantiation is the function it instantiates")

	many, ok := heldIdent(&ast.IndexListExpr{
		X:       ast.NewIdent("spawn"),
		Indices: []ast.Expr{ast.NewIdent("int"), ast.NewIdent("string")},
	})
	want.True(ok)
	want.Equal("spawn", many.Name, "and so is a two-argument one")

	_, ok = heldIdent(&ast.CallExpr{Fun: ast.NewIdent("OnceValue")})
	want.False(ok, "a call names its result, not a function anything holds")
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
