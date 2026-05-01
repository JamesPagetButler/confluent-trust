package compute

import (
	"math"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// DefaultAxiomEntropyBits is assigned to a Tier-0 axiom when no explicit
// per-axiom entropy table is supplied. It is intentionally conservative
// (1.0 bit per axiom) and meant to be replaced by a programme-specific
// table or by the sensitivity-bracket reporting requirement (Definition 15).
const DefaultAxiomEntropyBits = 1.0

// inputBitsPerSigFig is the information cost of one significant decimal
// digit, log2(10) ≈ 3.3219.
const inputBitsPerSigFig = 3.321928094887362

// AxiomEntropy returns η(v) for an axiom (Definition 8). When the axiom's
// id appears in assigned, that value is returned; otherwise
// DefaultAxiomEntropyBits.
func AxiomEntropy(a model.Axiom, assigned map[string]float64) float64 {
	if v, ok := assigned[a.ID]; ok {
		return v
	}
	return DefaultAxiomEntropyBits
}

// ResidualEntropy returns η(v) for a Tier-1, Tier-2, or Tier-3 anchor per
// Definition 7. axiomEntropy is consulted only for Tier 0; Tier 3
// returns 0.0 here — chain-derived entropy lands with #5 (chain fidelity)
// and #7 (compression).
func ResidualEntropy(a model.Anchor, axiomEntropy map[string]float64) float64 {
	switch a.Tier {
	case model.TierAxiom:
		if v, ok := axiomEntropy[a.ID]; ok {
			return v
		}
		return DefaultAxiomEntropyBits

	case model.TierProof:
		// Tier 1: a proof is a lossless channel from axioms to theorem.
		// Anchors that explicitly carry an unfinished proof (sorry_count > 0)
		// retain residual entropy until completed.
		if a.SorryCount != nil && *a.SorryCount > 0 {
			return float64(*a.SorryCount)
		}
		return 0.0

	case model.TierMeasurement:
		// Tier 2: -log2(1 - |delta|) for non-zero discrepancy; 0 for exact match.
		if a.DiscrepancyPct == nil {
			return 0.0
		}
		delta := math.Abs(*a.DiscrepancyPct) / 100.0
		if delta == 0 {
			return 0.0
		}
		return entropyFromDelta(delta)

	case model.TierPrediction:
		// Tier 3: chain-derived entropy lands with the chain fidelity (#5)
		// and compression (#7) work. Until then, untested predictions
		// contribute 0 to residual entropy.
		return 0.0

	default:
		return 0.0
	}
}

// ConfirmatoryInfo returns ι(v) per Definition 7a.
func ConfirmatoryInfo(a model.Anchor) float64 {
	switch a.Tier {
	case model.TierAxiom, model.TierProof:
		return 0.0

	case model.TierMeasurement:
		// Tier 2: log2(1/|delta|) for δ > 0; 1.0 for structural match (δ = 0).
		if a.DiscrepancyPct == nil {
			return 0.0
		}
		delta := math.Abs(*a.DiscrepancyPct) / 100.0
		if delta == 0 {
			return 1.0
		}
		// log2(1/delta) = -log2(delta). Clamp at the 1-bit floor for
		// large discrepancies — a measurement with δ ≥ 0.5 carries less
		// than one bit of confirmatory information.
		v := -math.Log2(delta)
		if v < 0 {
			return 0.0
		}
		return v

	case model.TierPrediction:
		// Untested predictions contribute no confirmation. Once promoted
		// to Tier 2 with a measurement, they re-enter the formula above.
		return 0.0

	default:
		return 0.0
	}
}

// InputEntropy returns the information cost of an input parameter measured
// to the given number of significant decimal digits: 3.32 * sf bits.
// A non-positive sigFigures defaults to 3 (the convention in the schema).
func InputEntropy(sigFigures int) float64 {
	if sigFigures <= 0 {
		sigFigures = 3
	}
	return float64(sigFigures) * inputBitsPerSigFig
}

// entropyFromDelta computes -log2(1 - delta) safely. When delta is at or
// above 1.0 (a discrepancy of 100% or worse) the formula diverges; we
// cap at a finite ceiling so the engine never produces +Inf.
//
// The cap of 1024 bits corresponds to delta ≈ 1 - 2^-1024 — well past any
// physically meaningful tolerance and large enough that the resulting
// entropy dominates any reasonable budget but does not break aggregation.
func entropyFromDelta(delta float64) float64 {
	const ceilingBits = 1024.0
	if delta <= 0 {
		return 0.0
	}
	if delta >= 1 {
		return ceilingBits
	}
	v := -math.Log2(1 - delta)
	if math.IsInf(v, 1) || math.IsNaN(v) {
		return ceilingBits
	}
	return v
}
