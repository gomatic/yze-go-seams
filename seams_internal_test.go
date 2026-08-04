package seams

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

// pkgSelector builds `<spelling>.<name>` with the qualifier resolved to the
// imported package at path, exactly as the type checker would resolve it.
func pkgSelector(path, spelling, name string) (*ast.SelectorExpr, *types.Info) {
	ident := &ast.Ident{Name: spelling}
	info := &types.Info{Uses: map[*ast.Ident]types.Object{
		ident: types.NewPkgName(0, nil, spelling, types.NewPackage(path, spelling)),
	}}
	return &ast.SelectorExpr{X: ident, Sel: &ast.Ident{Name: name}}, info
}

// TestASymbolNamesByPathNotBySpelling pins the resolution the whole rule rests
// on: two different packages that both spell themselves `rand` yield two
// different symbols, so listing the global math source can never silence — or
// implicate — OS entropy by accident.
func TestASymbolNamesByPathNotBySpelling(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	mathSel, mathInfo := pkgSelector("math/rand/v2", "rand", "IntN")
	got, ok := qualify(mathInfo, mathSel)
	want.True(ok)
	want.Equal(symbol("math/rand/v2.IntN"), got)

	cryptoSel, cryptoInfo := pkgSelector("crypto/rand", "rand", "Read")
	got, ok = qualify(cryptoInfo, cryptoSel)
	want.True(ok)
	want.Equal(symbol("crypto/rand.Read"), got)
}

// TestQualifyRejectsAValueQualifier pins the guard that keeps every injected
// collaborator silent: a qualifier the checker resolved to something other
// than an imported package names no stdlib symbol, and neither does a
// qualifier that is not a bare identifier at all.
func TestQualifyRejectsAValueQualifier(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	local := &ast.Ident{Name: "clock"}
	valued := &types.Info{Uses: map[*ast.Ident]types.Object{
		local: types.NewVar(0, nil, "clock", types.Typ[types.Int]),
	}}
	_, ok := qualify(valued, &ast.SelectorExpr{X: local, Sel: &ast.Ident{Name: "Now"}})
	want.False(ok, "a value named like a package is not a package")

	nested := &ast.SelectorExpr{
		X:   &ast.SelectorExpr{X: &ast.Ident{Name: "s"}, Sel: &ast.Ident{Name: "client"}},
		Sel: &ast.Ident{Name: "Do"},
	}
	_, ok = qualify(&types.Info{}, nested)
	want.False(ok, "a field access is not a package qualifier")
}

// TestReachedIsSilentOnACallThroughAnIdentifier pins the blessed seam at the
// use site: `readFile(at)` calls through an identifier, which names no package
// and is therefore never reported, whatever the identifier is bound to.
func TestReachedIsSilentOnACallThroughAnIdentifier(t *testing.T) {
	t.Parallel()

	_, ok := reached(&types.Info{}, &ast.Ident{Name: "readFile"})
	assert.False(t, ok)
}

// TestReachedNamesOnlyListedEntryPoints pins the list itself, in both
// directions and at the boundary that shrank it.
//
// time.Now is on the list; time.Since is not, so a pure function of its
// arguments is never mistaken for a reading of the world. os.ReadFile is not
// on it either, and that is the deliberate part: the filesystem was removed
// after producing 146 of 187 fleet findings with no defect among them, because
// t.TempDir reaches those branches and a rule cannot demand a seam for
// something already fully covered without one.
func TestReachedNamesOnlyListedEntryPoints(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	impureSel, impureInfo := pkgSelector("time", "time", "Now")
	got, ok := reached(impureInfo, impureSel)
	want.True(ok)
	want.Equal(symbol("time.Now"), got)

	pureSel, pureInfo := pkgSelector("time", "time", "Since")
	_, ok = reached(pureInfo, pureSel)
	want.False(ok, "time.Since is arithmetic over its argument")

	fsSel, fsInfo := pkgSelector("os", "os", "ReadFile")
	_, ok = reached(fsInfo, fsSel)
	want.False(ok, "the filesystem is reachable from a test and is off the list")
}

// TestViaGlobalNamesOnlyListedGlobals pins the one-selector-deeper shape and
// its boundary: a method call on net/http.DefaultClient is reported, and one
// on net/http.DefaultServeMux — process-wide state the rule does not claim —
// is not.
func TestViaGlobalNamesOnlyListedGlobals(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	client, info := pkgSelector("net/http", "http", "DefaultClient")
	got, ok := viaGlobal(info, &ast.SelectorExpr{X: client, Sel: &ast.Ident{Name: "Do"}})
	want.True(ok)
	want.Equal(symbol("net/http.DefaultClient"), got)

	mux, muxInfo := pkgSelector("net/http", "http", "DefaultServeMux")
	_, ok = viaGlobal(muxInfo, &ast.SelectorExpr{X: mux, Sel: &ast.Ident{Name: "Handle"}})
	want.False(ok, "an unlisted global is not claimed")

	shallow, shallowInfo := pkgSelector("os", "os", "ReadFile")
	_, ok = viaGlobal(shallowInfo, shallow)
	want.False(ok, "a package qualifier is not a global")
}

// importing builds a file whose import specs name the given paths.
func importing(paths ...string) *ast.File {
	specs := make([]*ast.ImportSpec, len(paths))
	for at, path := range paths {
		specs[at] = &ast.ImportSpec{Path: &ast.BasicLit{Value: strconv.Quote(path)}}
	}
	return &ast.File{Imports: specs}
}

// TestImportsTestingWantsTheFrameworkItself pins what marks test support: the
// `testing` import path exactly. A subpackage is a different package that
// registers no flags and appears in ordinary code — `testing/fstest` builds an
// in-memory filesystem, `testing/quick` generates values — so neither may be
// mistaken for the framework, and neither may a package whose path merely ends
// in the letters.
func TestImportsTestingWantsTheFrameworkItself(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.True(importsTesting(importing("testing")), "the framework marks test support")
	want.True(importsTesting(importing("os", "testing", "time")), "any position counts")

	want.False(importsTesting(importing()), "a file importing nothing imports no framework")
	want.False(importsTesting(importing("os", "time")), "ordinary imports are not the framework")
	want.False(importsTesting(importing("testing/fstest")), "a subpackage is not the framework")
	want.False(importsTesting(importing("testing/quick")), "nor is the generator")
	want.False(importsTesting(importing("github.com/gomatic/testing")), "nor a path that ends in it")
}

// filed adds a source file of the given name to fset and returns an ast.File
// positioned inside it, importing the given paths — enough for isTestFile to
// resolve the name and for importsTesting to read the imports.
func filed(fset *token.FileSet, name string, paths ...string) *ast.File {
	file := importing(paths...)
	added := fset.AddFile(name, -1, 1)
	added.SetLines([]int{0})
	file.Package = added.Pos(0)
	return file
}

// passOver builds a pass over the given files, with the package identity the
// exemptions are decided against.
func passOver(fset *token.FileSet, files ...*ast.File) *analysis.Pass {
	return &analysis.Pass{Fset: fset, Files: files, Pkg: types.NewPackage("mod/store", "store")}
}

// TestIsTestSupportConsultsOnlyNonTestFiles pins the invariant the whole
// exemption rests on, in both directions. A NON-test file importing `testing`
// marks the package as test support; a _test.go file importing it marks
// nothing, because every tested package in existence has one — honouring that
// would exempt the fleet and silence the rule everywhere.
func TestIsTestSupportConsultsOnlyNonTestFiles(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	support := token.NewFileSet()
	want.True(isTestSupport(passOver(support, filed(support, "harness.go", "testing"))),
		"a non-test file importing testing is test support")

	tested := token.NewFileSet()
	want.False(isTestSupport(passOver(tested,
		filed(tested, "store.go", "os"),
		filed(tested, "store_test.go", "testing"),
	)), "a test file importing testing is every tested package, and exempts none")

	ordinary := token.NewFileSet()
	want.False(isTestSupport(passOver(ordinary, filed(ordinary, "store.go", "os", "time"))),
		"production code importing no framework is in scope")
}

// TestIsExemptCoversBothWholePackageExemptions pins that a package leaves scope
// for either reason on its own — the composition root by its path, test support
// by its imports — and stays in scope when neither holds.
func TestIsExemptCoversBothWholePackageExemptions(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	fset := token.NewFileSet()
	ordinary := passOver(fset, filed(fset, "store.go", "os"))
	want.False(isExempt(ordinary), "ordinary code is judged")

	rooted := passOver(fset, filed(fset, "main.go", "os"))
	rooted.Pkg = types.NewPackage("mod/cmd/tool", "tool")
	want.True(isExempt(rooted), "a composition root is exempt")

	supportFset := token.NewFileSet()
	support := passOver(supportFset, filed(supportFset, "harness.go", "testing"))
	want.True(isExempt(support), "test support is exempt")
}

// TestIsCompositionRootMatchesAPathElement pins the exemption's boundary. A
// main package and anything beneath a `cmd` element are where the real world
// is wired in; a package merely whose name CONTAINS "cmd" — `internal/command`
// — is ordinary code and stays in scope.
func TestIsCompositionRootMatchesAPathElement(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.True(isCompositionRoot("anything/at/all", "main"), "a main package is a composition root")
	want.True(isCompositionRoot("mod/cmd/tool", "tool"), "a package beneath cmd is a composition root")
	want.True(isCompositionRoot("cmd", "cmd"), "the cmd element may be the whole path")
	want.False(isCompositionRoot("mod/internal/command", "command"), "a substring is not an element")
	want.False(isCompositionRoot("mod/store", "store"), "ordinary code is in scope")
}

// TestBranchingClocksSeparatesAComparisonFromAStamp names branchingClocks's
// claim: only a clock reaching a comparison puts a branch out of a test's
// reach.
//
// Conflating the two was this rule's largest remaining source of noise — 24 of
// its 37 fleet findings were a timestamp written into a field or a lock file,
// where nothing downstream tests the value. An injected clock buys those sites
// a constructor parameter and no coverage.
func TestBranchingClocksSeparatesAComparisonFromAStamp(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.Empty(branchingClocks(mustParseExpr(t, `time.Now().UTC().Format(x)`)),
		"a stamped clock reaches no comparison")
	want.NotEmpty(branchingClocks(mustParseExpr(t, `time.Now().After(deadline)`)),
		"a compared clock does")
	want.NotEmpty(branchingClocks(mustParseExpr(t, `start.Before(time.Now())`)),
		"the clock as the argument is equally a branch")
	want.NotEmpty(branchingClocks(mustParseExpr(t, `time.Now().Sub(start) > budget`)),
		"and so is a comparison reached through arithmetic")
}

// TestComparisonMethodsHoldOnlyTheClockBranchingOnes names comparisonMethods's
// claim: these are the time.Time methods a test cannot steer at a direct call
// site, because it cannot choose what the clock returns. Formatting or
// re-zoning a timestamp is neither.
func TestComparisonMethodsHoldOnlyTheClockBranchingOnes(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.NotEmpty(comparisonMethods, "an empty set would make nothing a branch")
	for _, method := range []string{"After", "Before", "Equal", "Compare", "Sub"} {
		want.True(comparisonMethods[method], "%s turns a timestamp into a branch", method)
	}
	want.False(comparisonMethods["Format"], "formatting a timestamp is a stamp, not a branch")
	want.False(comparisonMethods["UTC"], "changing zone is a stamp, not a branch")
}

// TestComparisonOpsHoldOnlyRelationalOperators names comparisonOps's claim.
// The set is keyed by the operator's text because token.Token enumerates every
// operator AND keyword in the language, so the six that matter would never be
// stated plainly among the seventy that cannot appear here.
func TestComparisonOpsHoldOnlyRelationalOperators(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.NotEmpty(comparisonOps, "an empty operator set would make nothing a branch")
	for _, op := range []string{"<", ">", "<=", ">=", "==", "!="} {
		want.True(comparisonOps[op], "%s compares", op)
	}
	want.False(comparisonOps["+"], "arithmetic alone does not compare")
	want.False(comparisonOps["&&"], "nor does a logical connective")
}

// mustParseExpr parses a Go expression for the tests above.
func mustParseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	require.NoError(t, err)
	return expr
}
