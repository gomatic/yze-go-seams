// A name CONTAINING "_test.go" without ending in it. A matcher using Contains
// rather than HasSuffix would exempt this ordinary source file.
package named

import "os/exec"

// Golden is reported: the suffix is ".golden.go", not "_test.go".
func Golden(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}
