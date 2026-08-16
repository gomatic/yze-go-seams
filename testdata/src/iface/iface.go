// Package iface pins what makes a declared interface an exemption: something
// has to be able to hand an implementation through it. An interface nobody
// holds gives a test nothing to substitute, so declaring one silences nothing
// — otherwise four unused lines naming a method would exempt any method that
// matched them.
package iface

import "net/http"

// unheld names a method and is written nowhere else in the package: no
// parameter, no field, no result, no variable, and it is unexported, so no
// importer can hold one either.
type unheld interface {
	Charge(url string) (*http.Response, error)
}

// fielded is the same shape, written as the type of a field — which is the
// package saying an implementation is handed in through it.
type fielded interface {
	Refund(url string) (*http.Response, error)
}

// declared is the same evidence in the other spelling: the type of a variable.
type declared interface {
	Settle(url string) (*http.Response, error)
}

// Ledger holds a fielded, which is what makes fielded injectable.
type Ledger struct{ back fielded }

// Back reads the held collaborator.
func (l Ledger) Back() fielded { return l.back }

// settling holds a declared. Its type is written ONLY here — no parameter, no
// result, no field — so this declaration is the whole evidence that something
// can be handed through the interface.
var settling declared

// asserted is named only in a TYPE ASSERTION. A caller hands the package
// anything and the package uses it AS this interface, which is injection in
// the plainest form there is.
type asserted interface {
	Bill(url string) (*http.Response, error)
}

// Bill routes through whatever it was handed.
func Bill(at any, url string) (*http.Response, error) {
	if by, ok := at.(asserted); ok {
		return by.Bill(url)
	}
	return nil, nil
}

// switched is named only in a TYPE SWITCH case, which is the same evidence
// spelled the other way.
type switched interface {
	Void(url string) (*http.Response, error)
}

// Void routes through whatever it was handed.
func Void(at any, url string) (*http.Response, error) {
	switch by := at.(type) {
	case switched:
		return by.Void(url)
	}
	return nil, nil
}

// Client implements every interface's methods.
type Client struct{}

// Charge is reported: `unheld` names it, and nothing can be handed an unheld.
func (Client) Charge(url string) (*http.Response, error) {
	return http.Get(url) // want `net/http.Get is called directly`
}

// Refund is silent: `fielded` is the type of Ledger's field, so it is the
// injected abstraction the standard asks for.
func (Client) Refund(url string) (*http.Response, error) { return http.Get(url) }

// Settle is silent for the same reason, through a variable's type.
func (Client) Settle(url string) (*http.Response, error) { return http.Get(url) }

// Bill is silent: the assertion is where the package is handed one.
func (Client) Bill(url string) (*http.Response, error) { return http.Get(url) }

// Void is silent for the same reason, through a type-switch case.
func (Client) Void(url string) (*http.Response, error) { return http.Get(url) }
