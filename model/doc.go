// Package model contains the pure data types of the Confluent Trust
// Hypergraph framework (Theory v0.2 §4.1, plus the §2.9 fork extension).
//
// The package is stdlib-only on purpose: the theory made executable should
// be testable without any infrastructure. Computed quantities (residual
// entropy, confirmatory information, mutual information, domain
// classification) are NOT stored on these types — they live in package
// compute and are produced from the raw inventory state.
//
// All exported types support JSON round-tripping and carry tags that
// match both v0.1 and v0.2 fixture formats. Validation invariants are
// expressed as Validate() methods returning wrapped errors.
package model
