// A dot-imported package makes `Now()` indistinguishable from a local seam at
// the call site, so the rule stays silent on it. The reported sibling for this
// silence is Spawn, in the same package.
package boundary

import . "time"

// ExpiredDotted branches on the clock through a dot-imported package, which
// names no package at the call site and is not reported.
func ExpiredDotted(deadline Time) bool { return Now().After(deadline) }
