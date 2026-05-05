// Issue #24 acceptance: for the dark matter fork fixture, Branch A
// (no DM, Minimal) has lower information deficit than Branch B
// (DM exists, Extended). Both share the algebraic core. ρ_net differs.
package compute

import (
	"math"
	"testing"
)

func TestBranchHealth_DMFork_BranchAHasLowerDeficit(t *testing.T) {
	inv := loadFixture(t, "qbp_dm_fork.json")
	if len(inv.ForkPoints) == 0 {
		t.Fatal("fixture missing fork_points")
	}
	fork := inv.ForkPoints[0]

	noDM := BranchHealth(inv, fork, "branch-no-dm", nil)
	dm := BranchHealth(inv, fork, "branch-dm-exists", nil)

	if noDM.InformationDeficit >= dm.InformationDeficit {
		t.Errorf("acceptance: Branch A deficit (%v) should be less than Branch B (%v)",
			noDM.InformationDeficit, dm.InformationDeficit)
	}
}

func TestBranchHealth_DMFork_RhoNetDiffersBetweenBranches(t *testing.T) {
	inv := loadFixture(t, "qbp_dm_fork.json")
	fork := inv.ForkPoints[0]

	noDM := BranchHealth(inv, fork, "branch-no-dm", nil)
	dm := BranchHealth(inv, fork, "branch-dm-exists", nil)

	// Acceptance: ρ_net differs. Use a non-zero tolerance — both could be
	// 0 if the fixture had zero confirmed information; verify they're
	// finite and not both equal to zero.
	if math.Abs(noDM.RhoNet-dm.RhoNet) < 1e-12 {
		t.Errorf("ρ_net should differ between branches: noDM=%v dm=%v",
			noDM.RhoNet, dm.RhoNet)
	}
	for _, v := range []float64{noDM.RhoNet, dm.RhoNet} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			t.Errorf("ρ_net non-finite: %v", v)
		}
	}
}

func TestBranchHealth_AnchorAndConfluenceCounts(t *testing.T) {
	inv := loadFixture(t, "qbp_dm_fork.json")
	fork := inv.ForkPoints[0]

	noDM := BranchHealth(inv, fork, "branch-no-dm", nil)
	dm := BranchHealth(inv, fork, "branch-dm-exists", nil)

	// Both branches see the shared prefix anchors plus their own.
	// Counts must be > 0 if the fixture is well-formed.
	if noDM.AnchorCount == 0 {
		t.Errorf("branch-no-dm AnchorCount = 0")
	}
	if dm.AnchorCount == 0 {
		t.Errorf("branch-dm-exists AnchorCount = 0")
	}
}

func TestBranchHealth_UnknownBranchReturnsZeroFilled(t *testing.T) {
	inv := loadFixture(t, "qbp_dm_fork.json")
	fork := inv.ForkPoints[0]

	r := BranchHealth(inv, fork, "branch-does-not-exist", nil)
	if r.AnchorCount != 0 || r.RhoNet != 0 {
		t.Errorf("unknown branch should return zero-filled result, got %+v", r)
	}
}

func TestCompareBranches_DMFork_BranchALowerDeficit(t *testing.T) {
	// CompareBranches should also identify branch-no-dm as
	// lower-deficit (the issue acceptance phrased as a comparison).
	inv := loadFixture(t, "qbp_dm_fork.json")
	fork := inv.ForkPoints[0]

	cmp := CompareBranches(inv, fork, nil)
	if cmp.ForkID != fork.ID {
		t.Errorf("ForkID = %q, want %q", cmp.ForkID, fork.ID)
	}
	if len(cmp.Branches) != len(fork.Branches) {
		t.Errorf("Branches len = %d, want %d", len(cmp.Branches), len(fork.Branches))
	}
	if cmp.LowerDeficitBranch != "branch-no-dm" {
		t.Errorf("LowerDeficitBranch = %q, want branch-no-dm", cmp.LowerDeficitBranch)
	}
}
