package model

import "fmt"

// Axiom is a Tier 0 anchor: a programme's underivable assumption.
// In merged programmes an axiom may be marked Derivable=true with
// DerivedFromAxioms populated; in that case it is "stated as an axiom
// for architectural convenience" and the Validate() invariant relaxes.
type Axiom struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Statement         string   `json:"statement"`
	Layer             int      `json:"layer,omitempty"`
	Derivable         bool     `json:"derivable"`
	DerivedFromAxioms []string `json:"derived_from_axioms,omitempty"`
	InheritedFrom     string   `json:"inherited_from,omitempty"`
	Notes             string   `json:"notes,omitempty"`
}

// Validate enforces the Tier 0 derivability invariant: an axiom marked
// derivable=true must list its parent axioms.
func (a Axiom) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("axiom: empty id")
	}
	if a.Derivable && len(a.DerivedFromAxioms) == 0 {
		return fmt.Errorf("axiom %s: derivable=true requires derived_from_axioms", a.ID)
	}
	return nil
}

// DerivedPrinciple is a Tier 1 named derivation, conventionally with id DERIV-*.
// It is a high-level corollary of axioms and proofs that gets a stable name
// rather than being buried in a chain.
type DerivedPrinciple struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Statement    string   `json:"statement"`
	DerivedFrom  []string `json:"derived_from"`
	Layer        int      `json:"layer,omitempty"`
}

// Anchor is a Tier 1–3 node: a proof, measurement, prediction, or observation.
// Computed quantities (Domain, ResidualEntropy, ConfirmatoryInfo) are not
// stored here; package compute produces them on demand.
type Anchor struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Description           string  `json:"description"`
	Tier                  Tier    `json:"tier"`
	Provenance            Provenance `json:"provenance"`
	Status                Status  `json:"status"`
	PredictionChain       []string `json:"prediction_chain"`

	PredictedValue        *float64 `json:"predicted_value,omitempty"`
	PredictedUnit         string   `json:"predicted_unit,omitempty"`
	MeasuredValue         *float64 `json:"measured_value,omitempty"`
	MeasuredError         *float64 `json:"measured_error,omitempty"`
	MeasuredSource        string   `json:"measured_source,omitempty"`
	DiscrepancyPct        *float64 `json:"discrepancy_pct,omitempty"`

	LastTestedAt          *string  `json:"last_tested_at,omitempty"`
	InheritedFrom         string   `json:"inherited_from,omitempty"`
	InheritedAt           string   `json:"inherited_at,omitempty"`

	ProofSystem           string   `json:"proof_system,omitempty"`
	SorryCount            *int     `json:"sorry_count,omitempty"`
	ProofFile             string   `json:"proof_file,omitempty"`
	LeanTheorem           string   `json:"lean_theorem,omitempty"`
	LeanCompanionTheorems []string `json:"lean_companion_theorems,omitempty"`

	BridgeRole            string   `json:"bridge_role,omitempty"`
	TestableWhen          string   `json:"testable_when,omitempty"`
	Notes                 string   `json:"notes,omitempty"`

	BranchID              string   `json:"branch_id,omitempty"`
}

// Validate enforces anchor-level invariants.
func (a Anchor) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("anchor: empty id")
	}
	if a.Tier < TierProof || a.Tier > TierPrediction {
		return fmt.Errorf("anchor %s: tier %d out of range [1,3]", a.ID, a.Tier)
	}
	return nil
}

// Input is an underived parameter — an "eddy" in the river metaphor (§4.6).
// Its entropy contribution is 3.32 * SignificantFigures bits.
type Input struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	Status             string `json:"status"`
	SignificantFigures int    `json:"significant_figures,omitempty"`
	Notes              string `json:"notes,omitempty"`
}

// Validate enforces input-level invariants.
func (i Input) Validate() error {
	if i.ID == "" {
		return fmt.Errorf("input: empty id")
	}
	if i.Type != "input" {
		return fmt.Errorf("input %s: type must be \"input\", got %q", i.ID, i.Type)
	}
	return nil
}
