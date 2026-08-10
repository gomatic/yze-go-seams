// Package localstamp pins the documented per-expression boundary of the
// branching-clock reading: a clock captured into a local and compared LATER is
// judged a stamp at the call site and stays silent. Widening the rule to
// follow the value through locals is a deliberate future decision; this
// fixture exists so that widening shows up as a fixture change someone chose,
// never as silent drift.
package localstamp

import "time"

// ExpiredViaLocal branches on the clock through a local. The direct-expression
// spelling of the same branch is reported (see the direct fixture); this one
// is silent by the documented per-expression boundary.
func ExpiredViaLocal(deadline time.Time) bool {
	now := time.Now()
	return now.After(deadline)
}
