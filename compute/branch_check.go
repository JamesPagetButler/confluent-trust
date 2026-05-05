package compute

import (
	"fmt"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// ConsistencyViolation is one branch-inconsistency finding: an anchor on
// branch X whose prediction_chain pulls an input that belongs to a
// different branch Y. Theory v0.2 Definition 21 / §2.8 anti-drift
// mechanism.
type ConsistencyViolation struct {
	AnchorID    string
	InputID     string
	InputBranch string
	Description string
}

// CheckBranchConsistency reports every input in anchor.PredictionChain
// that does not belong to branch (the anchor's own branch) or
// sharedPrefix. The fork's other branches' inputs are surfaced in
// InputBranch when known; otherwise InputBranch is "" with a
// "not in this branch or shared prefix" description.
func CheckBranchConsistency(
	anchor model.Anchor,
	branch model.Branch,
	sharedPrefix []string,
) []ConsistencyViolation {
	if len(anchor.PredictionChain) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(branch.Inputs)+len(branch.Anchors)+len(sharedPrefix))
	for _, id := range branch.Inputs {
		allowed[id] = struct{}{}
	}
	for _, id := range branch.Anchors {
		allowed[id] = struct{}{}
	}
	for _, id := range sharedPrefix {
		allowed[id] = struct{}{}
	}

	var violations []ConsistencyViolation
	for _, dep := range anchor.PredictionChain {
		if _, ok := allowed[dep]; ok {
			continue
		}
		violations = append(violations, ConsistencyViolation{
			AnchorID:    anchor.ID,
			InputID:     dep,
			InputBranch: "", // filled by CheckAllAnchors when fork context is available
			Description: fmt.Sprintf("anchor %s on branch %s depends on %s, which is not in the shared prefix or this branch",
				anchor.ID, branch.ID, dep),
		})
	}
	return violations
}

// CheckAllAnchors runs CheckBranchConsistency over every anchor in every
// branch of fork. The result is keyed by branch ID; each value is the
// flat list of violations for that branch's anchors. InputBranch is
// populated with the offending input's actual branch when locatable.
//
// The anchor list for a branch is resolved by id: branch.Anchors lists
// the IDs of inv.Anchors that belong to this branch.
func CheckAllAnchors(inv model.Inventory, fork model.ForkPoint) map[string][]ConsistencyViolation {
	if len(fork.Branches) == 0 {
		return map[string][]ConsistencyViolation{}
	}

	// Build a quick lookup from input/anchor id → branch id.
	idToBranch := make(map[string]string, 32)
	for _, b := range fork.Branches {
		for _, id := range b.Inputs {
			idToBranch[id] = b.ID
		}
		for _, id := range b.Anchors {
			// Anchors on this branch could in principle be referenced
			// from a sibling branch's anchor; treat that as the same
			// kind of cross-branch contamination.
			if _, dup := idToBranch[id]; !dup {
				idToBranch[id] = b.ID
			}
		}
	}

	anchorByID := make(map[string]model.Anchor, len(inv.Anchors))
	for _, a := range inv.Anchors {
		anchorByID[a.ID] = a
	}

	out := make(map[string][]ConsistencyViolation, len(fork.Branches))
	for _, branch := range fork.Branches {
		var perBranch []ConsistencyViolation
		for _, anchorID := range branch.Anchors {
			anchor, ok := anchorByID[anchorID]
			if !ok {
				continue
			}
			vs := CheckBranchConsistency(anchor, branch, fork.SharedPrefix)
			for i := range vs {
				if owner, located := idToBranch[vs[i].InputID]; located && owner != branch.ID {
					vs[i].InputBranch = owner
					vs[i].Description = fmt.Sprintf(
						"anchor %s on branch %s depends on %s, which belongs to branch %s",
						vs[i].AnchorID, branch.ID, vs[i].InputID, owner)
				}
			}
			perBranch = append(perBranch, vs...)
		}
		out[branch.ID] = perBranch
	}
	return out
}
