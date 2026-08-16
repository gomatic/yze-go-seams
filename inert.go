// The bindings that reach nothing, which is what the value-function exemption
// is written as its complement of. Three of them: the blank identifier, a
// local variable, and an argument at a parameter that is not a function type.
//
// A binding is judged by where the value LANDS, so the walk carries a binding
// through the spellings that store nothing of their own — parentheses, an
// address-of, a conversion, and a composite literal's elements.

package seams

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// inertIdents is the set of identifiers referenced only where a test can
// substitute nothing: bound to a blank identifier, bound to a local variable,
// or handed to a call at a parameter that is not a function type.
func inertIdents(pass *analysis.Pass, file *ast.File) map[*ast.Ident]bool {
	inert := map[*ast.Ident]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch at := n.(type) {
		case *ast.ValueSpec:
			markInertSpec(pass, at, inert)
		case *ast.AssignStmt:
			markInertTargets(pass, at.Lhs, at.Rhs, inert)
		case *ast.CallExpr:
			markLooseArgs(pass, at, inert)
		}
		return true
	})
	return inert
}

// markInertSpec marks a declaration's inert positions. A declaration whose
// type annotation denotes a FUNCTION type is exempt from the blank rule: it is
// the compiler-checked seam-shape assertion the exemption honors.
func markInertSpec(pass *analysis.Pass, spec *ast.ValueSpec, inert map[*ast.Ident]bool) {
	if !funcTypeAnnotation(pass, spec.Type) {
		markInertTargets(pass, identExprs(spec.Names), spec.Values, inert)
	}
}

// identExprs widens declared names to expressions, so declarations and
// assignments share one target walk.
func identExprs(names []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(names))
	for at, name := range names {
		exprs[at] = name
	}
	return exprs
}

// markInertTargets marks the value at each position whose target binds nothing
// a test can reach. Positions are judged one by one, so a named sibling in the
// same declaration still holds; an unaligned binding takes several values from
// one expression and names no function at any position, so it marks nothing.
func markInertTargets(pass *analysis.Pass, targets, values []ast.Expr, inert map[*ast.Ident]bool) {
	if len(targets) != len(values) {
		return
	}
	for at, target := range targets {
		if isInertTarget(pass, target) {
			markHeld(pass, values[at], inert)
		}
	}
}

// isInertTarget reports a binding no test substitutes: the blank
// identifier, which holds nothing at all, and a variable whose extent is the
// function that declared it. Every other target — a package-level variable, a
// field, a map or slice element, a pointer's referent — is somewhere a test
// can arrange, so it is a hold.
func isInertTarget(pass *analysis.Pass, target ast.Expr) bool {
	ident, ok := target.(*ast.Ident)
	if !ok {
		return false
	}
	if isBlank(ident) {
		return true
	}
	at, ok := boundVar(pass, ident)
	return ok && at.Parent() != pass.Pkg.Scope()
}

// boundVar is the variable an assignment or declaration target names, whether
// the target declares it or writes to one already declared.
func boundVar(pass *analysis.Pass, ident *ast.Ident) (*types.Var, bool) {
	if at, ok := pass.TypesInfo.Defs[ident].(*types.Var); ok {
		return at, true
	}
	at, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	return at, ok
}

// isBlank reports the blank identifier, which holds nothing.
func isBlank(name *ast.Ident) bool {
	return name.Name == "_"
}

// markLooseArgs marks each argument a call binds to a parameter that is NOT a
// function type. `fmt.Sprint(spawn)` takes it as `any`, which every value
// satisfies, so the position says as little about the function as `var _ any =
// fn` does — while a function-typed parameter is the compiler-checked
// injection the rule exists to encourage.
func markLooseArgs(pass *analysis.Pass, call *ast.CallExpr, inert map[*ast.Ident]bool) {
	if isConversion(pass, call) {
		return
	}
	sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
	if !ok {
		return
	}
	for at, arg := range call.Args {
		if !isFuncParam(sig, argPosition(at)) {
			markHeld(pass, arg, inert)
		}
	}
}

// isConversion reports a call whose callee is a TYPE rather than a function.
// A conversion binds nothing of its own — it renames a value and hands it
// straight on — so whatever encloses it decides, and it is TRANSPARENT to the
// walk in both directions.
//
// This is not a corner: when a seam's declared type is an interface, the
// conversion is the ONLY spelling available. `var runCommand runner =
// runFunc(execRun)`, the http.HandlerFunc idiom, has no other form, so reading
// the conversion as a binding of its own would report a package written the
// only way it can be written.
func isConversion(pass *analysis.Pass, call *ast.CallExpr) bool {
	return pass.TypesInfo.Types[call.Fun].IsType()
}

// argPosition is an argument's index in a call's argument list.
type argPosition int

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

// tailType is what ONE argument of a variadic tail binds: the slice's element
// type rather than the slice the signature declares.
func tailType(sig *types.Signature, param types.Type) types.Type {
	slice, ok := param.(*types.Slice)
	if sig.Variadic() && ok {
		return slice.Elem()
	}
	return param
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

// markHeld records every function name one bound expression carries down to
// the position the binding actually reaches.
//
// A plain name, a method or another package's function through a selector, and
// an explicit generic instantiation each name a function. Parentheses, an
// address-of and a CONVERSION are transparent, because none of them stores
// anything. A composite literal is transparent to its elements: `var _ =
// []runner{spawn}` binds the SLICE to a blank, so nothing holds spawn, while
// the same literal bound to a name holds every element in it. An ordinary
// call names its result rather than a function, so it records nothing.
func markHeld(pass *analysis.Pass, value ast.Expr, inert map[*ast.Ident]bool) {
	switch at := value.(type) {
	case *ast.Ident:
		inert[at] = true
	case *ast.SelectorExpr:
		inert[at.Sel] = true
	case *ast.IndexExpr:
		markHeld(pass, at.X, inert)
	case *ast.IndexListExpr:
		markHeld(pass, at.X, inert)
	case *ast.ParenExpr:
		markHeld(pass, at.X, inert)
	case *ast.UnaryExpr:
		markHeld(pass, at.X, inert)
	case *ast.CompositeLit:
		markHeldElements(pass, at.Elts, inert)
	case *ast.CallExpr:
		markConverted(pass, at, inert)
	}
}

// markHeldElements carries a binding through a composite literal's elements,
// keyed or not.
func markHeldElements(pass *analysis.Pass, elts []ast.Expr, inert map[*ast.Ident]bool) {
	for _, elt := range elts {
		if pair, ok := elt.(*ast.KeyValueExpr); ok {
			markHeld(pass, pair.Value, inert)
			continue
		}
		markHeld(pass, elt, inert)
	}
}

// markConverted carries a binding through a conversion to what was converted.
// An ordinary call is where the carrying stops: its result is a new value that
// names no function.
//
// The operands are walked rather than indexed. A conversion that compiles
// carries one operand and no more, so an arity guard here would be a condition
// no input could fail — and a guard nothing can fail on is dead code rather
// than a missing case.
func markConverted(pass *analysis.Pass, call *ast.CallExpr, inert map[*ast.Ident]bool) {
	if isConversion(pass, call) {
		markHeldElements(pass, call.Args, inert)
	}
}
