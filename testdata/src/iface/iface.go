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

// Client implements all three interfaces' methods.
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
