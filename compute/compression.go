package compute

import "github.com/JamesPagetButler/confluent-trust/model"

// AnchorCompression is the per-anchor breakdown produced by NetCompression.
// It records the bits each confirmed Tier-2 anchor contributes (gross),
// the fractional input cost it absorbs, the resulting net contribution,
// and which inputs it depends on.
type AnchorCompression struct {
	AnchorID   string
	InputsUsed []string
	GrossBits  float64
	InputCost  float64
	NetBits    float64
}

// NetCompressionDetail records the ρ_net computation in full so callers
// (CLI report, dashboards) can show the per-anchor breakdown.
type NetCompressionDetail struct {
	PerAnchor   []AnchorCompression
	IConfirmed  float64
	IInputCost  float64
	HAxioms     float64
	InfoDeficit float64
	GrossRho    float64
	NetRho      float64
}

// VersionSnapshot is the minimal state needed to compute Δρ/Δn between
// two inventory versions for CompressionVelocity.
type VersionSnapshot struct {
	Rho         float64
	AnchorCount int
}

// AxiomEntropySum returns H_axioms = Σ η(v) over Tier-0 axioms.
func AxiomEntropySum(inv model.Inventory, axiomEntropy map[string]float64) float64 {
	var total float64
	for _, a := range inv.Axioms {
		total += AxiomEntropy(a, axiomEntropy)
	}
	return total
}

// InformationDeficit returns Δ(G) per Definition 11: the total entropy of
// irreducible inputs (the "eddies" in §4.6).
func InformationDeficit(inv model.Inventory) float64 {
	var total float64
	for _, in := range inv.Inputs {
		total += InputEntropy(in.SignificantFigures)
	}
	return total
}

// confirmedAnchors returns the Tier-2 anchors with status Coherent — the
// set that contributes I_confirmed to Definition 13.
func confirmedAnchors(inv model.Inventory) []model.Anchor {
	out := make([]model.Anchor, 0, len(inv.Anchors))
	for _, a := range inv.Anchors {
		if a.Tier == model.TierMeasurement && a.Status == model.StatusCoherent {
			out = append(out, a)
		}
	}
	return out
}

// inputIDs returns the set of input IDs declared in the inventory. Used
// to pick out which entries of an anchor's prediction_chain reference
// underived inputs (and therefore consume input entropy).
func inputIDs(inv model.Inventory) map[string]struct{} {
	ids := make(map[string]struct{}, len(inv.Inputs))
	for _, in := range inv.Inputs {
		ids[in.ID] = struct{}{}
	}
	return ids
}

// GrossCompression returns ρ_gross per Definition 13:
//
//	ρ = I_confirmed / (H_axioms + Δ)
//
// where I_confirmed is the sum of ι(v) over confirmed anchors,
// H_axioms is the total axiom entropy, and Δ is the input deficit.
// Returns 0 when the denominator is non-positive.
func GrossCompression(inv model.Inventory, axiomEntropy map[string]float64) float64 {
	var iConfirmed float64
	for _, a := range confirmedAnchors(inv) {
		iConfirmed += ConfirmatoryInfo(a)
	}
	denom := AxiomEntropySum(inv, axiomEntropy) + InformationDeficit(inv)
	if denom <= 0 {
		return 0
	}
	return iConfirmed / denom
}

// NetCompression returns ρ_net per Definition 13 (the "net compressor"
// variant) along with a per-anchor breakdown. Each input's entropy is
// divided equally among the confirmed anchors whose prediction_chain
// depends on it (fractional input cost allocation).
func NetCompression(inv model.Inventory, axiomEntropy map[string]float64) (float64, NetCompressionDetail) {
	confirmed := confirmedAnchors(inv)
	inputs := inputIDs(inv)

	// Pass 1: count consumers per input.
	consumers := make(map[string]int, len(inputs))
	for _, a := range confirmed {
		seen := make(map[string]struct{})
		for _, dep := range a.PredictionChain {
			if _, ok := inputs[dep]; !ok {
				continue
			}
			if _, dup := seen[dep]; dup {
				continue
			}
			seen[dep] = struct{}{}
			consumers[dep]++
		}
	}

	// Pass 2: per-anchor breakdown.
	perAnchor := make([]AnchorCompression, 0, len(confirmed))
	var iConfirmed, iInputCost float64
	for _, a := range confirmed {
		gross := ConfirmatoryInfo(a)
		iConfirmed += gross

		row := AnchorCompression{
			AnchorID:  a.ID,
			GrossBits: gross,
		}
		seen := make(map[string]struct{})
		for _, dep := range a.PredictionChain {
			if _, ok := inputs[dep]; !ok {
				continue
			}
			if _, dup := seen[dep]; dup {
				continue
			}
			seen[dep] = struct{}{}

			// Allocate this input's entropy across its consumers.
			consumerCount := consumers[dep]
			if consumerCount <= 0 {
				continue
			}
			var sf int
			for _, in := range inv.Inputs {
				if in.ID == dep {
					sf = in.SignificantFigures
					break
				}
			}
			share := InputEntropy(sf) / float64(consumerCount)
			row.InputCost += share
			row.InputsUsed = append(row.InputsUsed, dep)
		}
		row.NetBits = row.GrossBits - row.InputCost
		iInputCost += row.InputCost
		perAnchor = append(perAnchor, row)
	}

	hAxioms := AxiomEntropySum(inv, axiomEntropy)
	deficit := InformationDeficit(inv)
	denom := hAxioms + deficit

	var grossRho, netRho float64
	if denom > 0 {
		grossRho = iConfirmed / denom
		netRho = (iConfirmed - iInputCost) / denom
	}

	return netRho, NetCompressionDetail{
		IConfirmed:  iConfirmed,
		IInputCost:  iInputCost,
		HAxioms:     hAxioms,
		InfoDeficit: deficit,
		GrossRho:    grossRho,
		NetRho:      netRho,
		PerAnchor:   perAnchor,
	}
}

// CompressionVelocity returns Δρ / Δn between two inventory snapshots
// per Definition 14. Returns 0 when ΔanchorCount is zero (no progress
// between snapshots ⇒ velocity is undefined).
func CompressionVelocity(prev, curr VersionSnapshot) float64 {
	dN := curr.AnchorCount - prev.AnchorCount
	if dN == 0 {
		return 0
	}
	return (curr.Rho - prev.Rho) / float64(dN)
}
