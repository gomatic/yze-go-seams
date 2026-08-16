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

// collectValueFuncs marks every package function the package HOLDS in
// something a test can substitute — `var readDir dirReader = osReadDirNames`,
// or `generatedFiles{read: osReadHead}`.
//
// A function held that way is an injectable collaborator by construction: the
// hold is the seam a test replaces, and the function is the real
// implementation behind it.
//
// The hold has to be one a test can actually replace, and that is the whole
// content of the exemption — a reference that binds the function nowhere
// proves no seam and marks nothing. Three positions qualify, and they are the
// three a test can write over: the value of a package-level var declaration,
// an element or field value of a composite literal, and the right-hand side of
// an assignment to a field.
//
// A fourth position qualifies for the same reason the typed blank does, and it
// is the constructor injection the standard actually asks for: an argument at a
// parameter whose declared type is a FUNCTION type. `New(git.run, git.exists)`
// hands the collaborators to something that holds them, and a test calls the
// same constructor with fakes. The parameter's type is what carries the
// evidence, no differently from an annotation — an argument at an `any` parameter
// (`fmt.Sprint(spawn)`) binds nothing about the function and marks nothing.
//
// Deliberately NOT holds, because a test can substitute none of them: a local
// capture (`f := spawn`, `f := r.Spawn`) and a return value. A local lives and
// dies inside the function that made it, and a returned value is read by
// whoever already chose to call this code. A package that wants the exemption
// has the one-line honest spelling available — bind the function to a
// package-level var — so the conforming shape stays cheaper than the evading
// one.
//
// A bare `var _ = fn` is not a hold either: the blank identifier gives a test
// nothing to replace. The TYPED form `var _ Command = ExecCommand` is
// different in kind — it is the package asserting, checked by the compiler,
// that the function backs a declared seam type a composition root binds — and
// it marks.
func collectValueFuncs(pass *analysis.Pass, boundary exemptions) {
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			markHolds(pass, heldExprs(pass, file), boundary)
		}
	}
}

// markHolds marks the package function each held expression names.
func markHolds(pass *analysis.Pass, held []ast.Expr, boundary exemptions) {
	for _, expr := range held {
		if ident, ok := heldIdent(expr); ok {
			markPackageFunc(pass, ident, boundary)
		}
	}
}

// heldIdent is the identifier a held expression names, when it names one: a
// plain function (`execRun`), a method or another package's function
// (`s.Spawn`, `os.ReadFile`), or an explicit generic instantiation
// (`spawn[int]`). Anything else — a call, a literal, a conversion — binds no
// function name and marks nothing.
func heldIdent(expr ast.Expr) (*ast.Ident, bool) {
	switch at := expr.(type) {
	case *ast.Ident:
		return at, true
	case *ast.SelectorExpr:
		return at.Sel, true
	case *ast.IndexExpr:
		return heldIdent(at.X)
	case *ast.IndexListExpr:
		return heldIdent(at.X)
	}
	return nil, false
}

// heldExprs is every expression this file binds into a hold a test can
// substitute: the values of its package-level declarations, and the composite
// literals and field assignments anywhere beneath them.
func heldExprs(pass *analysis.Pass, file *ast.File) []ast.Expr {
	held := declaredHolds(pass, file)
	ast.Inspect(file, func(n ast.Node) bool {
		held = append(held, boundHolds(pass, n)...)
		return true
	})
	return held
}

// declaredHolds is every value a package-level declaration binds to a name a
// test can rebind.
func declaredHolds(pass *analysis.Pass, file *ast.File) []ast.Expr {
	held := []ast.Expr{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		held = append(held, specHolds(pass, gen)...)
	}
	return held
}

// specHolds is every named position's value across a declaration's specs.
// Aligned lists are judged position by position, and a multi-value binding
// from one expression is not a hold at all, since no single name holds the
// function rather than the call's result.
func specHolds(pass *analysis.Pass, gen *ast.GenDecl) []ast.Expr {
	held := []ast.Expr{}
	for _, spec := range gen.Specs {
		at, ok := spec.(*ast.ValueSpec)
		if !ok || len(at.Names) != len(at.Values) {
			continue
		}
		held = append(held, namedValues(pass, at)...)
	}
	return held
}

// namedValues is the values of a spec's non-blank positions, plus a blank
// position whose annotation denotes a FUNCTION type — `var _ Command =
// ExecCommand` is the compiler-checked seam-shape assertion the exemption
// honors, while `var _ any = fn` checks nothing, since every value satisfies
// any.
func namedValues(pass *analysis.Pass, spec *ast.ValueSpec) []ast.Expr {
	held := []ast.Expr{}
	for at, name := range spec.Names {
		if !isBlank(name) || funcTypeAnnotation(pass, spec.Type) {
			held = append(held, spec.Values[at])
		}
	}
	return held
}

// isBlank reports the blank identifier, which holds nothing.
func isBlank(name *ast.Ident) bool {
	return name.Name == "_"
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
	return isSignature(at)
}

// isSignature reports a type that denotes a function, through however many
// names it was declared behind.
func isSignature(at types.Type) bool {
	_, ok := at.Underlying().(*types.Signature)
	return ok
}

// boundHolds is every value a node binds into a hold a test can write over,
// wherever the node sits.
func boundHolds(pass *analysis.Pass, n ast.Node) []ast.Expr {
	switch at := n.(type) {
	case *ast.CompositeLit:
		return literalHolds(at)
	case *ast.AssignStmt:
		return fieldHolds(at)
	case *ast.CallExpr:
		return argumentHolds(pass, at)
	}
	return nil
}

// argumentHolds is every argument a call binds to a parameter declared with a
// FUNCTION type. That is the constructor injection the rule exists to
// encourage — the callee holds what it was handed, and a test calls it with
// something else — and the parameter's type is the compiler-checked evidence,
// the same kind `var _ Command = ExecCommand` carries. An argument at a
// parameter of any other type says nothing about the function it was handed.
//
// What the callee then DOES with the value is not traced, which is the same
// bound every other hold carries: nothing checks that a package-level seam var
// is ever called either. The evidence is the binding, not the use.
func argumentHolds(pass *analysis.Pass, call *ast.CallExpr) []ast.Expr {
	sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
	if !ok {
		return nil
	}
	held := []ast.Expr{}
	for at, arg := range call.Args {
		if isFuncParam(sig, argPosition(at)) {
			held = append(held, arg)
		}
	}
	return held
}

// isFuncParam reports whether the parameter one argument position binds is
// declared with a function type, counting a variadic tail as the element type
// each of its arguments binds. A conversion to a niladic function type is a
// call carrying an argument and declaring no parameter, and binds nothing.
func isFuncParam(sig *types.Signature, at argPosition) bool {
	params := sig.Params()
	last := argPosition(params.Len() - 1)
	if last < 0 {
		return false
	}
	if at < last {
		return isSignature(params.At(int(at)).Type())
	}
	return isSignature(tailType(sig, params.At(int(last)).Type()))
}

// argPosition is an argument's index in a call's argument list.
type argPosition int

// tailType is what ONE argument of a variadic tail binds: the slice's element
// type rather than the slice the signature declares.
func tailType(sig *types.Signature, param types.Type) types.Type {
	slice, ok := param.(*types.Slice)
	if sig.Variadic() && ok {
		return slice.Elem()
	}
	return param
}

// literalHolds is every element a composite literal binds — a struct field, a
// map value, a slice or array element. Each is a hold, because a test can
// build the same literal with something else in that position.
func literalHolds(lit *ast.CompositeLit) []ast.Expr {
	held := make([]ast.Expr, 0, len(lit.Elts))
	for _, elt := range lit.Elts {
		if pair, ok := elt.(*ast.KeyValueExpr); ok {
			held = append(held, pair.Value)
			continue
		}
		held = append(held, elt)
	}
	return held
}

// fieldHolds is every value an assignment writes into a FIELD — `s.run =
// execRun`. A field is a hold a test can substitute; a local variable, which
// lives only inside the function that declared it, is not.
func fieldHolds(assign *ast.AssignStmt) []ast.Expr {
	if len(assign.Lhs) != len(assign.Rhs) {
		return nil
	}
	held := []ast.Expr{}
	for at, target := range assign.Lhs {
		if _, ok := target.(*ast.SelectorExpr); ok {
			held = append(held, assign.Rhs[at])
		}
	}
	return held
}

// markPackageFunc marks ident when it resolves to a function at all. A
// function from another package lands in the set inertly: the set is consulted
// for the declarations of this package alone, so a foreign entry sits unread.
func markPackageFunc(pass *analysis.Pass, ident *ast.Ident, boundary exemptions) {
	if fn, ok := pass.TypesInfo.Uses[ident].(*types.Func); ok {
		boundary[fn] = true
	}
}
