package compute

import "github.com/JamesPagetButler/confluent-trust/model"

// BranchHealthResult is the per-branch health metric set produced by
// BranchHealth per Theory v0.2 Definition 20. ρ_net is computed using
// the shared prefix ∪ branch-specific anchors; the deficit is split
// into theoretical (axiom-entropy + Tier-2 anchor entropy not yet
// confirmed by structural match) and engineering (input entropy).
type BranchHealthResult struct {
	BranchID                string
	RhoNet                  float64
	RhoNetSensitivityHalfH  float64
	RhoNetSensitivityBaseH  float64
	RhoNetSensitivityDouble float64
	InformationDeficit      float64
	TheoreticalDeficit      float64
	EngineeringDeficit      float64
	CoherenceRatio          float64
	AnchorCount             int
	ConfluenceCount         int
}

// ForkComparison is the output of CompareBranches: a side-by-side view
// of every branch under a fork plus convenience pointers to the
// lower-deficit and higher-ρ_net winners.
type ForkComparison struct {
	ForkID             string
	LowerDeficitBranch string
	HigherRhoNetBranch string
	Branches           []BranchHealthResult
}

// BranchHealth computes the metrics for one branch of a fork. The set
// {shared prefix anchors} ∪ {branch-specific anchors} is treated as a
// virtual sub-inventory; ρ_net, deficit, and coherence ratio are
// computed over that set. Inputs and chains follow the same union rule.
//
// axiomEntropy is consulted via the existing AxiomEntropy helper; pass
// nil to use DefaultAxiomEntropyBits everywhere.
func BranchHealth(
	inv model.Inventory,
	fork model.ForkPoint,
	branchID string,
	axiomEntropy map[string]float64,
) BranchHealthResult {
	branch, ok := findBranch(fork, branchID)
	if !ok {
		return BranchHealthResult{BranchID: branchID}
	}

	sub := branchSubInventory(inv, fork, branch)

	netRho, _ := NetCompression(sub, axiomEntropy)
	half, base, double := SensitivityBracket(sub, axiomEntropy)
	deficit := InformationDeficit(sub)
	theoretical := AxiomEntropySum(sub, axiomEntropy) + tier2UnconfirmedEntropy(sub)
	coherence := coherenceRatio(sub, axiomEntropy)

	return BranchHealthResult{
		BranchID:                branch.ID,
		RhoNet:                  netRho,
		RhoNetSensitivityHalfH:  half,
		RhoNetSensitivityBaseH:  base,
		RhoNetSensitivityDouble: double,
		InformationDeficit:      deficit,
		TheoreticalDeficit:      theoretical,
		EngineeringDeficit:      deficit,
		CoherenceRatio:          coherence,
		AnchorCount:             len(sub.Anchors),
		ConfluenceCount:         len(sub.ConfluencePoints),
	}
}

// CompareBranches runs BranchHealth across every branch of fork and
// returns a comparison record naming the branch with the lowest
// information deficit (the more "minimal" winner) and the branch with
// the highest ρ_net (the better "compressor"). Ties go to the
// lexicographically-lower branch ID for determinism.
func CompareBranches(
	inv model.Inventory,
	fork model.ForkPoint,
	axiomEntropy map[string]float64,
) ForkComparison {
	cmp := ForkComparison{ForkID: fork.ID}
	if len(fork.Branches) == 0 {
		return cmp
	}

	cmp.Branches = make([]BranchHealthResult, 0, len(fork.Branches))
	for _, b := range fork.Branches {
		cmp.Branches = append(cmp.Branches, BranchHealth(inv, fork, b.ID, axiomEntropy))
	}

	// Lower-deficit winner.
	cmp.LowerDeficitBranch = cmp.Branches[0].BranchID
	for _, r := range cmp.Branches[1:] {
		current := findResult(cmp.Branches, cmp.LowerDeficitBranch)
		if r.InformationDeficit < current.InformationDeficit ||
			(r.InformationDeficit == current.InformationDeficit && r.BranchID < cmp.LowerDeficitBranch) {
			cmp.LowerDeficitBranch = r.BranchID
		}
	}

	// Higher-ρ_net winner.
	cmp.HigherRhoNetBranch = cmp.Branches[0].BranchID
	for _, r := range cmp.Branches[1:] {
		current := findResult(cmp.Branches, cmp.HigherRhoNetBranch)
		if r.RhoNet > current.RhoNet ||
			(r.RhoNet == current.RhoNet && r.BranchID < cmp.HigherRhoNetBranch) {
			cmp.HigherRhoNetBranch = r.BranchID
		}
	}

	return cmp
}

// branchSubInventory builds the per-branch sub-inventory: shared-prefix
// nodes plus this branch's anchors/inputs/chains/confluences.
func branchSubInventory(inv model.Inventory, fork model.ForkPoint, branch model.Branch) model.Inventory {
	sharedSet := make(map[string]struct{}, len(fork.SharedPrefix))
	for _, id := range fork.SharedPrefix {
		sharedSet[id] = struct{}{}
	}
	branchAnchorSet := make(map[string]struct{}, len(branch.Anchors))
	for _, id := range branch.Anchors {
		branchAnchorSet[id] = struct{}{}
	}
	branchInputSet := make(map[string]struct{}, len(branch.Inputs))
	for _, id := range branch.Inputs {
		branchInputSet[id] = struct{}{}
	}
	branchChainSet := make(map[string]struct{}, len(branch.Chains))
	for _, id := range branch.Chains {
		branchChainSet[id] = struct{}{}
	}
	branchConfSet := make(map[string]struct{}, len(branch.Confluences))
	for _, id := range branch.Confluences {
		branchConfSet[id] = struct{}{}
	}

	sub := model.Inventory{
		Programme: inv.Programme,
		Version:   inv.Version,
	}

	for _, a := range inv.Axioms {
		if _, sh := sharedSet[a.ID]; sh {
			sub.Axioms = append(sub.Axioms, a)
		}
	}
	for _, a := range inv.Anchors {
		_, sh := sharedSet[a.ID]
		_, br := branchAnchorSet[a.ID]
		if sh || br {
			sub.Anchors = append(sub.Anchors, a)
		}
	}
	for _, in := range inv.Inputs {
		_, sh := sharedSet[in.ID]
		_, br := branchInputSet[in.ID]
		if sh || br {
			sub.Inputs = append(sub.Inputs, in)
		}
	}
	for _, c := range inv.Chains {
		_, br := branchChainSet[c.ID]
		// Include chain if it's branch-owned, OR if its target is in the
		// shared prefix (those chains are part of the shared topology).
		_, shTarget := sharedSet[c.TargetID]
		if br || shTarget {
			sub.Chains = append(sub.Chains, c)
		}
	}
	for _, cp := range inv.ConfluencePoints {
		_, br := branchConfSet[cp.ID]
		_, shAnchor := sharedSet[cp.AnchorID]
		if br || shAnchor {
			sub.ConfluencePoints = append(sub.ConfluencePoints, cp)
		}
	}
	return sub
}

func findBranch(fork model.ForkPoint, id string) (model.Branch, bool) {
	for _, b := range fork.Branches {
		if b.ID == id {
			return b, true
		}
	}
	return model.Branch{}, false
}

func findResult(rs []BranchHealthResult, id string) BranchHealthResult {
	for _, r := range rs {
		if r.BranchID == id {
			return r
		}
	}
	return BranchHealthResult{}
}

// tier2UnconfirmedEntropy is the residual entropy of Tier-2 anchors that
// did not confirm a structural match (δ != 0). They contribute to the
// "theoretical" deficit slice even though they are not axioms.
func tier2UnconfirmedEntropy(inv model.Inventory) float64 {
	var total float64
	for _, a := range inv.Anchors {
		if a.Tier != model.TierMeasurement {
			continue
		}
		total += ResidualEntropy(a, nil)
	}
	return total
}

// coherenceRatio = 1 - (incoherent entropy / total entropy across all
// nodes). Mirrors §4.7's R_c definition.
func coherenceRatio(inv model.Inventory, axiomEntropy map[string]float64) float64 {
	var totalEntropy, incoherentEntropy float64
	for _, a := range inv.Axioms {
		totalEntropy += AxiomEntropy(a, axiomEntropy)
	}
	for _, a := range inv.Anchors {
		eta := ResidualEntropy(a, axiomEntropy)
		totalEntropy += eta
		if a.Status == model.StatusIncoherent {
			incoherentEntropy += eta
		}
	}
	if totalEntropy <= 0 {
		return 1.0
	}
	return 1.0 - (incoherentEntropy / totalEntropy)
}
