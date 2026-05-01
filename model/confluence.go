package model

import (
	"encoding/json"
	"fmt"
)

// ChainRef identifies a chain participating in a confluence point.
// New in Theory v0.2: a confluence is a *set* of ChainRefs (N-ary),
// each tagged with whether it is internal to the programme, external,
// or cross-programme.
type ChainRef struct {
	Fidelity   *float64        `json:"fidelity,omitempty"`
	ChainID    string          `json:"chain_id"`
	Programme  string          `json:"programme,omitempty"`
	Summary    string          `json:"summary,omitempty"`
	Provenance ChainProvenance `json:"provenance,omitempty"`
}

// ConfluencePoint is the parity-check of the framework: an anchor reached
// independently by multiple chains. Theory v0.2 generalizes to N-ary;
// pre-v0.2 inventories use binary path_a / path_b which the JSON store
// migrates to the Paths field on load.
type ConfluencePoint struct {
	LegacyPathA    *string    `json:"path_a,omitempty"`
	LegacyPathB    *string    `json:"path_b,omitempty"`
	ID             string     `json:"id"`
	AnchorID       string     `json:"anchor_id"`
	Description    string     `json:"description,omitempty"`
	Paths          []ChainRef `json:"paths,omitempty"`
	MutualInfoBits float64    `json:"mutual_info_bits,omitempty"`
	Status         Status     `json:"status"`
}

// Validate enforces confluence-level invariants. A confluence with zero
// paths is allowed — it represents a "named convergence point" where the
// chains have not yet been formalized (e.g. cross-field structural
// convergences).
func (c ConfluencePoint) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("confluence: empty id")
	}
	if c.AnchorID == "" {
		return fmt.Errorf("confluence %s: empty anchor_id", c.ID)
	}
	for i, p := range c.Paths {
		if p.Fidelity != nil && (*p.Fidelity < 0 || *p.Fidelity > 1) {
			return fmt.Errorf("confluence %s: paths[%d]: fidelity %g out of [0,1]", c.ID, i, *p.Fidelity)
		}
	}
	return nil
}

// NormalizePaths populates c.Paths from legacy path_a / path_b fields if
// Paths is empty. The legacy fields are then cleared. Programme is the
// owning inventory's programme name, used to tag the synthesized refs.
func (c *ConfluencePoint) NormalizePaths(programme string) {
	if len(c.Paths) > 0 {
		return
	}
	if c.LegacyPathA != nil && *c.LegacyPathA != "" {
		c.Paths = append(c.Paths, ChainRef{
			ChainID:    *c.LegacyPathA,
			Programme:  programme,
			Provenance: ChainProvenanceInternal,
		})
	}
	if c.LegacyPathB != nil && *c.LegacyPathB != "" {
		c.Paths = append(c.Paths, ChainRef{
			ChainID:    *c.LegacyPathB,
			Programme:  programme,
			Provenance: ChainProvenanceInternal,
		})
	}
	c.LegacyPathA = nil
	c.LegacyPathB = nil
}

// MarshalJSON drops the legacy fields on output so saved files are always
// in the v0.2 shape, even if loaded from a v0.1 fixture.
func (c ConfluencePoint) MarshalJSON() ([]byte, error) {
	type alias ConfluencePoint
	clone := alias(c)
	clone.LegacyPathA = nil
	clone.LegacyPathB = nil
	return json.Marshal(clone)
}
