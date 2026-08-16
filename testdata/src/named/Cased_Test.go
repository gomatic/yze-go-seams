// An uppercase spelling of the suffix. The go tool's test-file rule is
// case-SENSITIVE, so this is ordinary source even on the case-insensitive
// filesystem this fleet is developed on — and case-folding the name before
// matching, which is the ordinary instinct of anyone bitten by macOS or
// Windows, would exempt a file the build ships and no test ever runs.
package named

import "os/exec"

// Cased is reported: Cased_Test.go is not _test.go.
func Cased(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}
