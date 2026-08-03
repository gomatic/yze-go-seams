package tested

import (
	"os"
	"testing"
	"time"
)

// TestNameIsReadBack reaches for the real filesystem and the real clock inside
// a _test.go file, where the rule stays silent.
func TestNameIsReadBack(t *testing.T) {
	if _, err := os.ReadFile("nothing-is-here"); err == nil {
		t.Fatal("want an error from the real filesystem")
	}
	if time.Now().IsZero() {
		t.Fatal("want a real clock reading")
	}
	if Name != "tested" {
		t.Fatalf("got %q", Name)
	}
}
