package seams

// symbol is a stdlib identifier written as its import path joined to its name,
// e.g. "os.ReadFile" or "math/rand/v2.IntN". The path is taken from the type
// checker, never from the qualifier a file wrote, so the two packages that
// both spell themselves `rand` are never confused for one another.
type symbol string

// impureFuncs is the stdlib call surface whose result depends on something
// outside the process — and, just as importantly, whose failure branch is out
// of a test's reach at a direct call site.
//
// The list is deliberately narrow. Everything on it reads the clock, the
// hidden global random source, the filesystem, the network, or spawns a
// process. A constructor that takes its state as an argument is NOT on it:
// `rand.New`, `rand.NewPCG`, `rand.NewSource`, `http.NewRequest`, `net.JoinHostPort`
// are reproducible functions of their inputs and injectable by construction.
var impureFuncs = map[symbol]bool{
	// The clock.
	"time.Now": true,

	// The package-global random source. A generator built from an explicit
	// source is deterministic and needs no seam, so only the top-level
	// functions that read the hidden global are listed.
	"math/rand.ExpFloat64":     true,
	"math/rand.Float32":        true,
	"math/rand.Float64":        true,
	"math/rand.Int":            true,
	"math/rand.Int31":          true,
	"math/rand.Int31n":         true,
	"math/rand.Int63":          true,
	"math/rand.Int63n":         true,
	"math/rand.Intn":           true,
	"math/rand.NormFloat64":    true,
	"math/rand.Perm":           true,
	"math/rand.Read":           true,
	"math/rand.Seed":           true,
	"math/rand.Shuffle":        true,
	"math/rand.Uint32":         true,
	"math/rand.Uint64":         true,
	"math/rand/v2.ExpFloat64":  true,
	"math/rand/v2.Float32":     true,
	"math/rand/v2.Float64":     true,
	"math/rand/v2.Int":         true,
	"math/rand/v2.Int32":       true,
	"math/rand/v2.Int32N":      true,
	"math/rand/v2.Int64":       true,
	"math/rand/v2.Int64N":      true,
	"math/rand/v2.IntN":        true,
	"math/rand/v2.N":           true,
	"math/rand/v2.NormFloat64": true,
	"math/rand/v2.Perm":        true,
	"math/rand/v2.Shuffle":     true,
	"math/rand/v2.Uint":        true,
	"math/rand/v2.Uint32":      true,
	"math/rand/v2.Uint32N":     true,
	"math/rand/v2.Uint64":      true,
	"math/rand/v2.Uint64N":     true,
	"math/rand/v2.UintN":       true,

	// OS entropy.
	"crypto/rand.Int":   true,
	"crypto/rand.Prime": true,
	"crypto/rand.Read":  true,
	"crypto/rand.Text":  true,

	// The filesystem is DELIBERATELY ABSENT. It was on this list, and it was
	// wrong: it produced 146 of 187 findings across 105 modules, none of them a
	// defect.
	//
	// The rule's premise is that the branches around the call cannot be reached
	// from a test. That is true of the clock and the hidden random source, and
	// false of the filesystem — t.TempDir gives a test a real directory, and a
	// bad path reaches the failure branch. gomatic/go-archive settles it: a
	// filesystem library at 100% coverage, calling os directly throughout. A
	// rule cannot demand a seam for something already fully covered without one.
	//
	// Worse, the demand was self-defeating on the packages that exist precisely
	// to BE the filesystem seam. Telling an adapter to inject the call it was
	// written to wrap only moves the untestable line somewhere else.
	//
	// A genuinely unreachable filesystem branch — a rename failing midway — is
	// still handled by a package-level function var, which the standard
	// sanctions. That is an option where a test needs it, not a shape every call
	// site owes.

	// The network.
	"net.Dial":                   true,
	"net.DialIP":                 true,
	"net.DialTCP":                true,
	"net.DialTimeout":            true,
	"net.DialUDP":                true,
	"net.DialUnix":               true,
	"net.Listen":                 true,
	"net.ListenPacket":           true,
	"net.ListenTCP":              true,
	"net.ListenUDP":              true,
	"net.ListenUnix":             true,
	"net.LookupAddr":             true,
	"net.LookupCNAME":            true,
	"net.LookupHost":             true,
	"net.LookupIP":               true,
	"net.LookupMX":               true,
	"net.LookupNS":               true,
	"net.LookupPort":             true,
	"net.LookupSRV":              true,
	"net.LookupTXT":              true,
	"net/http.Get":               true,
	"net/http.Head":              true,
	"net/http.ListenAndServe":    true,
	"net/http.ListenAndServeTLS": true,
	"net/http.Post":              true,
	"net/http.PostForm":          true,
	"net/http.Serve":             true,
	"net/http.ServeTLS":          true,

	// A subprocess.
	"os/exec.Command":        true,
	"os/exec.CommandContext": true,
	"os/exec.LookPath":       true,
}

// impureGlobals is the stdlib package-level state whose methods reach the real
// world. A call on one of these is a call on a process-wide singleton that no
// test can replace — `http.DefaultClient.Do(req)` sends the request for real,
// where an injected `*http.Client` would not.
var impureGlobals = map[symbol]bool{
	"net/http.DefaultClient": true,
}
