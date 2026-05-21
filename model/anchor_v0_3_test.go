package model

import (
	"strings"
	"testing"
)

// verificationRecord returns a valid VerificationRecord for use in tests.
func verificationRecord() *VerificationRecord {
	return &VerificationRecord{
		Toolchain: "leanprover/lean4:v4.30.0-rc2",
		Libraries: map[string]LibraryRef{
			"mathlib": {Ref: "v4.30.0", SHA: "abc123def456"},
		},
		VerifiedAt: "2026-05-20T10:00:00Z",
		Verifier:   "lake-build-ci",
		Result:     "zero-sorry",
	}
}

// baseAnchor returns a minimal valid anchor for use as a test base.
func baseAnchor(id string) Anchor {
	return Anchor{
		ID:              id,
		Name:            id,
		Description:     "test anchor",
		Tier:            TierProof,
		Provenance:      ProvenanceTheoretical,
		Status:          StatusCoherent,
		PredictionChain: []string{"AXIOM-base"},
	}
}

// ---- Invariant 2: proof_state verified/partial requires verification ----

func TestInvariant2_VerifiedRequiresVerification_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv2-neg")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	// no Verification set
	if err := a.Validate(); err == nil {
		t.Error("expected error: verified proof without verification record, got nil")
	} else if !strings.Contains(err.Error(), "verification record") {
		t.Errorf("error %q does not mention verification record", err)
	}
}

func TestInvariant2_PartialRequiresVerification_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv2-partial-neg")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStatePartial
	// no Verification set
	if err := a.Validate(); err == nil {
		t.Error("expected error: partial proof without verification record, got nil")
	}
}

func TestInvariant2_VerifiedRequiresToolchain_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv2-toolchain-neg")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	a.Verification = &VerificationRecord{
		// Toolchain intentionally empty
		Libraries:  map[string]LibraryRef{"mathlib": {Ref: "v4", SHA: "abc123"}},
		VerifiedAt: "2026-05-20T10:00:00Z",
		Verifier:   "ci",
		Result:     "zero-sorry",
	}
	a.Theorems = []TheoremRef{{Name: "thm", Status: TheoremStatusVerified}}
	if err := a.Validate(); err == nil {
		t.Error("expected error: empty toolchain, got nil")
	} else if !strings.Contains(err.Error(), "toolchain") {
		t.Errorf("error %q does not mention toolchain", err)
	}
}

func TestInvariant2_VerifiedRequiresLibrarySHA_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv2-sha-neg")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	a.Verification = &VerificationRecord{
		Toolchain:  "leanprover/lean4:v4.30.0",
		Libraries:  map[string]LibraryRef{"mathlib": {Ref: "v4", SHA: ""}}, // empty SHA
		VerifiedAt: "2026-05-20T10:00:00Z",
		Verifier:   "ci",
		Result:     "zero-sorry",
	}
	a.Theorems = []TheoremRef{{Name: "thm", Status: TheoremStatusVerified}}
	if err := a.Validate(); err == nil {
		t.Error("expected error: empty library sha, got nil")
	} else if !strings.Contains(err.Error(), "sha") {
		t.Errorf("error %q does not mention sha", err)
	}
}

func TestInvariant2_VerifiedProof_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv2-pos")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	a.Verification = verificationRecord()
	a.Theorems = []TheoremRef{{Name: "my_theorem", Status: TheoremStatusVerified}}
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

// ---- Invariant 3: proof_state verified ⟹ all theorems verified ----

func TestInvariant3_VerifiedWithUnverifiedTheorem_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv3-neg")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	a.Verification = verificationRecord()
	a.Theorems = []TheoremRef{
		{Name: "thm_ok", Status: TheoremStatusVerified},
		{Name: "thm_bad", Status: TheoremStatusWritten}, // not verified
	}
	if err := a.Validate(); err == nil {
		t.Error("expected error: verified state with non-verified theorem, got nil")
	} else if !strings.Contains(err.Error(), "all theorems verified") {
		t.Errorf("error %q does not mention 'all theorems verified'", err)
	}
}

func TestInvariant3_VerifiedWithAllTheoremVerified_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv3-pos")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	a.Verification = verificationRecord()
	a.Theorems = []TheoremRef{
		{Name: "thm_a", Status: TheoremStatusVerified},
		{Name: "thm_b", Status: TheoremStatusVerified},
	}
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestInvariant3_PartialAllowsMixedTheoremStatus_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv3-partial-pos")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStatePartial
	a.Verification = verificationRecord()
	a.Theorems = []TheoremRef{
		{Name: "thm_verified", Status: TheoremStatusVerified},
		{Name: "thm_written", Status: TheoremStatusWritten},
	}
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid for partial with mixed theorems, got: %v", err)
	}
}

// ---- Invariant 4: provenance_kind != proof ⟹ no proof-* fields ----

func TestInvariant4_TheoryWithProofState_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv4-theory-neg")
	a.ProvenanceKind = ProvenanceKindTheory
	a.ProofState = ProofStateWritten // must be absent
	if err := a.Validate(); err == nil {
		t.Error("expected error: theory with proof_state, got nil")
	} else if !strings.Contains(err.Error(), "proof_state") {
		t.Errorf("error %q does not mention proof_state", err)
	}
}

func TestInvariant4_TheoryWithProofLanguage_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv4-lang-neg")
	a.ProvenanceKind = ProvenanceKindTheory
	a.ProofLanguage = "lean4" // must be absent
	if err := a.Validate(); err == nil {
		t.Error("expected error: theory with proof_language, got nil")
	} else if !strings.Contains(err.Error(), "proof_language") {
		t.Errorf("error %q does not mention proof_language", err)
	}
}

func TestInvariant4_TheoryWithTheorems_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv4-theorems-neg")
	a.ProvenanceKind = ProvenanceKindTheory
	a.Theorems = []TheoremRef{{Name: "thm", Status: TheoremStatusVerified}}
	if err := a.Validate(); err == nil {
		t.Error("expected error: theory with theorems, got nil")
	} else if !strings.Contains(err.Error(), "theorems") {
		t.Errorf("error %q does not mention theorems", err)
	}
}

func TestInvariant4_TheoryWithVerification_Negative(t *testing.T) {
	a := baseAnchor("PROOF-inv4-verification-neg")
	a.ProvenanceKind = ProvenanceKindTheory
	a.Verification = verificationRecord() // must be absent
	if err := a.Validate(); err == nil {
		t.Error("expected error: theory with verification, got nil")
	} else if !strings.Contains(err.Error(), "verification") {
		t.Errorf("error %q does not mention verification", err)
	}
}

func TestInvariant4_TheoryWithNoProsFields_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv4-theory-pos")
	a.ProvenanceKind = ProvenanceKindTheory
	// no proof-* fields set
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid theory anchor, got: %v", err)
	}
}

func TestInvariant4_InternalComputeClean_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv4-ic-pos")
	a.ProvenanceKind = ProvenanceKindInternalCompute
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid internal-compute anchor, got: %v", err)
	}
}

func TestInvariant4_PhilosophyClean_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv4-phil-pos")
	a.ProvenanceKind = ProvenanceKindPhilosophy
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid philosophy anchor, got: %v", err)
	}
}

func TestInvariant4_ProofKindAllowsProofFields_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv4-proof-pos")
	a.ProvenanceKind = ProvenanceKindProof
	a.ProofState = ProofStateVerified
	a.ProofLanguage = "lean4"
	a.Verification = verificationRecord()
	a.Theorems = []TheoremRef{{Name: "thm", Status: TheoremStatusVerified}}
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid proof anchor, got: %v", err)
	}
}

// TestInvariant4_UnknownProvenanceKindAllowsAll verifies that when
// ProvenanceKind is unknown (zero value), proof-* fields are not rejected.
// This preserves backwards-compatibility with v0.2 anchors that lack
// provenance_kind (design §7 transitional dual-field reading).
func TestInvariant4_UnknownProvenanceKindAllowsAll_Positive(t *testing.T) {
	a := baseAnchor("PROOF-inv4-unknown-pos")
	// ProvenanceKind intentionally left as zero (Unknown)
	a.ProofState = ProofStateVerified
	a.ProofLanguage = "lean4"
	a.Verification = verificationRecord()
	a.Theorems = []TheoremRef{{Name: "thm", Status: TheoremStatusVerified}}
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid anchor with unknown provenance_kind, got: %v", err)
	}
}
