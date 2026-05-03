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

// stepFidelityTable maps a normalized step-type label to its fidelity per
// the §4.4 table. Lookup is case-insensitive; unknown labels return 1.0 so
// the caller can decide whether to flag them.
var stepFidelityTable = map[string]float64{
	"lean4_proof":         1.000,
	"established_math":    1.000,
	"standard_physics":    0.999,
	"numerical_verified":  0.999,
	"domain_boundary":     0.95,
	"semi_empirical":      0.95,
	"unproven_conjecture": 0.50,
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
		return "heavy"
	case mu >= FidelityLaminar:
		return "laminar"
	case mu >= FidelityLowSediment:
		return "low_sediment"
	case mu >= FidelityModerate:
		return "moderate"
	default:
		return "heavy"
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
