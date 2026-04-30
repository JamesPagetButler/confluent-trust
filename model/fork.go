package model

import "fmt"

// ForkPoint represents an unresolved binary (or N-ary) hypothesis question
// inside a programme — Theory v0.2 §2.8. The shared prefix contributes to
// every branch's metrics; branch-specific items contribute only to their
// own branch.
type ForkPoint struct {
	ID            string              `json:"id"`
	BranchNodeID  string              `json:"branch_node_id"`
	Question      string              `json:"question"`
	SharedPrefix  []string            `json:"shared_prefix,omitempty"`
	Branches      []Branch            `json:"branches"`
	Observations  []BranchObservation `json:"branch_observations,omitempty"`
}

// Branch is one of the alternative interpretations under a ForkPoint.
type Branch struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Hypothesis  string   `json:"hypothesis"`
	Burden      Burden   `json:"burden"`
	Anchors     []string `json:"anchors,omitempty"`
	Chains      []string `json:"chains,omitempty"`
	Confluences []string `json:"confluences,omitempty"`
	Inputs      []string `json:"inputs,omitempty"`
	Predictions []string `json:"predictions,omitempty"`
}

// BranchObservation tags an observation anchor with its per-branch
// interpretation — Definition 23.
type BranchObservation struct {
	AnchorID        string                 `json:"anchor_id"`
	Interpretations []BranchInterpretation `json:"interpretations"`
}

// BranchInterpretation is one branch's reading of a shared observation.
type BranchInterpretation struct {
	BranchID        string   `json:"branch_id"`
	Interpretation  string   `json:"interpretation"`
	Status          Status   `json:"status"`
	PredictionChain []string `json:"prediction_chain,omitempty"`
}

// Validate enforces fork-level invariants:
//   - at least 2 branches
//   - at least one branch with burden = Minimal
//   - branch IDs unique
func (f ForkPoint) Validate() error {
	if f.ID == "" {
		return fmt.Errorf("fork: empty id")
	}
	if len(f.Branches) < 2 {
		return fmt.Errorf("fork %s: need >= 2 branches, got %d", f.ID, len(f.Branches))
	}
	seen := make(map[string]struct{}, len(f.Branches))
	hasMinimal := false
	for _, b := range f.Branches {
		if b.ID == "" {
			return fmt.Errorf("fork %s: branch with empty id", f.ID)
		}
		if _, dup := seen[b.ID]; dup {
			return fmt.Errorf("fork %s: duplicate branch id %s", f.ID, b.ID)
		}
		seen[b.ID] = struct{}{}
		if b.Burden == BurdenMinimal {
			hasMinimal = true
		}
	}
	if !hasMinimal {
		return fmt.Errorf("fork %s: at least one branch must have burden=Minimal", f.ID)
	}
	return nil
}
