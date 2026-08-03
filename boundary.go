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

// collectValueFuncs marks every package function the package refers to as a
// VALUE rather than calling — `var readDir dirReader = osReadDirNames`, or
// `generatedFiles{read: osReadHead}`.
//
// A function handed around as a value is an injectable collaborator by
// construction: something else in the package holds it in a variable or a
// field that a test can replace. The function is the real implementation
// behind that seam, and the seam is what the standard asks for.
func collectValueFuncs(pass *analysis.Pass, boundary exemptions) {
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			markValueUses(pass, file, calleeIdents(file), boundary)
		}
	}
}

// calleeIdents is the set of identifiers naming the callee of a call, which
// are uses in call position rather than references to a function's value.
func calleeIdents(file *ast.File) map[*ast.Ident]bool {
	called := map[*ast.Ident]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			markCallee(called, call.Fun)
		}
		return true
	})
	return called
}

// markCallee records the identifier a callee expression names, if it names one.
func markCallee(called map[*ast.Ident]bool, fun ast.Expr) {
	switch callee := ast.Unparen(fun).(type) {
	case *ast.Ident:
		called[callee] = true
	case *ast.SelectorExpr:
		called[callee.Sel] = true
	}
}

// markValueUses marks the package functions this file names outside call
// position.
func markValueUses(pass *analysis.Pass, file *ast.File, called map[*ast.Ident]bool, boundary exemptions) {
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && !called[ident] {
			markPackageFunc(pass, ident, boundary)
		}
		return true
	})
}

// markPackageFunc marks ident when it resolves to a function at all. A
// function from another package lands in the set inertly: the set is consulted
// for the declarations of this package alone, so a foreign entry sits unread.
func markPackageFunc(pass *analysis.Pass, ident *ast.Ident, boundary exemptions) {
	if fn, ok := pass.TypesInfo.Uses[ident].(*types.Func); ok {
		boundary[fn] = true
	}
}

// collectInterfaceMethods marks every method that implements a method of an
// interface the package itself declares.
//
// That interface IS the injected abstraction the standard asks for, and the
// method is its real implementation: `type Clock interface { Now() time.Time }`
// alongside `func (System) Now() time.Time { return time.Now() }` is the
// pattern, not a defect. Only the methods the interface names are exempt — a
// second method on the same type, outside the interface, is ordinary code.
func collectInterfaceMethods(pass *analysis.Pass, boundary exemptions) {
	scope := pass.Pkg.Scope()
	for _, iface := range declaredInterfaces(scope) {
		markImplementations(pass.Pkg, scope, iface, boundary)
	}
}

// declaredInterfaces is every interface type the package declares.
//
// The empty interface needs no special case. Every type satisfies it, but an
// exemption is granted per interface METHOD, and an empty interface names
// none — so it exempts nothing, and excluding it would be a guard with no
// effect either way.
func declaredInterfaces(scope *types.Scope) []*types.Interface {
	var declared []*types.Interface
	for _, name := range scope.Names() {
		if iface, ok := namedInterface(scope.Lookup(name)); ok {
			declared = append(declared, iface)
		}
	}
	return declared
}

// namedInterface is the interface a type name denotes, if it denotes one.
func namedInterface(obj types.Object) (*types.Interface, bool) {
	named, ok := obj.(*types.TypeName)
	if !ok {
		return nil, false
	}
	iface, ok := types.Unalias(named.Type()).Underlying().(*types.Interface)
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
