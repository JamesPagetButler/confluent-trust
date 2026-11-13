package leanlink_test

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/internal/leanlink"
	"github.com/JamesPagetButler/confluent-trust/model"
)

// helpers to build minimal anchors for reconcile tests.

const testLeanLanguage = "lean4"

func proofAnchor(id, proofFile string, sorryCount *int, theorems []model.TheoremRef) model.Anchor {
	return model.Anchor{
		ID:              id,
		Name:            id,
		Description:     "test anchor " + id,
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofLanguage:   testLeanLanguage,
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

// 4b. Module::theorem join key (CTH #124): doc/design/proven-theorem-anchoring.md
// §3.4 specifies "<module>::<theorem>" as the join key for lean_theorem; the
// parser only ever captures bare declaration names, so the anchor must still
// resolve to Proven (not StaleRef), and the bare name must not also appear
// as an orphan.
func TestReconcile_ModuleJoinKeyResolves(t *testing.T) {
	t.Parallel()
	anchor := model.Anchor{
		ID:              "PROOF-GL-JOINKEY",
		Name:            "PROOF-GL-JOINKEY",
		Description:     "join-key test anchor",
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofLanguage:   testLeanLanguage,
		ProofFile:       "Gleipnir/OctonionRank/RankEighteen.lean",
		ProofState:      model.ProofStateWritten,
		SorryCount:      intPtr(0),
		LeanTheorem:     "Gleipnir.OctonionRank.RankEighteen::tensorRank_T_O_ge_eighteen",
		PredictionChain: []string{},
	}
	theorems := []leanlink.TheoremDecl{
		synthTheorem("tensorRank_T_O_ge_eighteen", "Gleipnir/OctonionRank/RankEighteen.lean", 0),
	}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	foundProven := false
	foundStale := false
	for _, cl := range r.Classifications {
		if cl.AnchorID != "PROOF-GL-JOINKEY" {
			continue
		}
		switch cl.Class {
		case leanlink.ClassProven:
			foundProven = true
		case leanlink.ClassStaleRef:
			foundStale = true
		}
	}
	if !foundProven {
		t.Errorf("expected ClassProven for module::theorem join key, got classifications: %v", r.Classifications)
	}
	if foundStale {
		t.Errorf("did not expect ClassStaleRef for a valid module::theorem join key, got: %v", r.Classifications)
	}
	for _, o := range r.Orphans {
		if o.Name == "tensorRank_T_O_ge_eighteen" {
			t.Errorf("tensorRank_T_O_ge_eighteen is wired via a join-key ref but still reported as orphan: %v", r.Orphans)
		}
	}
}

// 4c. Module::theorem join key with the WRONG module (CTH #124 fix-round
// finding, qbp-architecture): the module part of a join key is an identity
// claim, not decoration. A key whose module does not resolve to the
// anchor's proof_file must come out stale, not proven — otherwise the tool
// certifies an identity it never checked.
func TestReconcile_ModuleJoinKeyWrongModuleIsStale(t *testing.T) {
	t.Parallel()
	anchor := model.Anchor{
		ID:              "PROOF-GL-WRONGMOD",
		Name:            "PROOF-GL-WRONGMOD",
		Description:     "wrong-module join-key test anchor",
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofLanguage:   testLeanLanguage,
		ProofFile:       "Gleipnir/OctonionRank/RankEighteen.lean",
		ProofState:      model.ProofStateWritten,
		SorryCount:      intPtr(0),
		LeanTheorem:     "Gleipnir.Wrong.Module::tensorRank_T_O_ge_eighteen",
		PredictionChain: []string{},
	}
	theorems := []leanlink.TheoremDecl{
		synthTheorem("tensorRank_T_O_ge_eighteen", "Gleipnir/OctonionRank/RankEighteen.lean", 0),
	}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	foundProven := false
	foundStale := false
	for _, cl := range r.Classifications {
		if cl.AnchorID != "PROOF-GL-WRONGMOD" {
			continue
		}
		switch cl.Class {
		case leanlink.ClassProven:
			foundProven = true
		case leanlink.ClassStaleRef:
			foundStale = true
		}
	}
	if foundProven {
		t.Errorf("a wrong-module join key must not resolve to ClassProven, got: %v", r.Classifications)
	}
	if !foundStale {
		t.Errorf("expected ClassStaleRef for a wrong-module join key, got: %v", r.Classifications)
	}
}

// 4d. Module::theorem join key with an EXTRA LEADING SEGMENT (CTH #124,
// second fix round, qbp-architecture): a claimed module path must never be
// allowed to be longer than the real file's path just because the real
// path happens to be its tail. This is the exact case a reverted
// strings.HasSuffix(expected, file) mutant would wrongly accept — this test
// must kill that mutant.
func TestReconcile_ModuleJoinKeyExtraLeadingSegmentIsStale(t *testing.T) {
	t.Parallel()
	anchor := model.Anchor{
		ID:              "PROOF-GL-EXTRASEG",
		Name:            "PROOF-GL-EXTRASEG",
		Description:     "extra-leading-segment join-key test anchor",
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofLanguage:   testLeanLanguage,
		ProofFile:       "Gleipnir/OctonionRank/RankEighteen.lean",
		ProofState:      model.ProofStateWritten,
		SorryCount:      intPtr(0),
		LeanTheorem:     "Evil.Gleipnir.OctonionRank.RankEighteen::tensorRank_T_O_ge_eighteen",
		PredictionChain: []string{},
	}
	theorems := []leanlink.TheoremDecl{
		synthTheorem("tensorRank_T_O_ge_eighteen", "Gleipnir/OctonionRank/RankEighteen.lean", 0),
	}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	for _, cl := range r.Classifications {
		if cl.AnchorID != "PROOF-GL-EXTRASEG" {
			continue
		}
		if cl.Class == leanlink.ClassProven {
			t.Errorf("a module with an extra leading segment must not resolve to ClassProven, got: %v", r.Classifications)
		}
	}
	found := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-GL-EXTRASEG" && cl.Class == leanlink.ClassStaleRef {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ClassStaleRef for an extra-leading-segment module, got: %v", r.Classifications)
	}
}

// 4e. Module::theorem join key with a PARTIAL COMPONENT OVERLAP (CTH #124,
// second fix round): "Rank" must not match inside "OctonionRank" just
// because the tail characters happen to line up — the comparison has to
// respect path-component boundaries, not raw string suffixes.
func TestReconcile_ModuleJoinKeyPartialComponentOverlapIsStale(t *testing.T) {
	t.Parallel()
	anchor := model.Anchor{
		ID:              "PROOF-GL-PARTIALSEG",
		Name:            "PROOF-GL-PARTIALSEG",
		Description:     "partial-component-overlap join-key test anchor",
		Tier:            model.TierProof,
		Status:          model.StatusCoherent,
		ProvenanceKind:  model.ProvenanceKindProof,
		ProofLanguage:   testLeanLanguage,
		ProofFile:       "Gleipnir/OctonionRank/RankEighteen.lean",
		ProofState:      model.ProofStateWritten,
		SorryCount:      intPtr(0),
		LeanTheorem:     "Rank.RankEighteen::tensorRank_T_O_ge_eighteen",
		PredictionChain: []string{},
	}
	theorems := []leanlink.TheoremDecl{
		synthTheorem("tensorRank_T_O_ge_eighteen", "Gleipnir/OctonionRank/RankEighteen.lean", 0),
	}
	inv := makeInventory([]model.Anchor{anchor})

	r := leanlink.Reconcile(inv, theorems, leanlink.ToolchainSpec{})

	for _, cl := range r.Classifications {
		if cl.AnchorID != "PROOF-GL-PARTIALSEG" {
			continue
		}
		if cl.Class == leanlink.ClassProven {
			t.Errorf("a module matching only a partial path component must not resolve to ClassProven, got: %v", r.Classifications)
		}
	}
	found := false
	for _, cl := range r.Classifications {
		if cl.AnchorID == "PROOF-GL-PARTIALSEG" && cl.Class == leanlink.ClassStaleRef {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ClassStaleRef for a partial-component-overlap module, got: %v", r.Classifications)
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
