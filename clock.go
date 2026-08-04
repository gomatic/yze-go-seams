package seams

import (
	"go/ast"
)

// clockSymbol is the one impure symbol whose findings depend on how the value
// is USED rather than merely that it was called.
// It is deliberately UNTYPED: as a typed member of symbol it would read as
// part of that enum, and the exhaustive check would then demand it as a key in
// every symbol-keyed map — including impureFuncs, where adding it would change
// what the analyzer reports.
const clockSymbol = "time.Now"

// comparisonMethods are the time.Time methods that turn a timestamp into a
// branch. A test cannot steer them at a direct call site, because it cannot
// choose what the clock returns.
var comparisonMethods = map[string]bool{
	"After":   true,
	"Before":  true,
	"Equal":   true,
	"Compare": true,
	"Sub":     true,
}

// branchingClocks collects the time.Now calls beneath node whose value reaches
// a comparison — the only ones that put a branch out of a test's reach.
//
// Stamping is not branching, and conflating them was this rule's largest
// remaining source of noise: 24 of its 37 findings were a timestamp written
// into a field, a lock file, or a log line, where nothing downstream tests the
// value and there is no unreachable branch to complain about. Demanding an
// injected clock for those buys a constructor parameter and no coverage.
//
// What survives is the shape the rule was written for — `time.Now().After(deadline)`,
// `time.Now().Sub(start) > ttl` — where the branch genuinely cannot be reached
// without control of the clock.
func branchingClocks(node ast.Node) map[*ast.CallExpr]bool {
	branching := map[*ast.CallExpr]bool{}
	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.BinaryExpr:
			if comparisonOps[expr.Op.String()] {
				markClocks(expr, branching)
			}
		case *ast.CallExpr:
			markComparisonReceiver(expr, branching)
		}
		return true
	})
	return branching
}

// comparisonOps are the operators that form a comparison, keyed by the
// operator's own text.
//
// Keyed by text rather than by token.Token deliberately. token.Token is an
// enum spanning every operator AND keyword in the language, so both a switch
// and a map over it read as an incomplete enumeration — the exhaustiveness
// check demands all ~80 members, and the six that matter would be lost among
// the keywords that can never appear here.
var comparisonOps = map[string]bool{
	"<":  true,
	">":  true,
	"<=": true,
	">=": true,
	"==": true,
	"!=": true,
}

// markComparisonReceiver marks the clocks in a call to a time comparison
// method, on either side: the receiver (`time.Now().After(x)`) and the
// argument (`deadline.Before(time.Now())`) both make the branch clock-bound.
func markComparisonReceiver(call *ast.CallExpr, branching map[*ast.CallExpr]bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !comparisonMethods[selector.Sel.Name] {
		return
	}
	markClocks(selector.X, branching)
	for _, arg := range call.Args {
		markClocks(arg, branching)
	}
}

// markClocks records every call beneath expr, leaving the symbol check to the
// caller — reportCall already resolves whether a call is time.Now, and
// duplicating that resolution here would be a second place to get it wrong.
func markClocks(expr ast.Node, branching map[*ast.CallExpr]bool) {
	ast.Inspect(expr, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			branching[call] = true
		}
		return true
	})
}
