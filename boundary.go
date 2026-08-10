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
//
// A bare `var _ = fn` is the one value reference that holds nothing: the blank
// identifier gives a test nothing to replace, so it proves no seam and marks
// nothing. The TYPED form `var _ Command = ExecCommand` is different in kind —
// it is the package asserting, checked by the compiler, that the function
// backs a declared seam type a composition root binds — and it marks.
func collectValueFuncs(pass *analysis.Pass, boundary exemptions) {
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			markValueUses(pass, file, calleeIdents(file), inertIdents(pass, file), boundary)
		}
	}
}

// inertIdents is the set of identifiers referenced only as a hold nothing can
// replace: the values at BLANK positions of assignments and var declarations,
// in either spelling — `var _ = fn` and the statement form `_ = fn` alike.
// The one exception is a declaration whose type annotation denotes a FUNCTION
// type: `var _ Command = ExecCommand` is the package asserting, checked by the
// compiler, that the function backs a declared seam shape — while `var _ any =
// fn` checks nothing, since every value satisfies any, and stays inert.
func inertIdents(pass *analysis.Pass, file *ast.File) map[*ast.Ident]bool {
	inert := map[*ast.Ident]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch at := n.(type) {
		case *ast.ValueSpec:
			if !funcTypeAnnotation(pass, at.Type) {
				markBlankPositions(identExprs(at.Names), at.Values, inert)
			}
		case *ast.AssignStmt:
			markBlankPositions(at.Lhs, at.Rhs, inert)
		}
		return true
	})
	return inert
}

// funcTypeAnnotation reports a type annotation denoting a function type — the
// compiler-checked seam-shape assertion the exemption honors.
func funcTypeAnnotation(pass *analysis.Pass, annotation ast.Expr) bool {
	if annotation == nil {
		return false
	}
	at := pass.TypesInfo.TypeOf(annotation)
	if at == nil {
		return false
	}
	_, ok := at.Underlying().(*types.Signature)
	return ok
}

// identExprs widens declared names to expressions, so declarations and
// assignments share one blank-position walk.
func identExprs(names []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(names))
	for i, name := range names {
		exprs[i] = name
	}
	return exprs
}

// markBlankPositions marks the value identifiers held only by a blank target.
// Aligned lists are judged position by position; a multi-value binding from
// one expression is inert only when EVERY target is blank, since any named
// sibling holds the whole result.
func markBlankPositions(targets, values []ast.Expr, inert map[*ast.Ident]bool) {
	if len(targets) == len(values) {
		markAlignedBlanks(targets, values, inert)
		return
	}
	if !allBlankExprs(targets) {
		return
	}
	for _, value := range values {
		markIdentsUnder(value, inert)
	}
}

// markAlignedBlanks marks each blank position's value in an aligned binding.
func markAlignedBlanks(targets, values []ast.Expr, inert map[*ast.Ident]bool) {
	for i, target := range targets {
		if isBlank(target) {
			markIdentsUnder(values[i], inert)
		}
	}
}

// isBlank reports the blank identifier.
func isBlank(target ast.Expr) bool {
	ident, ok := target.(*ast.Ident)
	return ok && ident.Name == "_"
}

// allBlankExprs reports whether every target is the blank identifier.
func allBlankExprs(targets []ast.Expr) bool {
	for _, target := range targets {
		if !isBlank(target) {
			return false
		}
	}
	return len(targets) > 0
}

// markIdentsUnder records every identifier beneath one value expression.
func markIdentsUnder(value ast.Expr, inert map[*ast.Ident]bool) {
	ast.Inspect(value, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			inert[ident] = true
		}
		return true
	})
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
// position, excluding references an untyped blank var makes inertly.
func markValueUses(
	pass *analysis.Pass,
	file *ast.File,
	called map[*ast.Ident]bool,
	inert map[*ast.Ident]bool,
	boundary exemptions,
) {
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && !called[ident] && !inert[ident] {
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
