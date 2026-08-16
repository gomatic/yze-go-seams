package tested

import (
	"os/exec"
	"testing"
	"time"
)

// TestNameIsReadBack reaches for the real clock and the real process table
// inside a _test.go file, where the rule stays silent. Both are shapes the
// rule reports in a non-test file of the same package, so the file's scope is
// what makes them silent here.
func TestNameIsReadBack(t *testing.T) {
	if time.Now().After(time.Time{}) {
		t.Fatal("want a clock reading before the epoch's successor")
	}
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Logf("no git here: %v", err)
	}
	if Name != "tested" {
		t.Fatalf("got %q", Name)
	}
}
