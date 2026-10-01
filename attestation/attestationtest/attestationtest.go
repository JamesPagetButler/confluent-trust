// Package attestationtest provides the four fake Verifiers and the shared
// verifier contract (inter#149) that every consumer runs to prove the
// v0 → v1 → v2 swap never changes caller code.
//
// The fakes are deterministic stand-ins used by contract and consumer tests;
// the real v0 is attestation.StubV0. They live in an importable package (not
// under testdata/, which the go tool excludes from builds) so that consumers in
// other repos — the QBP ledger CI, the qbp-compute-unit gate — can import and
// run the same Contract against whatever Verifier they wire. Record-level
// fixtures (the bytes fed to Verify, with expected Results) live in
// testdata/attestation/.
package attestationtest

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/attestation"
)

// AlwaysUnsigned behaves like the v0 stub: every record is unsigned.
type AlwaysUnsigned struct{}

func (AlwaysUnsigned) Verify(_ []byte, _ string) attestation.Result {
	return attestation.Result{
		Verified: false,
		Signer:   attestation.SignerUnknown,
		Role:     attestation.RoleNone,
		Method:   attestation.MethodUnsigned,
		Reason:   "fake: always unsigned",
	}
}

// ValidNotary returns a valid notary-role signature bound to claimedSigner.
type ValidNotary struct{}

func (ValidNotary) Verify(_ []byte, claimedSigner string) attestation.Result {
	return attestation.Result{
		Verified: true,
		Signer:   claimedSigner,
		Role:     attestation.RoleNotary,
		Method:   attestation.MethodGitsign,
		Reason:   "fake: valid notary signature",
	}
}

// WrongRole returns a valid signature bound to claimedSigner whose role is NOT
// notary — the AC6 case a PA consumer must reject even though Verified is true.
type WrongRole struct{}

func (WrongRole) Verify(_ []byte, claimedSigner string) attestation.Result {
	return attestation.Result{
		Verified: true,
		Signer:   claimedSigner,
		Role:     attestation.RoleNone,
		Method:   attestation.MethodGitsign,
		Reason:   "fake: valid signature, non-notary role",
	}
}

// Forged models a record that carries a signature which does not actually bind
// the claimed signer: verification fails and attribution is unknown.
type Forged struct{}

func (Forged) Verify(_ []byte, _ string) attestation.Result {
	return attestation.Result{
		Verified: false,
		Signer:   attestation.SignerUnknown,
		Role:     attestation.RoleNone,
		Method:   attestation.MethodGitsign,
		Reason:   "fake: forged — signature does not bind the claimed signer",
	}
}

// Compile-time assertions that the fakes are Verifiers.
var (
	_ attestation.Verifier = AlwaysUnsigned{}
	_ attestation.Verifier = ValidNotary{}
	_ attestation.Verifier = WrongRole{}
	_ attestation.Verifier = Forged{}
)

// Contract runs the inter#149 verifier contract against v. Every Verifier — the
// v0 stub and the eventual v1/v2 — must satisfy these invariants, so a consumer
// that relies only on them never changes when the verifier is swapped:
//
//   - Role is one of {notary, beekeeper, none};
//   - Method is one of {unsigned, gitsign, ssh, yubikey};
//   - Signer is non-empty (a seat-id, or "unknown");
//   - an unsigned record is never Verified;
//   - Verify is deterministic for a given (record, claimedSigner).
//
// It deliberately does NOT assert any particular verdict: that is each
// Verifier's own business. It asserts only the shape every caller may rely on.
func Contract(t *testing.T, name string, v attestation.Verifier) {
	t.Helper()
	inputs := []struct {
		record string
		signer string
	}{
		{`{"claim":"X","evidence_ref":"a@sha#t"}`, "cth-implementor"},
		{`{}`, "notary-implementor"},
		{``, attestation.SignerUnknown},
	}
	roleOK := map[attestation.Role]bool{
		attestation.RoleNotary: true, attestation.RoleBeekeeper: true, attestation.RoleNone: true,
	}
	methodOK := map[attestation.Method]bool{
		attestation.MethodUnsigned: true, attestation.MethodGitsign: true,
		attestation.MethodSSH: true, attestation.MethodYubiKey: true,
	}
	for _, in := range inputs {
		r := v.Verify([]byte(in.record), in.signer)
		if !roleOK[r.Role] {
			t.Errorf("%s: Role %q not in the allowed set", name, r.Role)
		}
		if !methodOK[r.Method] {
			t.Errorf("%s: Method %q not in the allowed set", name, r.Method)
		}
		if r.Signer == "" {
			t.Errorf("%s: Signer is empty (must be a seat-id or %q)", name, attestation.SignerUnknown)
		}
		if r.Method == attestation.MethodUnsigned && r.Verified {
			t.Errorf("%s: unsigned record reported Verified=true", name)
		}
		if r2 := v.Verify([]byte(in.record), in.signer); r != r2 {
			t.Errorf("%s: Verify is not deterministic: %+v vs %+v", name, r, r2)
		}
	}
}
