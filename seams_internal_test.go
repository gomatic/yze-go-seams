package seams

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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
// time.Now, time.Since, and time.Until are all on the list: the stdlib defines
// Since and Until as shorthand for a Now it takes on the caller's behalf, so
// the common spelling reads the clock exactly as the explicit one does — an
// earlier revision called Since "arithmetic over its argument", which was
// false and guarded a real miss. os.ReadFile is NOT on the list, and that is
// the deliberate part: the filesystem was removed after producing 146 of 187
// fleet findings with no defect among them, because t.TempDir reaches those
// branches and a rule cannot demand a seam for something already fully covered
// without one.
func TestReachedNamesOnlyListedEntryPoints(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	impureSel, impureInfo := pkgSelector("time", "time", "Now")
	got, ok := reached(impureInfo, impureSel)
	want.True(ok)
	want.Equal(symbol("time.Now"), got)

	sinceSel, sinceInfo := pkgSelector("time", "time", "Since")
	got, ok = reached(sinceInfo, sinceSel)
	want.True(ok, "time.Since is time.Now().Sub in the stdlib's own definition")
	want.Equal(symbol("time.Since"), got)

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
