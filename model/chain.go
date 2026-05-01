package model

import "fmt"

// DomainBoundary marks a chain step that crosses verification domains
// (e.g. theoretical → experimental). New in Theory v0.2.
type DomainBoundary struct {
	FromDomain string  `json:"from_domain"`
	ToDomain   string  `json:"to_domain"`
	AtAnchorID string  `json:"at_anchor_id"`
	Hypothesis string  `json:"hypothesis,omitempty"`
	Fidelity   float64 `json:"fidelity"`
}

// Chain is a derivation hyperedge from one or more sources to a target.
// When Fidelity is set, package compute uses it directly; otherwise it
// computes from StepTypes (preferred) or falls back to the legacy
// WeakestLinkID heuristic.
type Chain struct {
	WeakestLinkID    *string          `json:"weakest_link_id,omitempty"`
	Fidelity         *float64         `json:"fidelity,omitempty"`
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	TargetID         string           `json:"target_id"`
	Notes            string           `json:"notes,omitempty"`
	SourceIDs        []string         `json:"source_ids"`
	StepTypes        []string         `json:"step_types,omitempty"`
	DomainBoundaries []DomainBoundary `json:"domain_boundaries,omitempty"`
	Steps            int              `json:"steps"`
	Status           Status           `json:"status"`
}

// Validate enforces chain-level invariants.
func (c Chain) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("chain: empty id")
	}
	if len(c.SourceIDs) == 0 {
		return fmt.Errorf("chain %s: empty source_ids", c.ID)
	}
	if c.TargetID == "" {
		return fmt.Errorf("chain %s: empty target_id", c.ID)
	}
	if c.Fidelity != nil && (*c.Fidelity < 0 || *c.Fidelity > 1) {
		return fmt.Errorf("chain %s: fidelity %g out of [0,1]", c.ID, *c.Fidelity)
	}
	for _, db := range c.DomainBoundaries {
		if db.Fidelity < 0 || db.Fidelity > 1 {
			return fmt.Errorf("chain %s: domain boundary at %s: fidelity %g out of [0,1]",
				c.ID, db.AtAnchorID, db.Fidelity)
		}
	}
	return nil
}
