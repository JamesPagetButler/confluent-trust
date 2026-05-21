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
// v0.3 adds: StatusKilled, StatusMarginal, StatusConverged, StatusFalsified
// (promoted from QBP-local convention per design §3).
const (
	StatusUnknown Status = iota
	StatusCoherent
	StatusUntested
	StatusIncoherent
	StatusContested
	StatusRefuted
	StatusKilled
	StatusMarginal
	StatusConverged
	StatusFalsified
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
	case StatusKilled:
		return "killed"
	case StatusMarginal:
		return "marginal"
	case StatusConverged:
		return "converged"
	case StatusFalsified:
		return "falsified"
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
	case "killed":
		*s = StatusKilled
	case "marginal":
		*s = StatusMarginal
	case "converged":
		*s = StatusConverged
	case "falsified":
		*s = StatusFalsified
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

// ---- ProvenanceKind (v0.3) ----

// ProvenanceKind is the v0.3 fine-grained provenance classifier.
// It supersedes the legacy Provenance enum (T/E/H) and extends it
// with proof, internal-compute, and philosophy values (design §2).
type ProvenanceKind uint8

// ProvenanceKind values per design §2.
const (
	ProvenanceKindUnknown         ProvenanceKind = iota
	ProvenanceKindProof                          // formal proof assistant
	ProvenanceKindTheory                         // internal mathematical argument
	ProvenanceKindTheoryExternal                 // external published theorem invoked as proof
	ProvenanceKindExperiment                     // empirical measurement
	ProvenanceKindHypothesis                     // tentative claim awaiting evidence
	ProvenanceKindInternalCompute                // numerical/symbolic computation
	ProvenanceKindPhilosophy                     // conceptual framing / programmatic principle
)

// String returns the canonical JSON string form of a ProvenanceKind.
func (p ProvenanceKind) String() string {
	switch p {
	case ProvenanceKindProof:
		return "proof"
	case ProvenanceKindTheory:
		return "theory"
	case ProvenanceKindTheoryExternal:
		return "theory-external"
	case ProvenanceKindExperiment:
		return "experiment"
	case ProvenanceKindHypothesis:
		return "hypothesis"
	case ProvenanceKindInternalCompute:
		return "internal-compute"
	case ProvenanceKindPhilosophy:
		return "philosophy"
	default:
		return ""
	}
}

// MarshalJSON encodes a ProvenanceKind as its canonical string, or null when unknown.
func (p ProvenanceKind) MarshalJSON() ([]byte, error) {
	v := p.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a ProvenanceKind from its canonical string.
// Unknown values produce a wrapped error; null and empty string become ProvenanceKindUnknown.
func (p *ProvenanceKind) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
		*p = ProvenanceKindUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("provenance_kind: %w", err)
	}
	switch raw {
	case "":
		*p = ProvenanceKindUnknown
	case "proof":
		*p = ProvenanceKindProof
	case "theory":
		*p = ProvenanceKindTheory
	case "theory-external":
		*p = ProvenanceKindTheoryExternal
	case "experiment":
		*p = ProvenanceKindExperiment
	case "hypothesis":
		*p = ProvenanceKindHypothesis
	case "internal-compute":
		*p = ProvenanceKindInternalCompute
	case "philosophy":
		*p = ProvenanceKindPhilosophy
	default:
		return fmt.Errorf("provenance_kind: unknown value %q", raw)
	}
	return nil
}

// ---- ProofState (v0.3) ----

// ProofState is the rollup state of a proof-bearing anchor (design §4.1).
type ProofState uint8

// ProofState values per design §4.1.
const (
	ProofStateUnknown  ProofState = iota // absent/null — no proof file
	ProofStateVerified                   // all theorems verified
	ProofStatePartial                    // mix of verified/written/not_started theorems
	ProofStateWritten                    // proof file exists, no theorems verified yet
)

// String returns the canonical JSON string form of a ProofState.
func (s ProofState) String() string {
	switch s {
	case ProofStateVerified:
		return "verified"
	case ProofStatePartial:
		return "partial"
	case ProofStateWritten:
		return "written"
	default:
		return ""
	}
}

// MarshalJSON encodes a ProofState as its canonical string, or null when unknown.
func (s ProofState) MarshalJSON() ([]byte, error) {
	v := s.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a ProofState from its canonical string.
// Unknown values produce a wrapped error; null and empty string become ProofStateUnknown.
func (s *ProofState) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
		*s = ProofStateUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("proof_state: %w", err)
	}
	switch raw {
	case "":
		*s = ProofStateUnknown
	case "verified":
		*s = ProofStateVerified
	case "partial":
		*s = ProofStatePartial
	case "written":
		*s = ProofStateWritten
	default:
		return fmt.Errorf("proof_state: unknown value %q", raw)
	}
	return nil
}

// ---- TheoremStatus (v0.3) ----

// TheoremStatus is the per-theorem verification state within a proof file (design §4).
type TheoremStatus uint8

// TheoremStatus values per design §4.
const (
	TheoremStatusUnknown    TheoremStatus = iota
	TheoremStatusVerified                 // theorem successfully verified by proof assistant
	TheoremStatusWritten                  // theorem written in file, not yet verified
	TheoremStatusNotStarted               // declared intent; theorem name not yet in proof file
)

// String returns the canonical JSON string form of a TheoremStatus.
func (t TheoremStatus) String() string {
	switch t {
	case TheoremStatusVerified:
		return "verified"
	case TheoremStatusWritten:
		return "written"
	case TheoremStatusNotStarted:
		return "not_started"
	default:
		return ""
	}
}

// MarshalJSON encodes a TheoremStatus as its canonical string, or null when unknown.
func (t TheoremStatus) MarshalJSON() ([]byte, error) {
	v := t.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a TheoremStatus from its canonical string.
// Unknown values produce a wrapped error; null and empty string become TheoremStatusUnknown.
func (t *TheoremStatus) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
		*t = TheoremStatusUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("theorem_status: %w", err)
	}
	switch raw {
	case "":
		*t = TheoremStatusUnknown
	case "verified":
		*t = TheoremStatusVerified
	case "written":
		*t = TheoremStatusWritten
	case "not_started":
		*t = TheoremStatusNotStarted
	default:
		return fmt.Errorf("theorem_status: unknown value %q", raw)
	}
	return nil
}
