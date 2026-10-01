package attestation_test

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/attestation"
	"github.com/JamesPagetButler/confluent-trust/attestation/attestationtest"
)

// TestVerifierContract is the shared contract every consumer runs: the v0 stub
// and all four fakes must satisfy the inter#149 invariants, so swapping the
// verifier never changes caller code.
func TestVerifierContract(t *testing.T) {
	verifiers := map[string]attestation.Verifier{
		"StubV0":         attestation.StubV0{},
		"AlwaysUnsigned": attestationtest.AlwaysUnsigned{},
		"ValidNotary":    attestationtest.ValidNotary{},
		"WrongRole":      attestationtest.WrongRole{},
		"Forged":         attestationtest.Forged{},
	}
	for name, v := range verifiers {
		t.Run(name, func(t *testing.T) {
			attestationtest.Contract(t, name, v)
		})
	}
}

// TestStubV0_AlwaysUnsigned pins the v0 verdict exactly (inter#149): every
// record unsigned, signer unknown, role none, verified false.
func TestStubV0_AlwaysUnsigned(t *testing.T) {
	r := attestation.StubV0{}.Verify([]byte(`{"claim":"PROOF-hessian"}`), "cth-implementor")
	if r.Verified {
		t.Errorf("v0 stub: Verified = true, want false")
	}
	if r.Signer != attestation.SignerUnknown {
		t.Errorf("v0 stub: Signer = %q, want %q", r.Signer, attestation.SignerUnknown)
	}
	if r.Role != attestation.RoleNone {
		t.Errorf("v0 stub: Role = %q, want %q", r.Role, attestation.RoleNone)
	}
	if r.Method != attestation.MethodUnsigned {
		t.Errorf("v0 stub: Method = %q, want %q", r.Method, attestation.MethodUnsigned)
	}
}

// TestFakes_DistinctVerdicts guards that the fakes model the distinct AC6 cases
// a PA consumer must tell apart: a valid-notary pass, a verified-but-wrong-role
// record (must be rejected despite Verified=true), and a forged record.
func TestFakes_DistinctVerdicts(t *testing.T) {
	const signer = "notary-implementor"

	vn := attestationtest.ValidNotary{}.Verify(nil, signer)
	if !(vn.Verified && vn.Role == attestation.RoleNotary && vn.Signer == signer) {
		t.Errorf("ValidNotary: want verified+notary+%s, got %+v", signer, vn)
	}

	wr := attestationtest.WrongRole{}.Verify(nil, signer)
	if !(wr.Verified && wr.Role != attestation.RoleNotary) {
		t.Errorf("WrongRole: want verified with role != notary, got %+v", wr)
	}

	fg := attestationtest.Forged{}.Verify(nil, signer)
	if fg.Verified || fg.Signer == signer {
		t.Errorf("Forged: want not-verified and signer not bound to %s, got %+v", signer, fg)
	}
}
