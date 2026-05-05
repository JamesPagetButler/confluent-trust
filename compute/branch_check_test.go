// Issue #25 acceptance: detects cross-branch contamination in a fixture
// with an intentionally planted violation; zero violations on a clean
// fixture. The DM fork fixture in testdata/qbp_dm_fork.json contains
// FLAG-rotcurve-B-uses-A as the planted violation (Branch B anchor
// pulling INPUT-alpha0-correction, a Branch A input).
package compute

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func TestCheckBranchConsistency_CleanAnchor(t *testing.T) {
	anchor := model.Anchor{
		ID:              "PRED-clean",
		PredictionChain: []string{"AX-shared", "INPUT-mine"},
	}
	branch := model.Branch{
		ID: "branch-a", Burden: model.BurdenMinimal,
		Inputs: []string{"INPUT-mine"},
	}
	shared := []string{"AX-shared"}

	if got := CheckBranchConsistency(anchor, branch, shared); len(got) != 0 {
		t.Errorf("clean anchor produced %d violations: %v", len(got), got)
	}
}

func TestCheckBranchConsistency_ContaminatedFromOtherBranch(t *testing.T) {
	anchor := model.Anchor{
		ID:              "PRED-bad",
		PredictionChain: []string{"AX-shared", "INPUT-other"},
	}
	branch := model.Branch{
		ID: "branch-a", Burden: model.BurdenMinimal,
		Inputs: []string{"INPUT-mine"},
	}
	shared := []string{"AX-shared"}

	got := CheckBranchConsistency(anchor, branch, shared)
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(got), got)
	}
	if got[0].AnchorID != "PRED-bad" || got[0].InputID != "INPUT-other" {
		t.Errorf("violation = %+v", got[0])
	}
}

func TestCheckBranchConsistency_SharedPrefixIsAllowed(t *testing.T) {
	// Same anchor used with different sharedPrefix lists: declaring
	// the input as shared eliminates the violation.
	anchor := model.Anchor{
		ID:              "PRED-x",
		PredictionChain: []string{"INPUT-debatable"},
	}
	branch := model.Branch{ID: "branch-a", Burden: model.BurdenMinimal}

	if got := CheckBranchConsistency(anchor, branch, nil); len(got) != 1 {
		t.Errorf("with no shared prefix: expected 1 violation, got %d", len(got))
	}
	if got := CheckBranchConsistency(anchor, branch, []string{"INPUT-debatable"}); len(got) != 0 {
		t.Errorf("with input in shared prefix: expected 0 violations, got %d", len(got))
	}
}

func TestCheckAllAnchors_DMFork_FlagsPlantedViolation(t *testing.T) {
	inv := loadFixture(t, "qbp_dm_fork.json")
	if len(inv.ForkPoints) == 0 {
		t.Fatal("qbp_dm_fork fixture has no fork_points; can't run #25 acceptance")
	}
	fork := inv.ForkPoints[0]

	violations := CheckAllAnchors(inv, fork)

	// The fixture's FLAG-rotcurve-B-uses-A is on branch-dm-exists and
	// pulls INPUT-alpha0-correction (a branch-no-dm input). Issue #25's
	// CheckAllAnchors must surface exactly this.
	dmExists := violations["branch-dm-exists"]
	var found bool
	for _, v := range dmExists {
		if v.AnchorID == "FLAG-rotcurve-B-uses-A" && v.InputID == "INPUT-alpha0-correction" {
			found = true
			if v.InputBranch != "branch-no-dm" {
				t.Errorf("InputBranch = %q, want branch-no-dm", v.InputBranch)
			}
		}
	}
	if !found {
		t.Errorf("planted FLAG-rotcurve-B-uses-A violation not detected; got %d violations on branch-dm-exists",
			len(dmExists))
	}
}

func TestCheckAllAnchors_DMFork_CleanBranchHasNoViolations(t *testing.T) {
	inv := loadFixture(t, "qbp_dm_fork.json")
	fork := inv.ForkPoints[0]

	violations := CheckAllAnchors(inv, fork)
	noDM := violations["branch-no-dm"]
	if len(noDM) != 0 {
		t.Errorf("branch-no-dm has unexpected violations: %v", noDM)
	}
}
