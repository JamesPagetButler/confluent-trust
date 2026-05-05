package report

import (
	"fmt"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Dashboard returns a compact text dashboard summarising an inventory's
// epistemic health. The layout:
//
//	╔════════════════════════════════════════════════════════════╗
//	║ CTH Health: <programme> v<version>                         ║
//	╠════════════════════════════════════════════════════════════╣
//	║ Anchors:        <total>  (axiom <a>, proof <p>, meas <m>,  ║
//	║                           pred <r>)                        ║
//	║ Coherence:      <R_c>                                      ║
//	║ ρ_net:          <ρ>  [<half>, <base>, <double>]            ║
//	║ Velocity:       (Δρ/Δn — supply two snapshots to populate) ║
//	║ Top bridge:     <id>                                       ║
//	║ Sediment:       <regime distribution>                      ║
//	║ Highest eddy:   <input id>  (η = <η>, g_w = <g>)           ║
//	╚════════════════════════════════════════════════════════════╝
//
// Issue #17 acceptance: 8 sections present with no <unset>/nil values
// for any computed metric.
func Dashboard(inv model.Inventory, fa FullAnalysis) string {
	var b strings.Builder
	b.Grow(1024)

	border := strings.Repeat("═", 60)
	fmt.Fprintf(&b, "╔%s╗\n", border)
	fmt.Fprintf(&b, "║ CTH Health: %-46s ║\n", fmt.Sprintf("%s v%s", inv.Programme, inv.Version))
	fmt.Fprintf(&b, "╠%s╣\n", border)

	tier := fa.TierBreakdown
	totalAnchors := len(inv.Axioms) + len(inv.Anchors)
	fmt.Fprintf(&b, "║ Anchors:        %-3d  (axiom %d, proof %d, meas %d, pred %d)%s ║\n",
		totalAnchors,
		tier[model.TierAxiom], tier[model.TierProof], tier[model.TierMeasurement], tier[model.TierPrediction],
		strings.Repeat(" ", 0))
	fmt.Fprintf(&b, "║ Coherence:      %-43.4f ║\n", fa.CoherenceRatio)
	fmt.Fprintf(&b, "║ ρ_net:          %-43s ║\n",
		fmt.Sprintf("%.4f  [%.4f, %.4f, %.4f]", fa.NetRho,
			fa.SensitivityHalfH, fa.SensitivityBaseH, fa.SensitivityDouble))
	fmt.Fprintf(&b, "║ ρ_gross:        %-43.4f ║\n", fa.GrossRho)
	fmt.Fprintf(&b, "║ Top bridge:     %-43s ║\n", emptyAsDash(fa.TopBridge))
	fmt.Fprintf(&b, "║ Sediment:       %-43s ║\n", sedimentSummary(fa))
	fmt.Fprintf(&b, "║ Highest eddy:   %-43s ║\n", eddySummary(fa))

	fmt.Fprintf(&b, "╚%s╝\n", border)
	return b.String()
}

func sedimentSummary(fa FullAnalysis) string {
	return fmt.Sprintf("L=%d  LS=%d  M=%d  H=%d  sharp=%t",
		len(fa.Sediment.Laminar.ChainIDs),
		len(fa.Sediment.LowSediment.ChainIDs),
		len(fa.Sediment.Moderate.ChainIDs),
		len(fa.Sediment.Heavy.ChainIDs),
		fa.Sediment.SharpPartition)
}

func eddySummary(fa FullAnalysis) string {
	if len(fa.EddyRanking) == 0 {
		return "—"
	}
	top := fa.EddyRanking[0]
	return fmt.Sprintf("%s  (π_w=%.4f, g_w=%.4f)", top.InputID, top.Proximity, top.Gap)
}

func emptyAsDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
