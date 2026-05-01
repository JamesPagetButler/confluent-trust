package model

import "fmt"

// Inventory is the top-level CTH document — a single programme's hypergraph
// state at a point in time. It is the unit consumed and produced by the
// JSON store and the unit on which package compute operates.
type Inventory struct {
	Programme         string             `json:"programme"`
	Version           string             `json:"version"`
	SchemaVersion     string             `json:"schema_version,omitempty"`
	FullName          string             `json:"full_name,omitempty"`
	Timestamp         string             `json:"timestamp,omitempty"`
	ParentProgrammes  []string           `json:"parent_programmes,omitempty"`
	MetaAxiom         *MetaAxiom         `json:"meta_axiom,omitempty"`
	Axioms            []Axiom            `json:"axioms"`
	DerivedPrinciples []DerivedPrinciple `json:"derived_principles,omitempty"`
	Anchors           []Anchor           `json:"anchors"`
	Inputs            []Input            `json:"inputs,omitempty"`
	Chains            []Chain            `json:"chains"`
	ConfluencePoints  []ConfluencePoint  `json:"confluence_points,omitempty"`
	ForkPoints        []ForkPoint        `json:"fork_points,omitempty"`
	Health            *Health            `json:"health,omitempty"`
	Changelog         []ChangelogEntry   `json:"changelog,omitempty"`
}

// MetaAxiom is the optional "axiom about axioms" — e.g. "all knowledge is
// provisional within its axiomatic system".
type MetaAxiom struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Statement string `json:"statement"`
}

// ChangelogEntry records a programme-version note inside the inventory.
type ChangelogEntry struct {
	Version string `json:"version"`
	Date    string `json:"date"`
	Note    string `json:"note"`
}

// Health is a cached snapshot of computed metrics. It is round-trippable
// for inspection but not authoritative — the source of truth is
// recomputation by package compute.
type Health struct {
	RhoNet              *float64       `json:"rho_net,omitempty"`
	RhoNetSensitivity   *Sensitivity   `json:"rho_net_sensitivity,omitempty"`
	CoherenceRatio      *float64       `json:"coherence_ratio,omitempty"`
	CompressionVelocity *float64       `json:"compression_velocity,omitempty"`
	TierBreakdown       map[string]int `json:"tier_breakdown,omitempty"`
	ComputedAt          string         `json:"computed_at,omitempty"`
	EngineVersion       string         `json:"engine_version,omitempty"`
	TopBridge           string         `json:"top_bridge,omitempty"`
	HighestValueEddy    string         `json:"highest_value_eddy,omitempty"`
	AnchorCount         int            `json:"anchor_count,omitempty"`
}

// Sensitivity is the half/base/double bracket from Definition 15.
type Sensitivity struct {
	HalfH   float64 `json:"half_h"`
	BaseH   float64 `json:"base_h"`
	DoubleH float64 `json:"double_h"`
}

// Validate runs all child invariants. Errors are wrapped with the
// inventory + child id for easy localization.
func (inv *Inventory) Validate() error {
	if inv.Programme == "" {
		return fmt.Errorf("inventory: empty programme")
	}
	if inv.Version == "" {
		return fmt.Errorf("inventory %s: empty version", inv.Programme)
	}
	for _, a := range inv.Axioms {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("inventory %s: %w", inv.Programme, err)
		}
	}
	for _, a := range inv.Anchors {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("inventory %s: %w", inv.Programme, err)
		}
	}
	for _, i := range inv.Inputs {
		if err := i.Validate(); err != nil {
			return fmt.Errorf("inventory %s: %w", inv.Programme, err)
		}
	}
	for _, c := range inv.Chains {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("inventory %s: %w", inv.Programme, err)
		}
	}
	for _, cp := range inv.ConfluencePoints {
		if err := cp.Validate(); err != nil {
			return fmt.Errorf("inventory %s: %w", inv.Programme, err)
		}
	}
	for _, f := range inv.ForkPoints {
		if err := f.Validate(); err != nil {
			return fmt.Errorf("inventory %s: %w", inv.Programme, err)
		}
	}
	return nil
}

// NormalizeConfluences upgrades every legacy binary confluence into the
// N-ary Paths shape, tagged with the inventory's programme.
func (inv *Inventory) NormalizeConfluences() {
	for i := range inv.ConfluencePoints {
		inv.ConfluencePoints[i].NormalizePaths(inv.Programme)
	}
}
