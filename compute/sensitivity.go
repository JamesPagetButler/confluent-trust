package compute

import "github.com/JamesPagetButler/confluent-trust/model"

// SensitivityBracket reports ρ_net at three axiom-entropy scales per
// Theory v0.2 Definition 15 + §4.6: half the assigned axiom entropies,
// the base assignment, and double the base. The triple
// [halfH, baseH, doubleH] is the recommended sensitivity report for any
// ρ_net value: it bounds how much the metric depends on the inherently
// hand-assigned axiom entropies.
//
// axiomEntropy is the per-axiom entropy table; pass nil to use
// DefaultAxiomEntropyBits for every axiom. The returned values are
// computed at scales 0.5x / 1.0x / 2.0x of that table.
func SensitivityBracket(inv model.Inventory, axiomEntropy map[string]float64) (halfH, baseH, doubleH float64) {
	half := scaleEntropy(inv, axiomEntropy, 0.5)
	base := scaleEntropy(inv, axiomEntropy, 1.0)
	double := scaleEntropy(inv, axiomEntropy, 2.0)

	halfH, _ = NetCompression(inv, half)
	baseH, _ = NetCompression(inv, base)
	doubleH, _ = NetCompression(inv, double)
	return halfH, baseH, doubleH
}

// SensitivityRatio returns doubleH / halfH. A ratio above 0.5 indicates
// the ρ_net metric is robust to axiom entropy misestimation by a factor
// of 2 in either direction; below 0.5 means the programme's reported
// health depends critically on a single hand-assigned scale.
//
// halfH = 0 (degenerate inventory with no confirmed information) returns
// 0, signalling "ratio undefined" rather than +Inf or NaN.
func SensitivityRatio(halfH, doubleH float64) float64 {
	if halfH <= 0 {
		return 0
	}
	return doubleH / halfH
}

// scaleEntropy multiplies every entry in the source map by factor and
// fills in DefaultAxiomEntropyBits for any axiom missing from the input.
// The returned map is independent of source so callers can mutate it.
func scaleEntropy(inv model.Inventory, source map[string]float64, factor float64) map[string]float64 {
	scaled := make(map[string]float64, len(inv.Axioms))
	for _, a := range inv.Axioms {
		base := DefaultAxiomEntropyBits
		if source != nil {
			if v, ok := source[a.ID]; ok {
				base = v
			}
		}
		scaled[a.ID] = base * factor
	}
	return scaled
}
