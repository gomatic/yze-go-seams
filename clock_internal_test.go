package seams

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBranchingClocksSeparatesAComparisonFromAStamp names branchingClocks's
// claim: only a clock reaching a comparison puts a branch out of a test's
// reach.
//
// Conflating the two was this rule's largest remaining source of noise — 24 of
// its 37 fleet findings were a timestamp written into a field or a lock file,
// where nothing downstream tests the value. An injected clock buys those sites
// a constructor parameter and no coverage.
func TestBranchingClocksSeparatesAComparisonFromAStamp(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.Empty(branchingClocks(mustParseExpr(t, `time.Now().UTC().Format(x)`)),
		"a stamped clock reaches no comparison")
	want.NotEmpty(branchingClocks(mustParseExpr(t, `time.Now().After(deadline)`)),
		"a compared clock does")
	want.NotEmpty(branchingClocks(mustParseExpr(t, `start.Before(time.Now())`)),
		"the clock as the argument is equally a branch")
	want.NotEmpty(branchingClocks(mustParseExpr(t, `time.Now().Sub(start) > budget`)),
		"and so is a comparison reached through arithmetic")
}

// TestComparisonMethodsHoldOnlyTheVerdictMethods names comparisonMethods's
// claim: these are the time.Time methods whose result IS a verdict the code
// branches on. Sub and Compare return a VALUE — a measurement, an ordering —
// which is a stamp until an operator compares it, so `time.Now().Sub(start)`
// returned whole draws the same silent verdict as the `time.Since(start)` it
// is defined to equal; the operator path reports both when a branch happens.
// Formatting or re-zoning a timestamp is neither.
func TestComparisonMethodsHoldOnlyTheVerdictMethods(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.NotEmpty(comparisonMethods, "an empty set would make nothing a branch")
	for _, method := range []string{"After", "Before", "Equal"} {
		want.True(comparisonMethods[method], "%s turns a timestamp into a verdict", method)
	}
	want.False(comparisonMethods["Sub"], "a returned measurement is a stamp, like the Since it equals")
	want.False(comparisonMethods["Compare"], "a returned ordering is a value, not yet a branch")
	want.False(comparisonMethods["Format"], "formatting a timestamp is a stamp, not a branch")
	want.False(comparisonMethods["UTC"], "changing zone is a stamp, not a branch")
}

// TestComparisonOpsHoldOnlyRelationalOperators names comparisonOps's claim.
// The set is keyed by the operator's text because token.Token enumerates every
// operator AND keyword in the language, so the six that matter would never be
// stated plainly among the seventy that cannot appear here.
func TestComparisonOpsHoldOnlyRelationalOperators(t *testing.T) {
	t.Parallel()
	want := assert.New(t)

	want.NotEmpty(comparisonOps, "an empty operator set would make nothing a branch")
	for _, op := range []string{"<", ">", "<=", ">=", "==", "!="} {
		want.True(comparisonOps[op], "%s compares", op)
	}
	want.False(comparisonOps["+"], "arithmetic alone does not compare")
	want.False(comparisonOps["&&"], "nor does a logical connective")
}

// mustParseExpr parses a Go expression for the tests above.
func mustParseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	require.NoError(t, err)
	return expr
}
