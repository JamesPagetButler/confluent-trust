package leanlink_test

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/internal/leanlink"
	"github.com/JamesPagetButler/confluent-trust/model"
)

// helpers to build minimal anchors for reconcile tests.

func proofAnchor(id, proofFile string, sorryCount *int, theorems []model.TheoremRef) model.Anchor {
	return model.Anchor{
		ID:              id,
		Name:            id,
		Description:     "test anchor " + id,
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofLanguage:   "lean4",
		ProofFile:       proofFile,
		ProofState:      model.ProofStateWritten,
		SorryCount:      sorryCount,
		Theorems:        theorems,
		PredictionChain: []string{},
	}
}

func intPtr(n int) *int { return &n }

func synthTheorem(name, file string, sorryCount int) leanlink.TheoremDecl {
	return leanlink.TheoremDecl{
		Name:       name,
		Kind:       "theorem",
		File:       file,
		Line:       1,
		SorryCount: sorryCount,
	}
}

func makeInventory(anchors []model.Anchor) model.Inventory {
	return model.Inventory{
		Programme:     "test",
		Version:       "0.0.1",
		SchemaVersion: "v0.3",
		Anchors:       anchors,
	}
}

// 1. Proven case.
func TestReconcile_ProvenCase(t *testing.T) {
	t.Parallel()
	anchor := proofAnchor("PROOF-A", "Proofs/Foo.lean", intPtr(0),
		[]model.TheoremRef{{Name: "foo_theorem", Status: model.TheoremStatusWritten}})
	theorems := []leanlink.TheoremDecl{synthTheorem("foo_theorem", "Proofs/Foo.lean", 0)}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	foundProven := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-A" && cl.Class == leanlink.ClassProven {
			foundProven = true
		}
	}
	if !foundProven {
		t.Errorf("expected ClassProven for PROOF-A, got classifications: %v", r.Classifications)
	}
}

// 2. Orphan case.
func TestReconcile_OrphanCase(t *testing.T) {
	t.Parallel()
	// No anchors reference orphan_theorem.
	theorems := []leanlink.TheoremDecl{synthTheorem("orphan_theorem", "Proofs/Orphan.lean", 0)}
	inv := makeInventory(nil)

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	if len(r.Orphans) != 1 || r.Orphans[0].Name != "orphan_theorem" {
		t.Errorf("expected orphan_theorem in orphans, got %v", r.Orphans)
	}
}

// 3. Stale-ref case 1: anchor has no proof_file.
func TestReconcile_StaleRef_NoProofFile(t *testing.T) {
	t.Parallel()
	anchor := model.Anchor{
		ID:              "PROOF-B",
		Name:            "PROOF-B",
		Description:     "no proof file",
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofState:      model.ProofStateWritten,
		PredictionChain: []string{},
		Theorems: []model.TheoremRef{
			{Name: "missing_theorem", Status: model.TheoremStatusWritten},
		},
	}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, nil, leanlink.ToolchainSpec{})

	found := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-B" && cl.Class == leanlink.ClassStaleRef {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ClassStaleRef for PROOF-B (no proof_file), got %v", r.Classifications)
	}
}

// 4. Stale-ref case 2: anchor has proof_file but theorem name not in file.
func TestReconcile_StaleRef_TheoremNotFound(t *testing.T) {
	t.Parallel()
	anchor := proofAnchor("PROOF-C", "Proofs/Bar.lean", intPtr(0),
		[]model.TheoremRef{{Name: "nonexistent_theorem", Status: model.TheoremStatusWritten}})
	// File exists in corpus (different theorem).
	theorems := []leanlink.TheoremDecl{synthTheorem("other_theorem", "Proofs/Bar.lean", 0)}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	found := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-C" && cl.Class == leanlink.ClassStaleRef {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ClassStaleRef for PROOF-C (theorem not found), got %v", r.Classifications)
	}
}

// 5. Drift case.
func TestReconcile_DriftCase(t *testing.T) {
	t.Parallel()
	anchor := proofAnchor("PROOF-D", "Proofs/Drift.lean", intPtr(0),
		[]model.TheoremRef{{Name: "drift_theorem", Status: model.TheoremStatusWritten}})
	// File has 2 sorries but anchor says 0.
	theorems := []leanlink.TheoremDecl{synthTheorem("drift_theorem", "Proofs/Drift.lean", 2)}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	foundDrift := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-D" && cl.Class == leanlink.ClassDrift {
			foundDrift = true
		}
	}
	if !foundDrift {
		t.Errorf("expected ClassDrift for PROOF-D, got %v", r.Classifications)
	}
	// Proposed update should be present.
	foundUpdate := false
	for _, u := range r.Updates {
		if u.AnchorID == "PROOF-D" && u.Field == "sorry_count" {
			foundUpdate = true
		}
	}
	if !foundUpdate {
		t.Errorf("expected sorry_count update proposal for PROOF-D, got %v", r.Updates)
	}
}

// 6. Phantom-theorem (Invariant 5) case.
func TestReconcile_PhantomTheoremCase(t *testing.T) {
	t.Parallel()
	anchor := proofAnchor("PROOF-E", "Proofs/Phantom.lean", nil,
		[]model.TheoremRef{{Name: "phantom_proof", Status: model.TheoremStatusNotStarted}})
	// Theorem exists in the file — Invariant 5 violation.
	theorems := []leanlink.TheoremDecl{synthTheorem("phantom_proof", "Proofs/Phantom.lean", 0)}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	found := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-E" && cl.Class == leanlink.ClassPhantomTheorem {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ClassPhantomTheorem for PROOF-E, got %v", r.Classifications)
	}
}

// 7. Verification proposal + proof_state advancement.
func TestReconcile_VerificationProposal(t *testing.T) {
	t.Parallel()
	anchor := proofAnchor("PROOF-F", "Proofs/Verified.lean", intPtr(0),
		[]model.TheoremRef{{Name: "verified_theorem", Status: model.TheoremStatusWritten}})
	theorems := []leanlink.TheoremDecl{synthTheorem("verified_theorem", "Proofs/Verified.lean", 0)}
	inv := makeInventory([]model.Anchor{anchor})

	spec := leanlink.ToolchainSpec{
		Toolchain: "leanprover/lean4:v4.30.0-rc2",
		Libraries: map[string]model.LibraryRef{
			"mathlib": {SHA: "abc123", Ref: "v4.30.0"},
		},
	}

	r := leanlink.Reconcile(inv, theorems, spec)

	// Must be proven.
	foundProven := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-F" && cl.Class == leanlink.ClassProven {
			foundProven = true
		}
	}
	if !foundProven {
		t.Errorf("expected ClassProven for PROOF-F, got %v", r.Classifications)
	}

	// Verification + proof_state updates should be proposed.
	var hasVerification, hasProofState bool
	for _, u := range r.Updates {
		if u.AnchorID == "PROOF-F" {
			switch u.Field {
			case "verification":
				hasVerification = true
			case "proof_state":
				hasProofState = true
			}
		}
	}
	if !hasVerification {
		t.Errorf("expected verification update proposal for PROOF-F, got %v", r.Updates)
	}
	if !hasProofState {
		t.Errorf("expected proof_state update proposal for PROOF-F, got %v", r.Updates)
	}
}
