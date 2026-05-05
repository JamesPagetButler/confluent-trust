package report

import (
	"fmt"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// RiverMap returns a narrative description of the inventory's epistemic
// hydrology — the "river map" from Theory v0.2's hydrodynamic metaphor.
// It is intentionally short (a few paragraphs) and oriented for human
// reading, not machine consumption.
func RiverMap(inv model.Inventory) string {
	var b strings.Builder
	b.Grow(1024)

	fmt.Fprintf(&b, "# %s v%s — river map\n\n", inv.Programme, inv.Version)

	fmt.Fprintf(&b, "%d axioms feed the source. ", len(inv.Axioms))
	if len(inv.Inputs) > 0 {
		fmt.Fprintf(&b, "%d eddies — irreducible inputs that the river has not yet absorbed — sit beside the main flow. ",
			len(inv.Inputs))
	} else {
		fmt.Fprint(&b, "No eddies remain — every input has been derived. ")
	}
	fmt.Fprintf(&b, "Downstream, %d derivations weave together at %d confluence points, where independent chains meet to verify a shared target.\n\n",
		len(inv.Chains), len(inv.ConfluencePoints))

	if len(inv.ForkPoints) > 0 {
		fmt.Fprintf(&b, "The river splits at %d fork points; each branch carries a competing hypothesis under the shared upstream prefix.\n\n",
			len(inv.ForkPoints))
	}

	// Flow regimes by tier.
	tierCounts := map[model.Tier]int{}
	for _, a := range inv.Anchors {
		tierCounts[a.Tier]++
	}
	fmt.Fprint(&b, "## Flow regimes\n\n")
	fmt.Fprintf(&b, "- **Laminar bedrock** (Tier 1 proofs): %d. Each is a lossless channel from axioms to theorem.\n",
		tierCounts[model.TierProof])
	fmt.Fprintf(&b, "- **Empirical surface** (Tier 2 measurements): %d. The river meets the world here.\n",
		tierCounts[model.TierMeasurement])
	fmt.Fprintf(&b, "- **Untested predictions** (Tier 3): %d. The river's leading edge — what awaits future measurement.\n\n",
		tierCounts[model.TierPrediction])

	if len(inv.Inputs) > 0 {
		fmt.Fprint(&b, "## Eddy character\n\n")
		var measurable, irreducible, unmeasurable int
		for _, in := range inv.Inputs {
			switch in.Status {
			case "irreducible":
				irreducible++
			case "unmeasurable":
				unmeasurable++
			default:
				measurable++
			}
		}
		fmt.Fprintf(&b, "Of the eddies, %d are measurable (engineering deficit; closeable by improving instrumentation), %d are irreducible (theoretical deficit; closeable only by new theory), and %d are unmeasurable.\n",
			measurable, irreducible, unmeasurable)
	}

	return b.String()
}
