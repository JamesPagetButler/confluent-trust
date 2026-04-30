package model

import (
	"encoding/json"
	"fmt"
)

// Tier is an anchor's position in the derivation hierarchy.
//
//	Tier 0: axioms (underivable assumptions)
//	Tier 1: theoretical proofs / derived principles
//	Tier 2: experimental measurements
//	Tier 3: predictions awaiting evidence
type Tier int8

const (
	TierAxiom       Tier = 0
	TierProof       Tier = 1
	TierMeasurement Tier = 2
	TierPrediction  Tier = 3
)

// Status classifies an anchor or chain's epistemic state.
type Status uint8

const (
	StatusUnknown Status = iota
	StatusCoherent
	StatusUntested
	StatusIncoherent
	StatusContested
	StatusRefuted
)

// Provenance distinguishes theoretical, experimental, and hypothesis anchors.
type Provenance uint8

const (
	ProvenanceUnknown Provenance = iota
	ProvenanceTheoretical
	ProvenanceExperimental
	ProvenanceHypothesis
)

// ChainProvenance distinguishes how a chain participates in a confluence.
type ChainProvenance uint8

const (
	ChainProvenanceUnknown ChainProvenance = iota
	ChainProvenanceInternal
	ChainProvenanceExternal
	ChainProvenanceCrossProgramme
)

// Burden is the assumption load of a fork branch (Theory §2.8 Def 19a).
type Burden uint8

const (
	BurdenUnknown Burden = iota
	BurdenMinimal
	BurdenExtended
)

// ---- Status ----

func (s Status) String() string {
	switch s {
	case StatusCoherent:
		return "coherent"
	case StatusUntested:
		return "untested"
	case StatusIncoherent:
		return "incoherent"
	case StatusContested:
		return "contested"
	case StatusRefuted:
		return "refuted"
	default:
		return ""
	}
}

func (s Status) MarshalJSON() ([]byte, error) {
	v := s.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

func (s *Status) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = StatusUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("status: %w", err)
	}
	switch raw {
	case "":
		*s = StatusUnknown
	case "coherent":
		*s = StatusCoherent
	case "untested":
		*s = StatusUntested
	case "incoherent":
		*s = StatusIncoherent
	case "contested":
		*s = StatusContested
	case "refuted":
		*s = StatusRefuted
	default:
		return fmt.Errorf("status: unknown value %q", raw)
	}
	return nil
}

// ---- Provenance ----

func (p Provenance) String() string {
	switch p {
	case ProvenanceTheoretical:
		return "T"
	case ProvenanceExperimental:
		return "E"
	case ProvenanceHypothesis:
		return "H"
	default:
		return ""
	}
}

func (p Provenance) MarshalJSON() ([]byte, error) {
	v := p.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

func (p *Provenance) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*p = ProvenanceUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("provenance: %w", err)
	}
	switch raw {
	case "":
		*p = ProvenanceUnknown
	case "T":
		*p = ProvenanceTheoretical
	case "E":
		*p = ProvenanceExperimental
	case "H":
		*p = ProvenanceHypothesis
	default:
		return fmt.Errorf("provenance: unknown value %q", raw)
	}
	return nil
}

// ---- ChainProvenance ----

func (c ChainProvenance) String() string {
	switch c {
	case ChainProvenanceInternal:
		return "Internal"
	case ChainProvenanceExternal:
		return "External"
	case ChainProvenanceCrossProgramme:
		return "CrossProgramme"
	default:
		return ""
	}
}

func (c ChainProvenance) MarshalJSON() ([]byte, error) {
	v := c.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

func (c *ChainProvenance) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*c = ChainProvenanceUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("chain provenance: %w", err)
	}
	switch raw {
	case "":
		*c = ChainProvenanceUnknown
	case "Internal":
		*c = ChainProvenanceInternal
	case "External":
		*c = ChainProvenanceExternal
	case "CrossProgramme":
		*c = ChainProvenanceCrossProgramme
	default:
		return fmt.Errorf("chain provenance: unknown value %q", raw)
	}
	return nil
}

// ---- Burden ----

func (b Burden) String() string {
	switch b {
	case BurdenMinimal:
		return "Minimal"
	case BurdenExtended:
		return "Extended"
	default:
		return ""
	}
}

func (b Burden) MarshalJSON() ([]byte, error) {
	v := b.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

func (b *Burden) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*b = BurdenUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("burden: %w", err)
	}
	switch raw {
	case "":
		*b = BurdenUnknown
	case "Minimal":
		*b = BurdenMinimal
	case "Extended":
		*b = BurdenExtended
	default:
		return fmt.Errorf("burden: unknown value %q", raw)
	}
	return nil
}
