package compute

import (
	"sort"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// SedimentPartition is one regime bucket of the sediment report. ChainIDs
// are listed in the order they appear in the inventory; DomainCounts maps
// a domain label to the number of chains in this partition that target an
// anchor in that domain.
type SedimentPartition struct {
	DomainCounts map[string]int
	Regime       string
	ChainIDs     []string
}

// SedimentReport is the output of DetectSedimentPartitions: the four
// regime partitions (Method 3 of the Python engine), plus the derived
// clean/dirty domain split and the sharp-partition flag that fires when
// at least one domain appears exclusively in clean partitions and at
// least one other appears exclusively in dirty partitions.
type SedimentReport struct {
	Laminar          SedimentPartition
	LowSediment      SedimentPartition
	Moderate         SedimentPartition
	Heavy            SedimentPartition
	CleanOnlyDomains []string
	DirtyOnlyDomains []string
	SharpPartition   bool
}

// DetectSedimentPartitions partitions an inventory's chains by their
// fidelity regime (Issue #5 / Theory §4.4) and reports the domain
// composition of each partition.
//
// Domain attribution rules (this function, until #10 lands a real
// classifier):
//
//  1. chainDomain(c) — currently empty; reserved for #10's
//     ClassifyDomain. When it returns a non-empty label, that label is
//     credited.
//  2. Otherwise the chain's TargetID prefix (e.g. MEAS-, PROOF-, INST-,
//     PRED-, DERIV-, AXIOM-) is used as the domain label. A target_id
//     with no dash is credited as itself.
//
// "Clean-only" domains appear only in the laminar/low_sediment partitions;
// "dirty-only" domains appear only in moderate/heavy. A SharpPartition
// fires when at least one clean-only AND at least one dirty-only domain
// exist — the qualitative signal that the inventory has bimodally split
// across a fidelity threshold.
func DetectSedimentPartitions(inv model.Inventory) SedimentReport {
	report := SedimentReport{
		Laminar:     SedimentPartition{Regime: RegimeLaminar, DomainCounts: map[string]int{}},
		LowSediment: SedimentPartition{Regime: RegimeLowSediment, DomainCounts: map[string]int{}},
		Moderate:    SedimentPartition{Regime: RegimeModerate, DomainCounts: map[string]int{}},
		Heavy:       SedimentPartition{Regime: RegimeHeavy, DomainCounts: map[string]int{}},
	}

	// cleanDomains: domains that have appeared in laminar or low_sediment.
	// dirtyDomains: domains that have appeared in moderate or heavy.
	cleanDomains := map[string]struct{}{}
	dirtyDomains := map[string]struct{}{}

	for _, c := range inv.Chains {
		mu := ChainFidelity(c)
		regime := ClassifyFidelityRegime(mu)
		domain := chainDomain(c)
		if domain == "" {
			domain = targetPrefix(c.TargetID)
		}

		var part *SedimentPartition
		switch regime {
		case RegimeLaminar:
			part = &report.Laminar
			cleanDomains[domain] = struct{}{}
		case RegimeLowSediment:
			part = &report.LowSediment
			cleanDomains[domain] = struct{}{}
		case RegimeModerate:
			part = &report.Moderate
			dirtyDomains[domain] = struct{}{}
		case RegimeHeavy:
			part = &report.Heavy
			dirtyDomains[domain] = struct{}{}
		default:
			// Unknown regime: skip rather than miscount.
			continue
		}
		part.ChainIDs = append(part.ChainIDs, c.ID)
		part.DomainCounts[domain]++
	}

	report.CleanOnlyDomains = sortedDifference(cleanDomains, dirtyDomains)
	report.DirtyOnlyDomains = sortedDifference(dirtyDomains, cleanDomains)
	report.SharpPartition = len(report.CleanOnlyDomains) > 0 && len(report.DirtyOnlyDomains) > 0

	return report
}

// chainDomain is the hook for Issue #10's ClassifyDomain. Until that
// lands, this is intentionally trivial — returning "" defers to the
// target_id prefix fallback in DetectSedimentPartitions.
func chainDomain(_ model.Chain) string {
	return ""
}

// targetPrefix extracts the dash-prefix of a target_id. "MEAS-foo" →
// "MEAS"; "REBCO" (no dash) → "REBCO"; "" → "".
func targetPrefix(targetID string) string {
	if i := strings.IndexByte(targetID, '-'); i > 0 {
		return targetID[:i]
	}
	return targetID
}

// sortedDifference returns the elements of a not in b, sorted.
func sortedDifference(a, b map[string]struct{}) []string {
	if len(a) == 0 {
		return nil
	}
	out := make([]string, 0, len(a))
	for k := range a {
		if _, in := b[k]; in {
			continue
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
