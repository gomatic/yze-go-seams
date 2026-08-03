// Which code the rule judges at all: whole packages that are out of scope, and
// the files within a package that are.
//
// This is a separate question from what a call reaches for. A package can be
// exempt because of what it IS — the composition root, or test support — before
// any call in it is classified, and both exemptions are decided once per pass.

package seams

import (
	"go/ast"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// packagePath is the import path of the package under analysis.
type packagePath string

// packageName is the name in the package clause of the package under analysis.
type packageName string

// isExempt reports whether the whole package is out of scope, in either of the
// two ways a package can be: it is the composition root, where the real world
// is meant to be wired in, or it is test support, which only ever runs under
// `go test`.
func isExempt(pass *analysis.Pass) bool {
	return isCompositionRoot(packagePath(pass.Pkg.Path()), packageName(pass.Pkg.Name())) ||
		isTestSupport(pass)
}

// isCompositionRoot reports whether the package is the place the real world is
// meant to be wired in: a `main` package, or anything beneath a `cmd` element
// of an import path. Those packages exist precisely to hand real collaborators
// to the packages that took them as parameters.
func isCompositionRoot(at packagePath, name packageName) bool {
	return name == "main" || slices.Contains(strings.Split(string(at), "/"), "cmd")
}

// testingImport is the testing package's import path as an import spec spells
// it, quotes included.
const testingImport = `"testing"`

// isTestSupport reports whether the package exists only to serve tests,
// recognised by a NON-test file importing `testing`.
//
// The property is a proof rather than a heuristic, which is why it is the one
// consulted. Importing `testing` outside a _test.go file links the framework
// into every binary that transitively imports the package — registering the
// whole `-test.*` flag set on it — so production code does not do it, and a
// package that does is compiled for `go test` and nothing else. Reaching for
// the real clock or the real filesystem there is the same as reaching for one
// inside a _test.go file, and is exempt for the same reason.
//
// The package's NAME is deliberately not consulted. `pgtest`, `lanetest`, and
// `registrytest` are test support and `latest` and `protest` are ordinary
// words, and no suffix rule can tell those two groups apart — while `testutil`
// and `harness` are test support carrying no `test` suffix at all. The import
// decides all five correctly.
//
// Only a file of THIS package is consulted, and never a _test.go file: a test
// importing `testing` is every tested package in existence, and honouring that
// would exempt the fleet.
func isTestSupport(pass *analysis.Pass) bool {
	for _, file := range pass.Files {
		if !isTestFile(pass, file) && importsTesting(file) {
			return true
		}
	}
	return false
}

// importsTesting reports whether the file imports the testing package itself.
// A subpackage — `testing/fstest`, `testing/quick` — is a different import path
// and does not count, since those carry no flag registration and appear in
// ordinary code.
func importsTesting(file *ast.File) bool {
	return slices.ContainsFunc(file.Imports, func(spec *ast.ImportSpec) bool {
		return spec.Path.Value == testingImport
	})
}

// isTestFile reports whether file is a _test.go file, where reaching for a real
// resource is the test's own business.
func isTestFile(pass *analysis.Pass, file *ast.File) bool {
	return strings.HasSuffix(pass.Fset.File(file.Pos()).Name(), "_test.go")
}
