package seams_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	seams "github.com/gomatic/yze-go-seams"
)

// TestADirectCallIsReported pins the rule: every impure stdlib entry point on
// the list — the clock, the global random source, OS entropy, the filesystem,
// the network, a subprocess — is reported at the callee when it is called
// through its package, in package state as well as in a function body.
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
// implementation would produce. time.Since and Time.Add are pure functions of
// their arguments, and rand.New/rand.NewPCG build a generator from a seed the
// caller chose — all reproducible, none in need of a seam. They live in the
// same fixture as the conforming forms because a single silent package is the
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

// TestATestFileIsExempt pins that a _test.go file reaching for the real
// filesystem and the real clock is left alone, even in a package that is
// otherwise in scope.
func TestATestFileIsExempt(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), seams.Analyzer, "tested")
}

// TestRegistrationIsWellFormed pins the yze wiring.
func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, seams.Registration.Validate())
	assert.Equal(t, "yze/seams", seams.Registration.RuleID())
	assert.Same(t, seams.Analyzer, seams.Registration.Analyzer)
}
