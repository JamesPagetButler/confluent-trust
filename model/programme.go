package model

// Programme is an envelope for cross-programme operations: it pairs an
// inventory with a stable identifier so merge / compare functions can
// reference the source. Most callers operate directly on Inventory.
type Programme struct {
	ID        string    `json:"id"`
	Inventory Inventory `json:"inventory"`
}
