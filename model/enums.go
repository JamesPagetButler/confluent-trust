package model

import (
	"encoding/json"
	"fmt"
)

// jsonNull is the JSON literal for a null value, used in the enum
// MarshalJSON / UnmarshalJSON fast paths.
const jsonNull = "null"

// Tier is an anchor's position in the derivation hierarchy.
//
//	Tier 0: axioms (underivable assumptions)
//	Tier 1: theoretical proofs / derived principles
//	Tier 2: experimental measurements
//	Tier 3: predictions awaiting evidence
type Tier int8

// Tier values per Theory v0.2 §4.1.
const (
	TierAxiom       Tier = 0
	TierProof       Tier = 1
	TierMeasurement Tier = 2
	TierPrediction  Tier = 3
)

// Status classifies an anchor or chain's epistemic state.
type Status uint8

// Status values per Theory v0.2 §4.1. StatusUnknown is the zero value
// for anchors that have not yet been classified.
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

// Provenance values: T = theoretical, E = experimental, H = hypothesis.
const (
	ProvenanceUnknown Provenance = iota
	ProvenanceTheoretical
	ProvenanceExperimental
	ProvenanceHypothesis
)

// ChainProvenance distinguishes how a chain participates in a confluence.
type ChainProvenance uint8

// ChainProvenance values: Internal (within programme), External (different
// research community), CrossProgramme (different HE programme).
const (
	ChainProvenanceUnknown ChainProvenance = iota
	ChainProvenanceInternal
	ChainProvenanceExternal
	ChainProvenanceCrossProgramme
)

// Burden is the assumption load of a fork branch (Theory §2.8 Def 19a).
type Burden uint8

// Burden values: Minimal adds no assumptions beyond the shared prefix;
// Extended carries the burden of justifying additional assumptions.
const (
	BurdenUnknown Burden = iota
	BurdenMinimal
	BurdenExtended
)

// ---- Status ----

// String returns the canonical string form of a Status.
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

// MarshalJSON encodes a Status as its canonical string, or null when unknown.
func (s Status) MarshalJSON() ([]byte, error) {
	v := s.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a Status from its canonical string. Unknown values
// produce a wrapped error; null and the empty string become StatusUnknown.
func (s *Status) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
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

// String returns the canonical string form of a Provenance.
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

// MarshalJSON encodes a Provenance as T / E / H, or null when unknown.
func (p Provenance) MarshalJSON() ([]byte, error) {
	v := p.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a Provenance from T / E / H. Unknown values produce
// a wrapped error; null and the empty string become ProvenanceUnknown.
func (p *Provenance) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
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

// String returns the canonical string form of a ChainProvenance.
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

// MarshalJSON encodes a ChainProvenance as Internal / External /
// CrossProgramme, or null when unknown.
func (c ChainProvenance) MarshalJSON() ([]byte, error) {
	v := c.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a ChainProvenance from its canonical string.
// Unknown values produce a wrapped error.
func (c *ChainProvenance) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
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

// String returns the canonical string form of a Burden.
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

// MarshalJSON encodes a Burden as Minimal / Extended, or null when unknown.
func (b Burden) MarshalJSON() ([]byte, error) {
	v := b.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a Burden from Minimal / Extended. Unknown values
// produce a wrapped error.
func (b *Burden) UnmarshalJSON(data []byte) error {
	if string(data) == jsonNull {
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
