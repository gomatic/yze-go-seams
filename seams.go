// Package seams provides a go/analysis analyzer enforcing the gomatic
// dependency-injection standard at authorship: outside a composition root, a
// package must not call an impure stdlib entry point — the clock, the global
// random source, the filesystem, the network, a subprocess — directly.
//
// The standard's reasoning is testability, not purity: "Dependency injection
// everywhere. Functions and constructors accept their collaborators
// (filesystem, clock, readers, writers) as parameters or interfaces. Never
// reach for a global or a real OS resource where an injected abstraction
// works. This is what makes 100% coverage reachable." A direct `os.ReadFile`
// call site has an error branch no test can enter, so the 100%-coverage gate
// catches it — but only at the end, after the design is written. This rule
// catches the same defect at the call site.
//
// # The conforming forms, all silent by construction
//
//   - An injected collaborator. `d.readFile(path)`, `fs.Open(name)` — the
//     qualifier is a value, not an imported package, so nothing is reported.
//   - A package-level seam variable, which the standard blesses explicitly as
//     the correct Go answer where a stdlib call has no other seam. `var
//     readFile = os.ReadFile` is a reference, never a call, so the declaration
//     is silent; and `readFile(path)` at the use site is a call through an
//     identifier, not through a package, so it is silent too.
//   - The same seam written as a closure, because the seam's signature differs
//     from the stdlib's: `var createTemp = func(dir, pattern string) (tempFile,
//     error) { return os.CreateTemp(dir, pattern) }`. A function literal in a
//     package-level var initializer is a seam declaration, not a call site.
//   - The real implementation BEHIND a seam. Every dependency-injected design
//     bottoms out in one place that touches the world, and that place is
//     ordinary Go. Two shapes are recognised: a function the package hands
//     around as a value (`var readDir dirReader = osReadDirNames`), and a
//     method implementing an interface the package declares (`type Clock
//     interface{ Now() time.Time }` beside `func (System) Now() time.Time`).
//     Both mean the seam already exists one level up.
//   - Passing the function itself. `New(os.ReadFile)` is the injection this
//     rule exists to encourage; only a CALL is reported.
//
// # Deliberate boundaries
//
// A rule that fires on legitimate code is worse than no rule, so every case
// this analyzer cannot decide reliably resolves to silence:
//
//   - Composition roots are exempt wholesale. A `main` package, or any package
//     whose import path contains a `cmd` element, is where the real world is
//     supposed to be wired in.
//   - Test files are out of scope: reaching for a real resource in a test is
//     the test's business. A test-SUPPORT package is out of scope for the same
//     reason, recognised by the one property that proves it — a NON-test file
//     importing `testing`. `internal/pgtest`, `tests/harness`, and
//     `internal/testutil` all carry it, and production code cannot: importing
//     `testing` outside a _test.go file links the framework's `-test.*` flag
//     registration into every binary that transitively imports the package.
//     The package NAME is deliberately not consulted, because `pgtest` and
//     `latest` are the same string shape and only one of them is a word.
//   - Only genuinely impure entry points are listed. `time.Since` and duration
//     arithmetic are pure functions of their arguments; `rand.New(rand.NewPCG(…))`
//     builds a generator from an explicit, reproducible source. Neither is
//     reported — only the functions that read a hidden global or the OS.
//   - A dot-imported package (`. "os"`) makes `ReadFile(path)` indistinguishable
//     from a local seam at the call site, so it is not reported.
//   - `os.Getenv` and friends are not listed. Process environment is read once
//     at the edge often enough that flagging it would report configuration
//     plumbing far more often than a testability defect.
//   - An adapter that satisfies an interface from ANOTHER package is not
//     recognised as one. Whether an exported type is someone else's
//     collaborator cannot be decided from this package's syntax, so such a
//     method is reported like any other.
//   - Pushing the impurity to a thin public wrapper — `func Generate(…) { return
//     generateAt(…, time.Now()) }` over a fully-tested `generateAt` — is the
//     right design and is still reported, because a one-line wrapper is not
//     distinguishable from a domain function that reaches for the clock.
package seams

import (
	"go/ast"
	"go/types"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
)

const message = "%s is called directly, so the branches around it cannot be reached from a test; " +
	"accept the collaborator as a parameter or bind a package-level seam (var … = %s)"

// Analyzer reports direct calls to impure stdlib entry points outside a composition root.
var Analyzer = &analysis.Analyzer{
	Name: "seams",
	Doc: "reports a direct call to an impure stdlib entry point (clock, global random source, " +
		"filesystem, network, subprocess) outside a composition root, where an injected " +
		"collaborator or a package-level seam variable is required instead",
	Run: run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "seams",
	Categories: []goyze.Category{"tests", "patterns"},
	URL:        "https://docs.gomatic.dev/yze/seams",
	Analyzer:   Analyzer,
}

// run reports every direct call to an impure stdlib entry point in the
// package's non-test files, unless the package is out of scope wholesale or the
// call sits inside a declared boundary.
func run(pass *analysis.Pass) (any, error) {
	if isExempt(pass) {
		return nil, nil
	}
	boundary := adapters(pass)
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			checkFile(pass, boundary, file)
		}
	}
	return nil, nil
}

// checkFile reports each impure call in one file, declaration by declaration
// so a declaration can be skipped whole.
func checkFile(pass *analysis.Pass, boundary exemptions, file *ast.File) {
	for _, decl := range file.Decls {
		checkDecl(pass, boundary, decl)
	}
}

// checkDecl walks one top-level declaration.
//
// A function or method the package hands around as a value, or one that
// implements an interface the package declares, is skipped whole: it is the
// real implementation behind a seam that already exists.
//
// Everything else is walked, with one exception that carries the standard's
// blessing. A function literal inside a package-level var initializer — `var
// createTemp = func(dir, pattern string) (tempFile, error) { return
// os.CreateTemp(dir, pattern) }` — is the seam ITSELF, written as a closure
// because the seam's signature differs from the stdlib's. It is the same
// declaration as `var readFile = os.ReadFile` and is silent for the same
// reason. A direct call in the initializer that is NOT inside a literal —
// `var started = time.Now()` — captures the real world into package state at
// init and is reported.
func checkDecl(pass *analysis.Pass, boundary exemptions, decl ast.Decl) {
	if fn, ok := decl.(*ast.FuncDecl); ok {
		if !boundary[pass.TypesInfo.ObjectOf(fn.Name)] {
			inspectCalls(pass, fn, intoLiterals)
		}
		return
	}
	inspectCalls(pass, decl, pastLiterals)
}

// literalPolicy decides whether a function literal's body is walked.
type literalPolicy string

// The two literal policies.
const (
	// intoLiterals walks a closure's body, which is ordinary code.
	intoLiterals literalPolicy = "into"
	// pastLiterals leaves a closure's body alone, because the closure is a
	// seam declaration rather than a call site.
	pastLiterals literalPolicy = "past"
)

// inspectCalls reports each impure call beneath node.
func inspectCalls(pass *analysis.Pass, node ast.Node, policy literalPolicy) {
	branching := branchingClocks(node)
	ast.Inspect(node, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok && policy == pastLiterals {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			reportCall(pass, call, branching)
		}
		return true
	})
}

// reportCall anchors a finding at the callee — the text a reader would replace
// with a parameter or a seam.
func reportCall(pass *analysis.Pass, call *ast.CallExpr, branching map[*ast.CallExpr]bool) {
	at, ok := reached(pass.TypesInfo, call.Fun)
	if !ok || (at == clockSymbol && !branching[call]) {
		return
	}
	pass.Reportf(call.Fun.Pos(), message, string(at), string(at))
}

// reached names the impure stdlib symbol a callee reaches for, in either of the
// two shapes that bypass injection: a call on the package itself
// (`os.ReadFile(…)`), and a call on the package's own global (`http.DefaultClient.Do(…)`).
//
// Every other callee shape is silent, and that is the whole point: a call
// through an identifier is a package-level seam or a local, and a call through
// a value's field or method is an injected collaborator.
func reached(info *types.Info, fun ast.Expr) (symbol, bool) {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if at, named := qualify(info, sel); named && impureFuncs[at] {
		return at, true
	}
	return viaGlobal(info, sel)
}

// viaGlobal names the impure stdlib global a method call is made on, when the
// callee is one selector deeper: `http.DefaultClient.Do` reads DefaultClient
// off net/http and calls a method on it.
func viaGlobal(info *types.Info, sel *ast.SelectorExpr) (symbol, bool) {
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	at, named := qualify(info, inner)
	if !named || !impureGlobals[at] {
		return "", false
	}
	return at, true
}

// qualify names the stdlib symbol a selector reads, and reports false when the
// qualifier is a value rather than an imported package.
//
// The package comes from the type checker's resolution of the qualifier, never
// from the identifier's spelling, so an aliased import, a shadowing local
// named `os`, and the two distinct packages that both call themselves `rand`
// are each resolved to what they actually are.
func qualify(info *types.Info, sel *ast.SelectorExpr) (symbol, bool) {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	imported, ok := info.Uses[ident].(*types.PkgName)
	if !ok {
		return "", false
	}
	return symbol(imported.Imported().Path() + "." + sel.Sel.Name), true
}
