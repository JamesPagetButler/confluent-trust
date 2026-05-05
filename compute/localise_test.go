package compute

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Acceptance line from Issue #13:
//
//   For FLAG-J in QBP v3.2, localises error to [U_dd → J] segment with
//   t_pd as weakest link.
//
// QBP v3.2 fixture (qbp_v3_2.json) is not yet committed, so the
// FLAG-J / U_dd / t_pd assertion is encoded against a synthetic
// inventory whose shape matches the spec line in Issue #13:
//
//   AX-1 (axiom)
//      → P-1 (proof, confluence target, coherent)
//          → P-2 (proof, incoming-chain fidelity 0.95, coherent)
//              → P-3 (proof, incoming-chain fidelity 0.7, coherent)
//                  → BAD (incoherent)
//
// LocaliseIncoherence("BAD") must return:
//   SegmentStart           == "P-1"
//   SegmentEnd             == "BAD"
//   LastCoherentConfluence == "P-1"
//   WeakestLinkID          == "P-3"   (lowest-fidelity incoming chain)
//   SegmentLength          == 3       (P-1 → P-2 → P-3 → BAD)

const (
	locP1  = "P-1"
	locP2  = "P-2"
	locP3  = "P-3"
	locBad = "BAD"
)

func buildLocaliseFixture() model.Inventory {
	zero := 0
	fid95 := 0.95
	fid70 := 0.70
	// Failing anchor's incoming chain has higher fidelity than P-3's: the
	// failure is not in BAD's own derivation step but in something
	// upstream. This matches the Issue #13 acceptance: "t_pd as weakest
	// link" identifies an *earlier* link than the FLAG anchor.
	fidBad := 0.99

	return model.Inventory{
		Programme: "synthetic",
		Version:   "0.0.1",
		Axioms:    []model.Axiom{{ID: testAxiomID}},
		Anchors: []model.Anchor{
			{ID: locP1, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero,
				PredictionChain: []string{testAxiomID}},
			{ID: locP2, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero,
				PredictionChain: []string{locP1}},
			{ID: locP3, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero,
				PredictionChain: []string{locP2}},
			{ID: locBad, Tier: model.TierProof, Status: model.StatusIncoherent,
				PredictionChain: []string{locP3}},
		},
		Chains: []model.Chain{
			{ID: "C-AX-P1", SourceIDs: []string{testAxiomID}, TargetID: locP1, Steps: 1, Status: model.StatusCoherent},
			{ID: "C-P1-P2", SourceIDs: []string{locP1}, TargetID: locP2, Fidelity: &fid95, Steps: 1, Status: model.StatusCoherent},
			{ID: "C-P2-P3", SourceIDs: []string{locP2}, TargetID: locP3, Fidelity: &fid70, Steps: 1, Status: model.StatusCoherent},
			{ID: "C-P3-BAD", SourceIDs: []string{locP3}, TargetID: locBad, Fidelity: &fidBad, Steps: 1, Status: model.StatusIncoherent},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{ID: "CONF-P1", AnchorID: locP1, Status: model.StatusCoherent},
		},
	}
}

func TestLocaliseIncoherence_AcceptanceShape(t *testing.T) {
	// Encodes the Issue #13 acceptance line on the synthetic fixture.
	inv := buildLocaliseFixture()
	got := LocaliseIncoherence(locBad, inv)

	if got.AnchorID != locBad {
		t.Errorf("AnchorID: got %q, want %q", got.AnchorID, locBad)
	}
	if got.SegmentStart != locP1 {
		t.Errorf("SegmentStart: got %q, want %q (LastCoherentConfluence)", got.SegmentStart, locP1)
	}
	if got.SegmentEnd != locBad {
		t.Errorf("SegmentEnd: got %q, want %q", got.SegmentEnd, locBad)
	}
	if got.LastCoherentConfluence != locP1 {
		t.Errorf("LastCoherentConfluence: got %q, want %q", got.LastCoherentConfluence, locP1)
	}
	// Weakest link = segment anchor whose incoming chain has the lowest
	// ChainFidelity. Incoming-chain fidelities in segment:
	//   P-1: incoming chain has no Fidelity → 1.0
	//   P-2: 0.95
	//   P-3: 0.70   ← min
	//   BAD: 0.99   (high — failure was upstream, not in BAD's own step)
	if got.WeakestLinkID != locP3 {
		t.Errorf("WeakestLinkID: got %q, want %q (lowest incoming fidelity in [P-1..BAD])",
			got.WeakestLinkID, locP3)
	}
	// Segment length: hops from P-1 to BAD via P-2, P-3 = 3.
	if got.SegmentLength != 3 {
		t.Errorf("SegmentLength: got %d, want 3", got.SegmentLength)
	}
}

func TestLocaliseIncoherence_NoCheckpoint(t *testing.T) {
	// No confluence anywhere in the ancestry. SegmentStart should be
	// the deepest ancestor reached; LastCoherentConfluence empty.
	zero := 0
	fid70 := 0.70
	inv := model.Inventory{
		Programme: "synthetic",
		Version:   "0.0.1",
		Axioms:    []model.Axiom{{ID: testAxiomID}},
		Anchors: []model.Anchor{
			{ID: locP1, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero,
				PredictionChain: []string{testAxiomID}},
			{ID: locBad, Tier: model.TierProof, Status: model.StatusIncoherent,
				PredictionChain: []string{locP1}},
		},
		Chains: []model.Chain{
			{ID: "C-AX-P1", SourceIDs: []string{testAxiomID}, TargetID: locP1, Steps: 1, Status: model.StatusCoherent},
			{ID: "C-P1-BAD", SourceIDs: []string{locP1}, TargetID: locBad, Fidelity: &fid70, Steps: 1, Status: model.StatusIncoherent},
		},
	}
	got := LocaliseIncoherence(locBad, inv)
	if got.LastCoherentConfluence != "" {
		t.Errorf("LastCoherentConfluence: got %q, want empty", got.LastCoherentConfluence)
	}
	// Deepest ancestor reached is the axiom AX-1.
	if got.SegmentStart != testAxiomID {
		t.Errorf("SegmentStart: got %q, want %q (deepest ancestor)", got.SegmentStart, testAxiomID)
	}
	if got.SegmentEnd != locBad {
		t.Errorf("SegmentEnd: got %q, want %q", got.SegmentEnd, locBad)
	}
	// Hops: AX-1 → P-1 → BAD = 2.
	if got.SegmentLength != 2 {
		t.Errorf("SegmentLength: got %d, want 2", got.SegmentLength)
	}
	// Weakest link in [AX-1, P-1, BAD]: AX-1 has no incoming chain
	// (it's an axiom, no chain targets it); P-1 has fidelity 1.0
	// (no Fidelity set, no StepTypes); BAD has 0.70. So WeakestLinkID
	// is BAD.
	if got.WeakestLinkID != locBad {
		t.Errorf("WeakestLinkID: got %q, want %q", got.WeakestLinkID, locBad)
	}
}

func TestLocaliseIncoherence_OrphanAnchorIsSingletonSegment(t *testing.T) {
	// Anchor with no prediction_chain at all — the only "ancestor" is
	// itself. SegmentLength must be 0 and WeakestLinkID = anchor ID.
	inv := model.Inventory{
		Programme: "synthetic",
		Version:   "0.0.1",
		Anchors: []model.Anchor{
			{ID: locBad, Tier: model.TierProof, Status: model.StatusIncoherent},
		},
	}
	got := LocaliseIncoherence(locBad, inv)
	if got.SegmentStart != locBad {
		t.Errorf("SegmentStart: got %q, want %q", got.SegmentStart, locBad)
	}
	if got.SegmentEnd != locBad {
		t.Errorf("SegmentEnd: got %q, want %q", got.SegmentEnd, locBad)
	}
	if got.WeakestLinkID != locBad {
		t.Errorf("WeakestLinkID: got %q, want %q", got.WeakestLinkID, locBad)
	}
	if got.SegmentLength != 0 {
		t.Errorf("SegmentLength: got %d, want 0", got.SegmentLength)
	}
	if got.LastCoherentConfluence != "" {
		t.Errorf("LastCoherentConfluence: got %q, want empty", got.LastCoherentConfluence)
	}
}

func TestLocaliseIncoherence_TiebreakAlphabetical(t *testing.T) {
	// Two anchors in the segment with identical lowest fidelity must
	// resolve to the alphabetically smaller ID.
	zero := 0
	fid70 := 0.70

	inv := model.Inventory{
		Programme: "synthetic",
		Version:   "0.0.1",
		Anchors: []model.Anchor{
			{ID: locP1, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero},
			{ID: locP2, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero,
				PredictionChain: []string{locP1}},
			{ID: locP3, Tier: model.TierProof, Status: model.StatusCoherent, SorryCount: &zero,
				PredictionChain: []string{locP2}},
			{ID: locBad, Tier: model.TierProof, Status: model.StatusIncoherent,
				PredictionChain: []string{locP3}},
		},
		Chains: []model.Chain{
			{ID: "C-P1-P2", SourceIDs: []string{locP1}, TargetID: locP2, Fidelity: &fid70, Steps: 1, Status: model.StatusCoherent},
			{ID: "C-P2-P3", SourceIDs: []string{locP2}, TargetID: locP3, Fidelity: &fid70, Steps: 1, Status: model.StatusCoherent},
			// BAD's incoming chain is high-fidelity so the tie is
			// strictly between P-2 and P-3.
			{ID: "C-P3-BAD", SourceIDs: []string{locP3}, TargetID: locBad, Steps: 1, Status: model.StatusIncoherent},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{ID: "CONF-P1", AnchorID: locP1, Status: model.StatusCoherent},
		},
	}
	got := LocaliseIncoherence(locBad, inv)
	if got.WeakestLinkID != locP2 {
		t.Errorf("WeakestLinkID tiebreak: got %q, want %q (alphabetical)", got.WeakestLinkID, locP2)
	}
}
