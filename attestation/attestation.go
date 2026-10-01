// Package attestation defines the Evidence Attestation Verifier interface
// (inter#149) and its v0 stub.
//
// Consumers — the CTH PA engine (confluent-trust#108), the QBP ledger CI, the
// qbp-compute-unit gate — depend ONLY on this interface, never on the signing
// mechanism behind it, so swapping v0 (stub) → v1 (gitsign) → v2 (YubiKey) must
// not change any caller's code. Under v0 every record is unsigned; a consumer's
// truth guarantee therefore comes from RE-EXECUTING the evidence, never from a
// signature. The shared contract that every Verifier must satisfy lives in
// package attestationtest, and is exercised against the stub plus the fakes.
//
// The package is stdlib-only, matching package model.
package attestation

// Role is the authority a signer holds, resolved from the authority registry
// (a Verðandi grant); it is not inferred from the signature itself.
type Role string

// The authority roles a signer may hold.
const (
	RoleNotary    Role = "notary"
	RoleBeekeeper Role = "beekeeper"
	RoleNone      Role = "none"
)

// Method is how a record was signed (or that it was not).
type Method string

// The signing methods a record may carry (or that it was not signed).
const (
	MethodUnsigned Method = "unsigned"
	MethodGitsign  Method = "gitsign"
	MethodSSH      Method = "ssh"
	MethodYubiKey  Method = "yubikey"
)

// SignerUnknown is the Signer value when attribution is unavailable.
const SignerUnknown = "unknown"

// Result is the verdict returned by Verify, in the inter#149 shape.
type Result struct {
	Signer   string `json:"signer"`
	Role     Role   `json:"role"`
	Method   Method `json:"method"`
	Reason   string `json:"reason"`
	Verified bool   `json:"verified"`
}

// Verifier verifies that recordBytes carries a valid signature for
// claimedSigner. Implementations MUST satisfy the contract in package
// attestationtest so they are swappable without changing any caller.
type Verifier interface {
	Verify(recordBytes []byte, claimedSigner string) Result
}

// StubV0 is the v0 verifier (inter#149): every record is unsigned, signer
// unknown, verified false. Consumers that see this must fall back to
// re-execution for their truth guarantee.
type StubV0 struct{}

// Verify always reports the record as unsigned.
func (StubV0) Verify(_ []byte, _ string) Result {
	return Result{
		Verified: false,
		Signer:   SignerUnknown,
		Role:     RoleNone,
		Method:   MethodUnsigned,
		Reason:   "v0 stub: all records unsigned; truth guarantee is re-execution, not signature",
	}
}

// Compile-time assertion that StubV0 is a Verifier.
var _ Verifier = StubV0{}
