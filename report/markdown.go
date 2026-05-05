package report

import (
	"fmt"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// MarkdownReport returns a long-form markdown analysis. Sections:
//
//  1. Header (programme, version, timestamp)
//  2. Compression (gross, net, sensitivity)
//  3. Tier breakdown
//  4. Coherence ratio
//  5. Top bridges (top 5 BridgeNodes)
//  6. Sediment partition
//  7. Eddies (top 5 by weighted proximity)
//  8. Ab-initio scoring (multi-path targets)
//  9. Confluence depth (anchors with depth > 0)
func MarkdownReport(inv model.Inventory, fa FullAnalysis) string {
	var b strings.Builder
	b.Grow(4096)

	fmt.Fprintf(&b, "# CTH analysis — %s v%s\n\n", inv.Programme, inv.Version)
	if inv.Timestamp != "" {
		fmt.Fprintf(&b, "*Inventory timestamp: %s*\n\n", inv.Timestamp)
	}
	if len(inv.ParentProgrammes) > 0 {
		fmt.Fprintf(&b, "*Parent programmes: %s*\n\n", strings.Join(inv.ParentProgrammes, ", "))
	}

	fmt.Fprint(&b, "## Compression\n\n")
	fmt.Fprintf(&b, "- **ρ_gross**: %.4f\n", fa.GrossRho)
	fmt.Fprintf(&b, "- **ρ_net**:   %.4f\n", fa.NetRho)
	fmt.Fprintf(&b, "- **Sensitivity bracket** (½H / H / 2H): %.4f / %.4f / %.4f\n",
		fa.SensitivityHalfH, fa.SensitivityBaseH, fa.SensitivityDouble)
	if fa.SensitivityHalfH > 0 {
		ratio := fa.SensitivityDouble / fa.SensitivityHalfH
		fmt.Fprintf(&b, "- **Sensitivity ratio**: %.4f (>0.5 = robust)\n\n", ratio)
	} else {
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "## Tier breakdown\n\n")
	fmt.Fprintf(&b, "- Tier 0 (axioms):       %d\n", fa.TierBreakdown[model.TierAxiom])
	fmt.Fprintf(&b, "- Tier 1 (proofs):       %d\n", fa.TierBreakdown[model.TierProof])
	fmt.Fprintf(&b, "- Tier 2 (measurements): %d\n", fa.TierBreakdown[model.TierMeasurement])
	fmt.Fprintf(&b, "- Tier 3 (predictions):  %d\n\n", fa.TierBreakdown[model.TierPrediction])

	fmt.Fprintf(&b, "## Coherence\n\n- **R_c**: %.4f\n\n", fa.CoherenceRatio)

	fmt.Fprint(&b, "## Top bridges\n\n")
	if len(fa.BridgeNodes) == 0 {
		fmt.Fprint(&b, "_No bridges detected._\n\n")
	} else {
		fmt.Fprint(&b, "| Anchor | Domains | Count |\n|---|---|---|\n")
		for i, br := range fa.BridgeNodes {
			if i >= 5 {
				break
			}
			fmt.Fprintf(&b, "| %s | %s | %d |\n",
				br.ID, strings.Join(br.Domains, ", "), br.DomainCount)
		}
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "## Sediment partition\n\n")
	fmt.Fprintf(&b, "- Laminar:      %d chains\n", len(fa.Sediment.Laminar.ChainIDs))
	fmt.Fprintf(&b, "- Low sediment: %d chains\n", len(fa.Sediment.LowSediment.ChainIDs))
	fmt.Fprintf(&b, "- Moderate:     %d chains\n", len(fa.Sediment.Moderate.ChainIDs))
	fmt.Fprintf(&b, "- Heavy:        %d chains\n", len(fa.Sediment.Heavy.ChainIDs))
	fmt.Fprintf(&b, "- Sharp partition: %t\n", fa.Sediment.SharpPartition)
	if len(fa.Sediment.CleanOnlyDomains) > 0 {
		fmt.Fprintf(&b, "- Clean-only domains: %s\n", strings.Join(fa.Sediment.CleanOnlyDomains, ", "))
	}
	if len(fa.Sediment.DirtyOnlyDomains) > 0 {
		fmt.Fprintf(&b, "- Dirty-only domains: %s\n", strings.Join(fa.Sediment.DirtyOnlyDomains, ", "))
	}
	fmt.Fprint(&b, "\n")

	fmt.Fprint(&b, "## Eddies (highest-value first)\n\n")
	if len(fa.EddyRanking) == 0 {
		fmt.Fprint(&b, "_No eddies (no inputs in inventory)._\n\n")
	} else {
		fmt.Fprint(&b, "| Input | π_w | g_w | nearest proven |\n|---|---|---|---|\n")
		for i, ed := range fa.EddyRanking {
			if i >= 5 {
				break
			}
			fmt.Fprintf(&b, "| %s | %.4f | %.4f | %s |\n",
				ed.InputID, ed.Proximity, ed.Gap, emptyAsDash(ed.NearestProven))
		}
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "## Ab-initio preference (multi-path targets)\n\n")
	if len(fa.AbInitio) == 0 {
		fmt.Fprint(&b, "_No multi-path targets._\n\n")
	} else {
		fmt.Fprint(&b, "| Target | Best chain | Score |\n|---|---|---|\n")
		for _, r := range fa.AbInitio {
			fmt.Fprintf(&b, "| %s | %s | %.4f |\n", r.TargetID, r.BestChainID, r.BestScore)
		}
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "## Confluence depth\n\n")
	depthCount := 0
	for _, d := range fa.AnchorDepth {
		if d > 0 {
			depthCount++
		}
	}
	fmt.Fprintf(&b, "%d / %d anchors have non-zero arity-weighted confluence depth.\n\n",
		depthCount, len(fa.AnchorDepth))

	return b.String()
}
