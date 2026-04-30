// Package compute contains the pure functions from Confluent-Trust-Hypergraph
// theory v0.2 — entropy, fidelity, mutual information, compression, gap,
// merge, fork health, branch consistency. All functions take model.Inventory
// (or fragments of it) and return scalars or small result structs.
//
// This package is stdlib-only and has no side effects. It is consumed by
// store/, report/, and cmd/cth/.
package compute
