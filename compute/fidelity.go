package compute

import (
	"math"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Fidelity classification thresholds per Issue #5 / Theory §4.4.
const (
	FidelityLaminar     = 0.999
	FidelityLowSediment = 0.90
	FidelityModerate    = 0.70
)

// Fidelity regime labels returned by ClassifyFidelityRegime. Exported so
// callers can match on them without string literals.
const (
	RegimeLaminar     = "laminar"
	RegimeLowSediment = "low_sediment"
	RegimeModerate    = "moderate"
	RegimeHeavy       = "heavy"
)

// Step-type labels recognized by the §4.4 fidelity table. Exported so
// inventory authors can reference them rather than embedding string
// literals in their data.
const (
	StepLean4Proof         = "lean4_proof"
	StepEstablishedMath    = "established_math"
	StepStandardPhysics    = "standard_physics"
	StepNumericalVerified  = "numerical_verified"
	StepDomainBoundary     = "domain_boundary"
	StepSemiEmpirical      = "semi_empirical"
	StepUnprovenConjecture = "unproven_conjecture"
)

// stepFidelityTable maps a normalized step-type label to its fidelity per
// the §4.4 table. Lookup is case-insensitive; unknown labels return 1.0 so
// the caller can decide whether to flag them.
var stepFidelityTable = map[string]float64{
	StepLean4Proof:         1.000,
	StepEstablishedMath:    1.000,
	StepStandardPhysics:    0.999,
	StepNumericalVerified:  0.999,
	StepDomainBoundary:     0.95,
	StepSemiEmpirical:      0.95,
	StepUnprovenConjecture: 0.50,
}

// StepFidelity returns the fidelity μ(e) for a derivation step type per the
// §4.4 table. Unknown labels return 1.0; the caller decides whether to log.
func StepFidelity(stepType string) float64 {
	if v, ok := stepFidelityTable[strings.ToLower(strings.TrimSpace(stepType))]; ok {
		return v
	}
	return 1.0
}

// ChainFidelity returns μ(C), the multiplicative product of edge fidelities
// per Definition 9. The selection rule (Issue #5):
//
//  1. If c.Fidelity is set, use it directly. This is the v0.2 preferred path —
//     authors record the chain's verified fidelity rather than relying on
//     step-type inference.
//  2. Else compute the product over c.StepTypes via StepFidelity.
//  3. Else return 1.0 (a chain with no metadata is treated as laminar).
//
// The legacy Python heuristic on weakest_link_id is not reproduced —
// inventories that need it should set c.Fidelity directly.
func ChainFidelity(c model.Chain) float64 {
	if c.Fidelity != nil {
		return clamp01(*c.Fidelity)
	}
	if len(c.StepTypes) == 0 {
		return 1.0
	}
	mu := 1.0
	for _, st := range c.StepTypes {
		mu *= StepFidelity(st)
	}
	return clamp01(mu)
}

// ClassifyFidelityRegime maps a chain fidelity to one of the four regimes
// from Issue #5: laminar (≥0.999), low_sediment (≥0.90), moderate (≥0.70),
// heavy (<0.70).
func ClassifyFidelityRegime(mu float64) string {
	switch {
	case math.IsNaN(mu):
		return RegimeHeavy
	case mu >= FidelityLaminar:
		return RegimeLaminar
	case mu >= FidelityLowSediment:
		return RegimeLowSediment
	case mu >= FidelityModerate:
		return RegimeModerate
	default:
		return RegimeHeavy
	}
}

func clamp01(v float64) float64 {
	switch {
	case math.IsNaN(v):
		return 0.0
	case v < 0:
		return 0.0
	case v > 1:
		return 1.0
	default:
		return v
	}
}
