// Package hold pins what makes a function-value reference an exemption: the
// hold behind it has to be one a TEST CAN REPLACE.
//
// The exemption is written as the reference minus the bindings that reach
// nothing, so this fixture is in two halves. The reported half is the three
// subtractions — a blank, a local, and an argument at a parameter that is not
// a function type — in every spelling each of them has. The silent half is the
// storage a test CAN arrange, and it is deliberately not a short list: a
// package-level var written at its declaration or by a later statement, a
// field, a registry entry, a constructor argument, and each of those reached
// through a conversion the language leaves no way around.
package hold

import "os/exec"

// runner is the seam type consumers hold.
type runner func(name string) error

// --- holds: silent, because a test can write over each of these ---

// runCommand is the package-level var hold at its declaration.
var runCommand runner = execRun

// execRun is the real implementation behind that hold.
func execRun(name string) error { return exec.Command(name).Run() }

// Run goes through the hold.
func Run(name string) error { return runCommand(name) }

// resetting is the same seam bound by a later STATEMENT rather than at its
// declaration. It is the identical var and the identical substitution, so the
// spelling cannot be what decides it.
var resetting runner

// Reset binds it.
func Reset() { resetting = resetRun }

// resetRun is behind the assigned hold.
func resetRun(name string) error { return exec.Command(name).Run() }

// Resetting goes through it.
func Resetting(name string) error { return resetting(name) }

// registry is a package-level registry a test rewrites entry by entry.
var registry = map[string]runner{}

// Register binds an entry.
func Register() { registry["shell"] = registryRun }

// registryRun is behind the registry entry.
func registryRun(name string) error { return exec.Command(name).Run() }

// Registered goes through it.
func Registered(name string) error { return registry["shell"](name) }

// chain holds its runners as ELEMENTS of a named slice.
var chain = []runner{listRun}

// listRun is behind the slice-element hold.
func listRun(name string) error { return exec.Command(name).Run() }

// Chain runs the first held runner.
func Chain(name string) error { return chain[0](name) }

// store holds a collaborator in a field.
type store struct{ run runner }

// NewStore binds the field in a composite literal, which a test builds with
// something else in that position.
func NewStore() *store { return &store{run: litRun} }

// litRun is behind the composite-literal hold.
func litRun(name string) error { return exec.Command(name).Run() }

// Bind writes the same field by assignment.
func (s *store) Bind() { s.run = assignRun }

// assignRun is behind the field assignment.
func assignRun(name string) error { return exec.Command(name).Run() }

// Do runs whatever the store holds.
func (s *store) Do(name string) error { return s.run(name) }

// injected is handed to a call at a parameter declared with a FUNCTION type,
// which is the constructor injection the rule exists to encourage: the callee
// holds what it was given and a test calls it with a fake.
func injected(name string) error { return exec.Command(name).Run() }

// UseInjected hands it over at a function-typed parameter.
func UseInjected(name string) error { return apply(injected, name) }

// apply takes the collaborator, which is what makes the argument a hold.
func apply(with runner, name string) error { return with(name) }

// varied is handed to a VARIADIC function-typed parameter, which binds it the
// same way a fixed one does.
func varied(name string) error { return exec.Command(name).Run() }

// UseVaried hands it to the variadic tail.
func UseVaried(name string) error { return applyAll(name, varied) }

// applyAll takes a variadic tail of collaborators.
func applyAll(name string, with ...runner) error {
	for _, each := range with {
		if err := each(name); err != nil {
			return err
		}
	}
	return nil
}

// asserted backs runner and says so, compiler-checked. A blank binding holds
// nothing, but a blank whose ANNOTATION is a function type is the package
// asserting the function backs a declared seam shape.
var _ runner = asserted

// asserted is behind that assertion.
func asserted(name string) error { return exec.Command(name).Run() }

// instantiated is the same hold spelled as an explicit instantiation.
var instantiated runner = heldGeneric[int]

// heldGeneric is behind it.
func heldGeneric[T any](name string) error { return exec.Command(name).Run() }

// Instantiated goes through the held instantiation.
func Instantiated(name string) error { return instantiated(name) }

// instantiatedPair is the two-argument spelling of the same thing.
var instantiatedPair runner = heldGeneric2[int, string]

// heldGeneric2 is behind it.
func heldGeneric2[T, U any](name string) error { return exec.Command(name).Run() }

// InstantiatedPair goes through it.
func InstantiatedPair(name string) error { return instantiatedPair(name) }

// remote is a value whose METHOD is held at a package-level var below.
var remote = remoteSpawner{}

// heldMethod is that hold, written through a selector rather than a bare name.
var heldMethod runner = remote.Spawn

// remoteSpawner carries the held method.
type remoteSpawner struct{}

// Spawn is behind the hold, exactly as a held function is.
func (remoteSpawner) Spawn(name string) error { return exec.Command(name).Run() }

// HeldMethod goes through the hold.
func HeldMethod(name string) error { return heldMethod(name) }

// dialer is a seam declared as an INTERFACE, which is the shape that leaves no
// way to bind a function without converting it first.
type dialer interface{ dial(name string) error }

// dialFunc adapts a function to that interface, the http.HandlerFunc idiom.
type dialFunc func(name string) error

// dial satisfies the interface.
func (f dialFunc) dial(name string) error { return f(name) }

// dialing is the package-level hold, and the conversion in it is mandatory:
// there is no spelling of this seam that omits it, so a conversion has to be
// transparent or a package written the only way it can be written is reported.
var dialing dialer = dialFunc(convRun)

// convRun is behind the converted hold.
func convRun(name string) error { return exec.Command(name).Run() }

// Dial goes through it.
func Dial(name string) error { return dialing.dial(name) }

// --- not holds: reported, because a test can reach none of these ---

// captured is referenced only by a LOCAL capture. A local lives and dies
// inside the function that made it, so no test can substitute it.
func captured(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseCaptured makes that local capture.
func UseCaptured(name string) error {
	spawn := captured
	return spawn(name)
}

// reassigned is written to a local by a later statement, which is the same
// non-hold spelled as an assignment.
func reassigned(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseReassigned assigns it to a local.
func UseReassigned(name string) error {
	var local runner
	local = reassigned
	return local(name)
}

// spawner carries a method captured into a local below.
type spawner struct{}

// Spawn is referenced only as a method VALUE bound to a local.
func (spawner) Spawn(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseMethod captures the method value into a local.
func UseMethod(at spawner, name string) error {
	spawn := at.Spawn
	return spawn(name)
}

// loose is handed to a call whose parameter is `any`. Every value satisfies
// any, so the position says as little about the function as `var _ any = fn`
// does — the same refusal, in the argument's spelling.
func loose(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseLoose hands it to an any-parameter.
func UseLoose() { record(loose) }

// record takes anything at all.
func record(any) {}

// blanked is bound to a blank, which holds nothing.
func blanked(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The blank binding that must not exempt blanked.
var _ = blanked

// listed is an element of a composite literal the declaration throws away. The
// blank holds the SLICE; nothing holds listed.
func listed(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The discarded literal.
var _ = []runner{listed}

// bodyListed is the same discarded literal written inside a function body,
// where a test can reach it even less.
func bodyListed(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// Discard throws the literal away.
func Discard() { _ = []runner{bodyListed} }

// reshaped is converted and thrown away. A conversion is transparent, so the
// blank behind it decides, exactly as it does without one.
func reshaped(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The discarded conversion.
var _ = runner(reshaped)

// generic is called through an explicit instantiation. An instantiation is a
// CALL whatever its type arguments, so nothing holds generic — adding an
// unused type parameter must not silence a body.
func generic[T any](name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseGeneric spells the instantiation the ordinary way.
func UseGeneric(name string) error { return generic[int](name) }

// parenthesised is called through a parenthesised callee, which is a call.
func parenthesised(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseParens calls it through parentheses.
func UseParens(name string) error { return (parenthesised)(name) }

// parenListed is bound to a blank through parentheses, which store nothing.
func parenListed(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The parenthesised blank binding.
var _ = (parenListed)

// keyedListed is a KEYED element of a literal the declaration throws away,
// behind an address-of that stores nothing either.
func keyedListed(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The discarded keyed literal.
var _ = &store{run: keyedListed}

// instListed is bound to a blank as an explicit instantiation.
func instListed[T any](name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The discarded instantiation, in both spellings.
var (
	_ = instListed[int]
	_ = instPairListed[int, string]
)

// instPairListed is the two-argument spelling.
func instPairListed[T, U any](name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// An ordinary CALL is where the carrying stops: its result is a new value that
// names no function, so nothing beneath it is judged by this binding.
var _ = build()

// build makes a runner rather than naming one.
func build() runner { return nil }
