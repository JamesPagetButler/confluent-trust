// Package validate enforces the inventory JSON Schema (Draft 2020-12).
// The schema is embedded at build time, so the validator has no runtime
// dependency on the file system. Consumers (store/json.go) call
// Validate before unmarshaling so that schema violations surface as a
// rich error rather than a silent type-mismatch deep in unmarshal.
//
// This package lives under internal/ because schema validation requires
// one external dependency (github.com/santhosh-tekuri/jsonschema/v6),
// while model/ is intentionally stdlib-only per project policy.
package validate
