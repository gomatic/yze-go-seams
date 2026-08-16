package seams

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// exemptions is the set of package functions and methods that ARE the
// package's boundary with the real world. A direct call inside one of them is
// the implementation behind an existing seam, not a call site that lacks one.
type exemptions map[types.Object]bool

// adapters is the set of declarations that are the package's own boundary with
// the real world, in the two shapes a Go package can express one.
func adapters(pass *analysis.Pass) exemptions {
	boundary := exemptions{}
	collectValueFuncs(pass, boundary)
	collectInterfaceMethods(pass, boundary)
	return boundary
}

// collectInterfaceMethods marks every method that implements a method of an
// interface the package itself declares AND can be injected through.
//
// That interface IS the injected abstraction the standard asks for, and the
// method is its real implementation: `type Clock interface { Now() time.Time }`
// alongside `func (System) Now() time.Time { return time.Now() }` is the
// pattern, not a defect. Only the methods the interface names are exempt — a
// second method on the same type, outside the interface, is ordinary code.
//
// And only an interface something can be handed through counts. An interface
// nobody holds gives a test nothing to substitute, so it proves no seam: four
// unused lines naming a method would otherwise silence any method that
// happened to match them, which is a marker acquired without the property.
func collectInterfaceMethods(pass *analysis.Pass, boundary exemptions) {
	scope := pass.Pkg.Scope()
	for _, iface := range declaredInterfaces(scope, injectableTypes(pass)) {
		markImplementations(pass.Pkg, scope, iface, boundary)
	}
}

// injectableTypes is every type name this package's non-test files USE — every
// mention of it anywhere outside its own declaration.
//
// Any mention of a type name is a use of it AS a type, and every use of an
// interface type is a place a value of that type can arrive: a parameter, a
// field, a result, a variable, a type assertion, a type-switch case, a
// conversion, a type argument. Enumerating those positions instead was tried
// and is a treadmill — an enumeration naming four of them missed the type
// assertion and the type switch, and each miss reports a package that
// genuinely does inject. The type checker already knows which identifiers name
// types, so it is asked rather than reimplemented.
func injectableTypes(pass *analysis.Pass) map[types.Object]bool {
	held := map[types.Object]bool{}
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			markUsedTypes(pass, file, held)
		}
	}
	return held
}

// markUsedTypes records every type name one file uses. A type's own
// declaration is a definition rather than a use, so an interface nothing
// mentions again stays absent.
func markUsedTypes(pass *analysis.Pass, file *ast.File, held map[types.Object]bool) {
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			markTypeName(pass, ident, held)
		}
		return true
	})
}

// markTypeName records ident when it resolves to a type name.
func markTypeName(pass *analysis.Pass, ident *ast.Ident, held map[types.Object]bool) {
	if name, ok := pass.TypesInfo.Uses[ident].(*types.TypeName); ok {
		held[name] = true
	}
}

// declaredInterfaces is every interface type the package declares that
// something can be injected through.
//
// The empty interface needs no special case. Every type satisfies it, but an
// exemption is granted per interface METHOD, and an empty interface names
// none — so it exempts nothing, and excluding it would be a guard with no
// effect either way.
func declaredInterfaces(scope *types.Scope, held map[types.Object]bool) []*types.Interface {
	var declared []*types.Interface
	for _, name := range scope.Names() {
		at := scope.Lookup(name)
		if iface, ok := namedInterface(at); ok && injectable(at, held) {
			declared = append(declared, iface)
		}
	}
	return declared
}

// injectable reports whether anything can hold the interface. An exported name
// is part of the package's API, so any importer can hold one and hand back
// whatever it likes; an unexported name has to be written as the type of a
// variable, a parameter, a result or a field before anything in the package
// can be handed through it.
func injectable(at types.Object, held map[types.Object]bool) bool {
	return at.Exported() || held[at]
}

// namedInterface is the interface a type name denotes, if it denotes one. An
// alias needs no unwrapping: (*types.Alias).Underlying already resolves
// through it, measured rather than assumed, so types.Unalias here would be a
// call no input could distinguish.
func namedInterface(obj types.Object) (*types.Interface, bool) {
	named, ok := obj.(*types.TypeName)
	if !ok {
		return nil, false
	}
	iface, ok := named.Type().Underlying().(*types.Interface)
	return iface, ok
}

// markImplementations marks the interface's methods on every package type that
// satisfies it.
func markImplementations(pkg *types.Package, scope *types.Scope, iface *types.Interface, boundary exemptions) {
	for _, name := range scope.Names() {
		markImplementor(pkg, iface, scope.Lookup(name), boundary)
	}
}

// markImplementor marks each of iface's methods on obj's type, when that type
// satisfies iface either as a value or through a pointer.
func markImplementor(pkg *types.Package, iface *types.Interface, obj types.Object, boundary exemptions) {
	named, ok := obj.(*types.TypeName)
	if !ok || !satisfies(named.Type(), iface) {
		return
	}
	for i := range iface.NumMethods() {
		markMethod(pkg, named.Type(), methodName(iface.Method(i).Name()), boundary)
	}
}

// satisfies reports whether the type, or a pointer to it, implements iface.
func satisfies(at types.Type, iface *types.Interface) bool {
	return types.Implements(at, iface) || types.Implements(types.NewPointer(at), iface)
}

// methodName is the name of a method in an interface's method set.
type methodName string

// markMethod marks the named method of a type as part of the package's
// boundary.
func markMethod(pkg *types.Package, at types.Type, name methodName, boundary exemptions) {
	found, _, _ := types.LookupFieldOrMethod(at, true, pkg, string(name))
	if method, ok := found.(*types.Func); ok {
		boundary[method] = true
	}
}
