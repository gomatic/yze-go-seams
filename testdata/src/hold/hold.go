// Package hold pins what makes a function-value reference an exemption: the
// hold has to be one a TEST CAN REPLACE. A package-level var, a field bound in
// a composite literal, and a field written by assignment are the three; a
// local capture, an argument, and a return value are not, because nothing a
// test can reach binds the function in any of them.
package hold

import "os/exec"

// runner is the seam type consumers hold.
type runner func(name string) error

// runCommand is the package-level var hold: a test rebinds it.
var runCommand runner = execRun

// execRun is the real implementation behind that hold, so its spawn is the
// seam's own implementation and is silent.
func execRun(name string) error { return exec.Command(name).Run() }

// Run goes through the hold.
func Run(name string) error { return runCommand(name) }

// store holds a collaborator in a field.
type store struct{ run runner }

// NewStore binds the field in a composite literal, which a test builds with
// something else in that position.
func NewStore() *store { return &store{run: litRun} }

// litRun is behind the composite-literal hold and is silent.
func litRun(name string) error { return exec.Command(name).Run() }

// Bind writes the same field by assignment, which is the same hold spelled as
// a statement.
func (s *store) Bind() { s.run = assignRun }

// assignRun is behind the field assignment and is silent.
func assignRun(name string) error { return exec.Command(name).Run() }

// Do runs whatever the store holds.
func (s *store) Do(name string) error { return s.run(name) }

// captured is referenced only by a LOCAL capture. A local lives and dies
// inside the function that made it, so no test can substitute it and the
// spawn is reported.
func captured(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseCaptured makes that local capture.
func UseCaptured(name string) error {
	spawn := captured
	return spawn(name)
}

// injected is handed to a call at a parameter declared with a FUNCTION type,
// which is the constructor injection the rule exists to encourage: the callee
// holds what it was given and a test calls it with a fake. It is silent.
func injected(name string) error { return exec.Command(name).Run() }

// UseInjected hands it over at a function-typed parameter.
func UseInjected(name string) error { return apply(injected, name) }

// apply takes the collaborator, which is what makes the argument a hold.
func apply(with runner, name string) error { return with(name) }

// loose is handed to a call whose parameter is `any`. Every value satisfies
// any, so the position binds nothing about the function and it stays reported
// — the same refusal `var _ any = fn` gets.
func loose(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseLoose hands it to an any-parameter, which is not evidence of anything.
func UseLoose() { record(loose) }

// record takes anything at all.
func record(what any) { _ = what }

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

// spread is handed to the same variadic tail as an already-built slice, which
// binds the slice rather than any function by name — the reference that holds
// it is the composite literal below.
var spreadable = []runner{spread}

// spread is behind that slice-element hold and is silent.
func spread(name string) error { return exec.Command(name).Run() }

// UseSpread passes the whole slice.
func UseSpread(name string) error { return applyAll(name, spreadable...) }

// returned is handed back to a caller, which is not a hold either.
func returned(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// Returned hands it back.
func Returned() runner { return returned }

// spawner carries a method captured into a local below.
type spawner struct{}

// Spawn is referenced only as a method VALUE bound to a local, which is the
// same non-hold as any other local.
func (spawner) Spawn(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseMethod captures the method value into a local.
func UseMethod(at spawner, name string) error {
	spawn := at.Spawn
	return spawn(name)
}

// generic is called through an explicit instantiation below. An instantiation
// is a CALL whatever its type arguments, so nothing holds generic and its
// spawn is reported.
func generic[T any](name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// UseGeneric spells the instantiation the ordinary way.
func UseGeneric(name string) error { return generic[int](name) }

// instantiated is the same function HELD at a package-level var, as an
// instantiation rather than a call — which is a hold, and silent.
var instantiated runner = heldGeneric[int]

// heldGeneric is behind that hold.
func heldGeneric[T any](name string) error { return exec.Command(name).Run() }

// Instantiated goes through the held instantiation.
func Instantiated(name string) error { return instantiated(name) }

// heldGeneric2 is held at a package-level var through a two-argument
// instantiation, which is the same hold spelled with a list.
var instantiatedPair runner = heldGeneric2[int, string]

// heldGeneric2 is behind that hold.
func heldGeneric2[T, U any](name string) error { return exec.Command(name).Run() }

// InstantiatedPair goes through it.
func InstantiatedPair(name string) error { return instantiatedPair(name) }

// chain holds its runners as ELEMENTS of a slice literal rather than as named
// fields; an element is a hold a test can write over exactly as a field is.
var chain = []runner{listRun}

// listRun is behind the slice-element hold and is silent.
func listRun(name string) error { return exec.Command(name).Run() }

// Chain runs the first held runner.
func Chain(name string) error { return chain[0](name) }

// pair yields two runners at once.
func pair() (runner, runner) { return listRun, litRun }

// A multi-value binding from ONE expression holds the call's results, not any
// function by name, so neither position marks anything.
var first, second = pair()

// Pair runs both.
func Pair(name string) error {
	if err := first(name); err != nil {
		return err
	}
	return second(name)
}

// Rebind writes both fields from one call, which binds the results and names
// no function.
func (s *store) Rebind(other *store) { s.run, other.run = pair() }

// remote is a value whose METHOD is held at a package-level var below.
var remote = remoteSpawner{}

// heldMethod is that hold, written through a selector rather than a bare name.
var heldMethod runner = remote.Spawn

// remoteSpawner carries the held method.
type remoteSpawner struct{}

// Spawn is behind the hold and is silent, exactly as a held function is.
func (remoteSpawner) Spawn(name string) error { return exec.Command(name).Run() }

// HeldMethod goes through the hold.
func HeldMethod(name string) error { return heldMethod(name) }

// converted is written inside a CONVERSION to a niladic function type, which
// Go spells as a call carrying one argument and declaring no parameter at all.
// There is no parameter for the argument to bind, so it is not a hold and the
// spawn is reported.
func converted() {
	_ = exec.Command("git", "status").Run() // want `os/exec.Command is called directly`
}

// The conversion that binds nothing.
var _ = (func())(converted)

// A CONVERSION is spelled as a call whose callee is a TYPE, so it declares no
// parameters and binds nothing — `runner(reshaped)` changes how the value is
// named, not who holds it.
func reshaped(name string) error {
	return exec.Command(name).Run() // want `os/exec.Command is called directly`
}

// The conversion that holds nothing.
var _ = runner(reshaped)

// A BUILTIN is spelled as a call whose parameters the checker supplies from
// the call itself, so `append(chain, appended)` binds its argument to a
// `runner` parameter exactly as `[]runner{appended}` does — and appending into
// the package's own slice is that same hold, one step later. It is silent.
func appended(name string) error { return exec.Command(name).Run() }

// Append grows the package's slice with it.
func Append() { chain = append(chain, appended) }
