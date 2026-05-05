// Issue #14 acceptance: merging QBP v3.2 + QBP-Q v0.1 produces correct
// zero-theoretical-deficit classification; shared Tier 1 anchors have
// bridge fidelity 1.0. QBP v3.2 fixture is absent from testdata/;
// structural acceptance is encoded on synthetic inventories.
package compute

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func TestMergeProgrammes_LosslessAtTier1(t *testing.T) {
	// Two inventories share an anchor at Tier 1 in both. Theorem 2:
	// bridge fidelity 1.0, merge is lossless.
	a := model.Inventory{
		Programme: "A", Version: "1",
		Axioms:  []model.Axiom{{ID: "AX-A"}},
		Anchors: []model.Anchor{{ID: "P-shared", Tier: model.TierProof, Status: model.StatusCoherent}},
	}
	b := model.Inventory{
		Programme: "B", Version: "1",
		Axioms:  []model.Axiom{{ID: "AX-B"}},
		Anchors: []model.Anchor{{ID: "P-shared", Tier: model.TierProof, Status: model.StatusCoherent}},
	}

	merged, report := MergeProgrammes(a, b)

	if !report.Lossless {
		t.Errorf("expected lossless merge at shared Tier 1; report = %+v", report)
	}
	if len(report.BridgeEdges) != 1 {
		t.Fatalf("expected 1 bridge edge, got %d", len(report.BridgeEdges))
	}
	if report.BridgeEdges[0].Fidelity != 1.0 {
		t.Errorf("bridge fidelity = %v, want 1.0 (Theorem 2)", report.BridgeEdges[0].Fidelity)
	}
	if len(report.SharedAnchorIDs) != 1 || report.SharedAnchorIDs[0] != "P-shared" {
		t.Errorf("shared anchors = %v, want [P-shared]", report.SharedAnchorIDs)
	}
	if merged.Programme != "A+B" {
		t.Errorf("merged programme = %q, want A+B", merged.Programme)
	}
}

func TestMergeProgrammes_TierMinRule(t *testing.T) {
	// Anchor is Tier 1 in A, Tier 2 in B → merged Tier 1 (more trustworthy).
	a := model.Inventory{
		Programme: "A", Version: "1",
		Anchors: []model.Anchor{{ID: "X", Tier: model.TierProof, Status: model.StatusCoherent}},
	}
	b := model.Inventory{
		Programme: "B", Version: "1",
		Anchors: []model.Anchor{{ID: "X", Tier: model.TierMeasurement, Status: model.StatusCoherent}},
	}
	merged, _ := MergeProgrammes(a, b)
	for _, anc := range merged.Anchors {
		if anc.ID == "X" && anc.Tier != model.TierProof {
			t.Errorf("merged tier = %v, want TierProof (1)", anc.Tier)
		}
	}
}

func TestMergeProgrammes_StatusDisagreementIsIncoherent(t *testing.T) {
	a := model.Inventory{
		Programme: "A", Version: "1",
		Anchors: []model.Anchor{{ID: "Y", Tier: model.TierMeasurement, Status: model.StatusCoherent}},
	}
	b := model.Inventory{
		Programme: "B", Version: "1",
		Anchors: []model.Anchor{{ID: "Y", Tier: model.TierMeasurement, Status: model.StatusIncoherent}},
	}
	merged, report := MergeProgrammes(a, b)
	for _, anc := range merged.Anchors {
		if anc.ID == "Y" && anc.Status != model.StatusIncoherent {
			t.Errorf("merged status = %v, want StatusIncoherent", anc.Status)
		}
	}
	if len(report.IncoherentMerges) != 1 || report.IncoherentMerges[0] != "Y" {
		t.Errorf("IncoherentMerges = %v, want [Y]", report.IncoherentMerges)
	}
	if report.Lossless {
		t.Errorf("Lossless = true, want false (incoherent merge)")
	}
}

func TestMergeProgrammes_BridgeFidelityNotLosslessWhenMixedTier(t *testing.T) {
	a := model.Inventory{
		Programme: "A", Version: "1",
		Anchors: []model.Anchor{{ID: "Z", Tier: model.TierProof, Status: model.StatusCoherent}},
	}
	b := model.Inventory{
		Programme: "B", Version: "1",
		Anchors: []model.Anchor{{ID: "Z", Tier: model.TierMeasurement, Status: model.StatusCoherent}},
	}
	_, report := MergeProgrammes(a, b)
	if report.Lossless {
		t.Errorf("Lossless = true, want false (mixed-tier shared anchor)")
	}
	if report.BridgeEdges[0].Fidelity >= 1.0 {
		t.Errorf("bridge fidelity = %v, want < 1.0 (mixed tier)",
			report.BridgeEdges[0].Fidelity)
	}
}

func TestMergeProgrammes_UnionWhenNoShared(t *testing.T) {
	a := model.Inventory{
		Programme: "A", Version: "1",
		Anchors: []model.Anchor{{ID: "A1", Tier: model.TierProof}},
	}
	b := model.Inventory{
		Programme: "B", Version: "1",
		Anchors: []model.Anchor{{ID: "B1", Tier: model.TierProof}},
	}
	merged, report := MergeProgrammes(a, b)
	if len(merged.Anchors) != 2 {
		t.Errorf("merged anchors = %d, want 2 (union)", len(merged.Anchors))
	}
	if len(report.SharedAnchorIDs) != 0 {
		t.Errorf("SharedAnchorIDs should be empty: %v", report.SharedAnchorIDs)
	}
	if !report.Lossless {
		t.Errorf("no shared anchors → vacuously lossless; got %+v", report)
	}
}

func TestMergeProgrammes_DeficitClassification(t *testing.T) {
	// Mix of irreducible (theoretical) and measurable (engineering) inputs.
	a := model.Inventory{
		Programme: "A", Version: "1",
		Inputs: []model.Input{
			{ID: "INST-irreducible", Type: testInputType, Status: "irreducible", SignificantFigures: 3},
			{ID: "INST-measurable", Type: testInputType, Status: testInputStatus, SignificantFigures: 3},
		},
	}
	b := model.Inventory{Programme: "B", Version: "1"}
	_, report := MergeProgrammes(a, b)
	if report.TheoreticalDeficit <= 0 {
		t.Errorf("TheoreticalDeficit = %v, want > 0 (irreducible input)", report.TheoreticalDeficit)
	}
	if report.EngineeringDeficit <= 0 {
		t.Errorf("EngineeringDeficit = %v, want > 0 (measurable input)", report.EngineeringDeficit)
	}
}

func TestMergeProgrammes_QBPQuantum_v01_v02(t *testing.T) {
	// Acceptance shadow: merging two QBP-Quantum versions runs without
	// crashing and produces a populated merge report. The QBP v3.2
	// fixture is absent so we substitute QBP-Q v0.1 + v0.2.
	a := loadFixture(t, "qbp_quantum_v0_1.json")
	b := loadFixture(t, "qbp_quantum_v0_2.json")
	merged, report := MergeProgrammes(a, b)
	if merged.Programme == "" {
		t.Errorf("merged Programme empty")
	}
	if len(merged.Anchors) == 0 {
		t.Errorf("merged Anchors empty")
	}
	// At least one shared anchor between QBP-Q v0.1 and v0.2 — same
	// programme, just different versions.
	if len(report.SharedAnchorIDs) == 0 {
		t.Errorf("expected at least one shared anchor between QBP-Q versions")
	}
}
