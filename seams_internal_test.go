package seams

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestReachedNamesOnlyListedEntryPoints pins the list itself: os.ReadFile is
// reported and time.Since is not, so a pure function of its arguments is never
// mistaken for a reading of the world.
func TestReachedNamesOnlyListedEntryPoints(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	impureSel, impureInfo := pkgSelector("os", "os", "ReadFile")
	got, ok := reached(impureInfo, impureSel)
	want.True(ok)
	want.Equal(symbol("os.ReadFile"), got)

	pureSel, pureInfo := pkgSelector("time", "time", "Since")
	_, ok = reached(pureInfo, pureSel)
	want.False(ok, "time.Since is arithmetic over its argument")
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
