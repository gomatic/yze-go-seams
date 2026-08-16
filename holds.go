// What makes a function-value reference an exemption: the hold behind it. A
// function the package binds somewhere a test can write over is the real
// implementation behind a seam that already exists; a reference that binds it
// nowhere reachable proves nothing.
//
// This is a separate question from the interface half in boundary.go, which
// asks the same thing of a declared abstraction.

package seams

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// collectValueFuncs marks every package function the package refers to as a
// VALUE rather than calling — `var readDir dirReader = osReadDirNames`, or
// `generatedFiles{read: osReadHead}`.
//
// A function handed around as a value is an injectable collaborator by
// construction: something else in the package holds it in a variable or a
// field that a test can replace. The function is the real implementation
// behind that seam, and the seam is what the standard asks for.
//
// The exemption is written as the reference MINUS the positions that hold
// nothing, rather than as a list of the positions that do, and that direction
// is the load-bearing choice. Enumerating the good positions was tried and
// measured wrong three times over: a constructor argument (`New(git.run,
// git.exists)`), a package-level var bound in an `init`, and a registry entry
// (`handlers["run"] = execRun`) are all real seams a test substitutes, and an
// enumeration naming a package var, a composite-literal field and a field
// assignment reported every one of them. A storage location a test can arrange
// is not an enumerable set; a binding no test reaches is.
//
// Three bind nothing a test can reach, and each is subtracted:
//
//   - The blank identifier. `var _ = fn` and `_ = fn` give a test nothing to
//     replace. The TYPED form `var _ Command = ExecCommand` is different in
//     kind — it is the package asserting, checked by the compiler, that the
//     function backs a declared seam type a composition root binds — and it
//     marks, while `var _ any = fn` checks nothing and stays inert.
//   - A LOCAL variable. `f := spawn`, `f := r.Spawn`, and `local = execRun`
//     inside a function bind the value to something whose extent is the
//     function that declared it, so no test can substitute it. This is the
//     forgery the exemption was widest to: two tokens silenced any body.
//   - An argument at a parameter whose type is NOT a function type.
//     `fmt.Sprint(spawn)` takes it as `any`, which every value satisfies, so
//     the position asserts as little as `var _ any = fn` does. An argument at a
//     function-typed parameter is the opposite — it is the constructor
//     injection this rule exists to encourage, compiler-checked by the
//     parameter's own type.
//
// What the holder then DOES with the value is not traced, which is the bound
// the whole exemption carries: nothing checks that a package-level seam var is
// ever called either. The evidence is the binding, not the use.
func collectValueFuncs(pass *analysis.Pass, boundary exemptions) {
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			markValueUses(pass, file, calleeIdents(file), inertIdents(pass, file), boundary)
		}
	}
}

// markValueUses marks the package functions this file names outside call
// position, excluding the references that bind nothing a test can reach.
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
//
// An explicit generic instantiation is unwrapped first: `spawn[int](name)` is
// a CALL to spawn whatever its type arguments, and reading the identifier
// underneath it as a value reference instead is a forgery an author can
// perform in one keystroke — adding an unused type parameter would silence
// every impure call in a function's body.
func markCallee(called map[*ast.Ident]bool, fun ast.Expr) {
	switch callee := calleeExpr(fun).(type) {
	case *ast.Ident:
		called[callee] = true
	case *ast.SelectorExpr:
		called[callee.Sel] = true
	}
}

// calleeExpr strips the wrappers that leave a callee the same callee:
// parentheses, and the type arguments of an explicit instantiation.
func calleeExpr(fun ast.Expr) ast.Expr {
	switch at := ast.Unparen(fun).(type) {
	case *ast.IndexExpr:
		return calleeExpr(at.X)
	case *ast.IndexListExpr:
		return calleeExpr(at.X)
	default:
		return at
	}
}
