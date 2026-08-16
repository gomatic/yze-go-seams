// A base name ending in "test.go" WITHOUT the underscore. The go tool
// compiles it as ordinary source, so a matcher that dropped the underscore
// would exempt a file the build ships.
package named

import "os/exec"

// Serve is reported: httptest.go is not a test file.
func Serve(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}
