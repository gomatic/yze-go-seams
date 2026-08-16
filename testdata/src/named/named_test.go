package named

import (
	"os/exec"
	"testing"
)

// TestControl reaches for the real process table in a file whose name IS
// `_test.go`, exactly, which is the only spelling that leaves scope.
func TestControl(t *testing.T) {
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Logf("no git here: %v", err)
	}
}
