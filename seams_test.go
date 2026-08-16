package seams_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	seams "github.com/gomatic/yze-go-seams"
)

// TestADirectCallIsReported pins the rule: every impure stdlib entry point on
// the list — the clock, the global random source, OS entropy, the network, a
// subprocess — is reported at the callee when it is called through its
// package, in package state as well as in a function body.
//
// It pins the list's boundary in the same fixture: the filesystem is
// deliberately off the list and is silent there beside those reported
// siblings, and a clock reading inside a NON-comparison binary expression is a
// stamp rather than a branch. It pins the seam-literal boundary too — a
// literal bound to a blank, and one buried inside an initializer's expression,
// are ordinary code and are reported.
func TestADirectCallIsReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "direct")
}

// TestTheBlessedSeamIsSilent pins the form the standard explicitly blesses,
// which the rule must never report: `var readFile = os.ReadFile` is a
// reference and not a call, and `readFile(at)` at the use site goes through an
// identifier rather than a package. It pins the other conforming forms with
// it — an injected function field, an injected *http.Client, a collaborator
// whose method is spelled `Now`, and passing os.ReadFile to a constructor.
func TestTheBlessedSeamIsSilent(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "seam")
}

// TestPureSiblingsAreSilent pins the false positives the obvious
// implementation would produce. Time.Add is a pure function of its arguments;
// a RETURNED time.Since is a clock read that only stamps, silent under the
// branching narrowing (the branching spelling is reported in the direct and
// latest fixtures); and rand.New/rand.NewPCG build a generator from a seed the
// caller chose — reproducible, in need of no seam. They live in the same
// fixture as the conforming forms because a single silent package is the
// claim: nothing in it is reported.
func TestPureSiblingsAreSilent(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "seam")
}

// TestTheBoundaryBehindASeamIsSilent pins the shape every dependency-injected
// design bottoms out in — the one place that touches the world — in both forms
// the language offers: a function the package hands around as a value, and a
// method implementing an interface the package declares. It pins the two
// boundaries of that exemption with them: a method on the same type that the
// interface does NOT name is still reported, and an empty interface exempts
// nothing (every type satisfies one, so honouring it would silence a package
// wholesale).
func TestTheBoundaryBehindASeamIsSilent(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "adapter")
}

// TestAFuncTypeSeamNeedsBindingEvidence pins how a function-type seam earns
// its exemption: the typed blank assertion `var _ Runner = ExecRun` is
// compiler-checked evidence the function backs the declared seam type, and it
// exempts. The same fixture pins both refusals: a function that merely MATCHES
// a declared type's signature is still reported (shape is not evidence — a
// `type task func()` would otherwise silence every niladic function), and a
// bare untyped `var _ = fn` is still reported (the blank holds nothing a test
// could replace).
func TestAFuncTypeSeamNeedsBindingEvidence(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "functype")
}

// TestAMethodOnAStdlibGlobalIsReported pins the one-selector-deeper shape:
// http.DefaultClient.Do sends the request through process-wide state no test
// can replace. The same fixture pins the boundary — http.DefaultServeMux is
// process-wide state that is NOT on the list, and is not reported.
func TestAMethodOnAStdlibGlobalIsReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "global")
}

// TestACompositionRootByPathIsExempt pins the exemption for anything beneath a
// cmd element: that is where the real world is supposed to be wired in.
func TestACompositionRootByPathIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "cmd/tool")
}

// TestACompositionRootByClauseIsExempt pins the same exemption for a main
// package, whatever its path.
func TestACompositionRootByClauseIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "root")
}

// TestATestFileIsExempt pins that a _test.go file reaching for the real clock
// and the real process table is left alone, even in a package that is
// otherwise in scope. Both shapes are reported in a non-test file of the same
// package, so the file's scope is what the fixture measures.
//
// It pins the boundary of the test-support exemption with it, which is the
// property that keeps that exemption from swallowing the fleet: tested_test.go
// imports `testing`, as every tested package's test file does, and the package
// is still in scope — the production reach for the clock in tested.go is
// reported. Only a NON-test file's import marks test support.
func TestATestFileIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "tested")
}

// TestATestSupportPackageIsExempt pins the exemption for a package that exists
// only to serve tests, proven by a non-test file importing `testing`. Its name
// carries no `test` suffix on purpose: the import is the rule, so `testutil`
// and `harness` are recognised where a suffix match would report them.
func TestATestSupportPackageIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "harness")
}

// TestAWordEndingInTestIsNotTestSupport pins the false positive the tempting
// rule would produce. Package `latest` ends in the letters "test" and is
// ordinary production code; it imports no testing package, so it stays in
// scope and its reaches are reported. `pgtest` and `latest` are the same string
// shape, which is precisely why the spelling is never consulted.
func TestAWordEndingInTestIsNotTestSupport(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "latest")
}

// TestRegistrationIsWellFormed pins the yze wiring.
func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, seams.Registration.Validate())
	assert.Equal(t, "yze/seams", seams.Registration.RuleID())
	assert.Same(t, seams.Analyzer, seams.Registration.Analyzer)
}

// TestALocalClockCaptureIsTheDocumentedBoundary pins the per-expression limit
// of the branching-clock reading: `now := time.Now(); now.After(deadline)` is
// silent, so widening the rule to follow locals must arrive as a deliberate
// fixture change, never as drift.
func TestALocalClockCaptureIsTheDocumentedBoundary(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "localstamp")
}

// TestAValueHoldMustBeReplaceable pins what makes a function-value reference
// an exemption at all: the hold has to be one a test can write over. A
// package-level var, a composite-literal element and an assignment to a field
// are the three; a local capture, a method value bound to a local, an argument
// and a return value are not, and each stays reported. An explicit generic
// instantiation is pinned in both directions with them — `spawn[int](name)` is
// a CALL and holds nothing, while `var held = spawn[int]` is a hold.
func TestAValueHoldMustBeReplaceable(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "hold")
}

// TestADeclaredInterfaceMustBeInjectable pins the same doctrine for the other
// boundary shape: an interface exempts its implementations only when
// something can hand one through it. An unexported interface written nowhere
// else is four lines that acquire the marker and none of the property, so its
// method stays reported, while the same interface named as the type of a field
// or a variable is the injected abstraction the standard asks for.
func TestADeclaredInterfaceMustBeInjectable(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "iface")
}

// TestTheDocumentedBoundariesHold pins the silences the package comment
// promises, each beside a reported sibling: the process environment is off the
// list, a dot-imported call names no package, an adapter for another package's
// interface is reported like any other method, and a thin public wrapper is
// reported over a non-clock entry point while the clock's stamp/branch
// narrowing keeps its wrapper silent.
func TestTheDocumentedBoundariesHold(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "boundary")
}

// TestACmdSubstringIsNotACompositionRoot pins the composition-root exemption
// at its boundary: `cmdutil` contains the letters and carries no `cmd`
// element, so it stays in scope. A substring match would silence it.
func TestACmdSubstringIsNotACompositionRoot(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "cmdutil")
}
