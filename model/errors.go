package model

import "errors"

// ErrIDCollision is returned by store.LiveInventory when appending a record
// whose ID collides with any existing record in the inventory (anchors,
// axioms, derived principles, inputs, chains, confluences). Wrapped via
// fmt.Errorf to include the colliding ID.
var ErrIDCollision = errors.New("model: id collision")

// ErrClosed is returned when a method is called on a closed LiveInventory.
var ErrClosed = errors.New("model: handle closed")

// ErrNotFound is returned by store.LiveInventory.UpdateAnchor / UpdateChain
// when no record with the given ID exists.
var ErrNotFound = errors.New("model: id not found")
