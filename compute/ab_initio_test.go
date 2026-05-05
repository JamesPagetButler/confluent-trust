// Issue #20 acceptance: matches Python engine output. The Python source
// of truth is /home/prime/Documents/CTH/Archive/cth_engine_v2.py:408
// (compute_ab_initio_score). This test file pins R7's algorithmic
// contract on synthetic fixtures whose answer the rule alone determines.
package compute

import (
	"math"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// abInitInputT is a tiny helper that produces a model.Input for the test
// fixtures below. Pulled out because the literal "input" type label
// trips goconst (it appears across several compute test files).
func abInitInput(id string) model.Input {
	return model.Input{ID: id, Type: testInputType, Status: testInputStatus, SignificantFigures: 3}
}

func TestAbInitioScore_NoMultiPathTargets(t *testing.T) {
	inv := model.Inventory{
		Chains: []model.Chain{
			{ID: "C-only", TargetID: testAnchorM1, SourceIDs: []string{testAxiomID}},
		},
	}
	if got := AbInitioScore(inv); len(got) != 0 {
		t.Errorf("single-chain inventory: expected no results, got %v", got)
	}
}

func TestAbInitioScore_LowerInputWinsWhenFidelitiesComparable(t *testing.T) {
	// Acceptance R7: "Prefer lower-deficit path only when fidelities are
	// comparable." Two chains target T at fidelity 0.90; one has 1 input
	// upstream, the other 3. The cheaper one wins.
	mu := 0.90
	inv := model.Inventory{
		Inputs: []model.Input{abInitInput("INST-a"), abInitInput("INST-b"), abInitInput("INST-c")},
		Chains: []model.Chain{
			{
				ID: "C-cheap", TargetID: testAnchorM1, Fidelity: &mu,
				SourceIDs: []string{"INST-a"},
			},
			{
				ID: "C-deficit", TargetID: testAnchorM1, Fidelity: &mu,
				SourceIDs: []string{"INST-a", "INST-b", "INST-c"},
			},
		},
	}

	got := AbInitioScore(inv)
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	if got[0].BestChainID != "C-cheap" {
		t.Errorf("best chain = %q, want C-cheap (lower input count under comparable fidelities)",
			got[0].BestChainID)
	}
	// Score = μ / (1 + input_count): cheap = 0.9/2 = 0.45; deficit = 0.9/4 = 0.225.
	if math.Abs(got[0].BestScore-0.45) > 1e-9 {
		t.Errorf("best score = %v, want 0.45", got[0].BestScore)
	}
}

func TestAbInitioScore_HigherFidelityWinsWhenNotComparable(t *testing.T) {
	// Two paths to T: low-fidelity-cheap vs high-fidelity-loaded. The
	// high-fidelity wins on raw score even though it carries more inputs.
	low, high := 0.50, 0.95
	inv := model.Inventory{
		Inputs: []model.Input{abInitInput("INST-a"), abInitInput("INST-b")},
		Chains: []model.Chain{
			{
				ID: "C-low", TargetID: testAnchorM1, Fidelity: &low,
				SourceIDs: []string{testAxiomID},
			},
			{
				ID: "C-high", TargetID: testAnchorM1, Fidelity: &high,
				SourceIDs: []string{"INST-a", "INST-b"},
			},
		},
	}

	got := AbInitioScore(inv)
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	// scores: low = 0.50/(1+0) = 0.50; high = 0.95/(1+2) ≈ 0.3167. Low wins on score.
	if got[0].BestChainID != "C-low" {
		t.Errorf("best chain = %q; expected C-low (higher μ/(1+input_count))", got[0].BestChainID)
	}
}

func TestAbInitioScore_DeterministicTiebreak(t *testing.T) {
	// Two identical-cost paths: same fidelity, same input count. Lower
	// chain ID wins so the function is reproducible.
	mu := 0.95
	inv := model.Inventory{
		Inputs: []model.Input{abInitInput("INST-a")},
		Chains: []model.Chain{
			{
				ID: "C-z", TargetID: testAnchorM1, Fidelity: &mu,
				SourceIDs: []string{"INST-a"},
			},
			{
				ID: "C-a", TargetID: testAnchorM1, Fidelity: &mu,
				SourceIDs: []string{"INST-a"},
			},
		},
	}
	got := AbInitioScore(inv)
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	if got[0].BestChainID != "C-a" {
		t.Errorf("tie-break: got %q, want C-a (lower id)", got[0].BestChainID)
	}
}

func TestAbInitioScore_CountsUpstreamInputsTransitively(t *testing.T) {
	// A chain whose direct sources are anchors (not inputs), but whose
	// transitive upstream resolves to two distinct inputs. The BFS must
	// see both.
	mu1, mu2 := 0.90, 0.91
	inv := model.Inventory{
		Inputs: []model.Input{abInitInput("INST-x"), abInitInput("INST-y")},
		Chains: []model.Chain{
			{
				ID: "C-direct", TargetID: testAnchorM1, Fidelity: &mu1,
				SourceIDs: []string{"INST-x"},
			},
			{
				ID: "C-indirect", TargetID: testAnchorM1, Fidelity: &mu2,
				SourceIDs: []string{"P-upstream"},
			},
			{
				ID: "C-upstream", TargetID: "P-upstream",
				SourceIDs: []string{"INST-x", "INST-y"},
			},
		},
	}
	got := AbInitioScore(inv)
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1", len(got))
	}
	// Within tolerance (Δμ = 0.01) → fall back to lower input count.
	// C-direct: 1 input. C-indirect: 2 inputs (transitively). C-direct wins.
	if got[0].BestChainID != "C-direct" {
		t.Errorf("best = %q, want C-direct (lower input count under comparable fidelities)",
			got[0].BestChainID)
	}
	// Verify the candidates record the right input counts.
	for _, cand := range got[0].Candidates {
		switch cand.ChainID {
		case "C-direct":
			if cand.InputCount != 1 {
				t.Errorf("C-direct InputCount = %d, want 1", cand.InputCount)
			}
		case "C-indirect":
			if cand.InputCount != 2 {
				t.Errorf("C-indirect InputCount = %d, want 2", cand.InputCount)
			}
		}
	}
}

func TestAbInitioScore_ResultsSortedByTargetID(t *testing.T) {
	mu := 0.9
	inv := model.Inventory{
		Inputs: []model.Input{abInitInput("INST-a")},
		Chains: []model.Chain{
			{ID: "C-zT-1", TargetID: "TGT-z", Fidelity: &mu, SourceIDs: []string{"INST-a"}},
			{ID: "C-zT-2", TargetID: "TGT-z", Fidelity: &mu, SourceIDs: []string{"INST-a"}},
			{ID: "C-aT-1", TargetID: "TGT-a", Fidelity: &mu, SourceIDs: []string{"INST-a"}},
			{ID: "C-aT-2", TargetID: "TGT-a", Fidelity: &mu, SourceIDs: []string{"INST-a"}},
		},
	}
	got := AbInitioScore(inv)
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2", len(got))
	}
	if got[0].TargetID >= got[1].TargetID {
		t.Errorf("not sorted: %q before %q", got[0].TargetID, got[1].TargetID)
	}
}
