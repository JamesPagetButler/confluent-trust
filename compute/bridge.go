package compute

import (
	"sort"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// DomainPrefixMap maps an ID-prefix (e.g. "AXIOM-") to the verification
// domain a node with that prefix belongs to. The defaults are seeded from
// the QBP-Compute-Unit v3.2 reference fixture: AXIOM- → math, PROOF- →
// lean, MEAS- / OBS- / INST- / INPUT- → lab, PRED- → prediction, DERIV-
// → lean, FLAG- → meta.
//
// The map is package-level and intentionally mutable so an inventory
// author can register additional prefixes (or override defaults) before
// calling ClassifyDomain or BridgeCentrality. Callers that need multiple
// independent classifications in the same process should not race on it;
// configuration is expected at init time.
var DomainPrefixMap = map[string]string{
	"AXIOM-": "math",
	"PROOF-": "lean",
	"DERIV-": "lean",
	"MEAS-":  "lab",
	"OBS-":   "lab",
	"INST-":  "lab",
	"INPUT-": "lab",
	"PRED-":  "prediction",
	"FLAG-":  "meta",
}

// ClassifyDomain returns the verification domain for an ID by looking it up
// in DomainPrefixMap with a longest-prefix-wins rule. An ID with no
// matching prefix returns the empty string.
func ClassifyDomain(id string) string {
	bestLen := 0
	bestDomain := ""
	for prefix, domain := range DomainPrefixMap {
		if len(prefix) <= bestLen {
			continue
		}
		if len(id) < len(prefix) {
			continue
		}
		if id[:len(prefix)] == prefix {
			bestLen = len(prefix)
			bestDomain = domain
		}
	}
	return bestDomain
}

// BridgeNode is one entry in the bridge-centrality result: an anchor that
// participates in chains spanning some number of distinct domains.
//
// Field order is chosen for govet -fieldalignment: Domains (24B slice
// header) first, ID (16B string header) second, DomainCount (8B int) last.
type BridgeNode struct {
	ID          string
	Domains     []string
	DomainCount int
}

// BridgeCentrality returns one BridgeNode per anchor in the inventory,
// sorted by DomainCount descending and ID ascending for stability.
//
// Each anchor's domain set is the union of (a) its own ID's domain via
// ClassifyDomain, and (b) the domains of every chain in which the anchor
// is either the target or appears in source_ids — for those chains, both
// the target's and every source's domain are added.
//
// When excludeAxioms is true, Tier-0 anchors are omitted from the result
// entirely; they are still allowed to contribute domain membership to
// other anchors via the chain-neighborhood expansion (matching the
// Python engine's "axioms trivially connect to everything, but the
// interesting bridges are the non-axiom downstream nodes" reasoning).
func BridgeCentrality(inv model.Inventory, excludeAxioms bool) []BridgeNode {
	// Index anchors by ID once. Use a stable view rather than copying the
	// whole struct so we keep the field-alignment win on Anchor itself.
	anchorByID := make(map[string]*model.Anchor, len(inv.Anchors))
	for i := range inv.Anchors {
		a := &inv.Anchors[i]
		anchorByID[a.ID] = a
	}

	// Seed each anchor's domain set with its own ID's classified domain.
	// Empty domains (unknown prefix) are not added to the set so they do
	// not inflate DomainCount.
	domains := make(map[string]map[string]struct{}, len(inv.Anchors))
	for i := range inv.Anchors {
		a := &inv.Anchors[i]
		set := make(map[string]struct{}, 4)
		if d := ClassifyDomain(a.ID); d != "" {
			set[d] = struct{}{}
		}
		domains[a.ID] = set
	}

	// Walk chains: every anchor that participates (target or source)
	// inherits the union of {target's domain} ∪ {each source's domain}.
	for _, c := range inv.Chains {
		neighborhood := make([]string, 0, 1+len(c.SourceIDs))
		if d := ClassifyDomain(c.TargetID); d != "" {
			neighborhood = append(neighborhood, d)
		}
		for _, src := range c.SourceIDs {
			if d := ClassifyDomain(src); d != "" {
				neighborhood = append(neighborhood, d)
			}
		}

		participants := make([]string, 0, 1+len(c.SourceIDs))
		participants = append(participants, c.TargetID)
		participants = append(participants, c.SourceIDs...)

		for _, p := range participants {
			set, ok := domains[p]
			if !ok {
				continue
			}
			for _, d := range neighborhood {
				set[d] = struct{}{}
			}
		}
	}

	out := make([]BridgeNode, 0, len(domains))
	for i := range inv.Anchors {
		a := &inv.Anchors[i]
		if excludeAxioms && a.Tier == model.TierAxiom {
			continue
		}
		set := domains[a.ID]
		domList := make([]string, 0, len(set))
		for d := range set {
			domList = append(domList, d)
		}
		sort.Strings(domList)
		out = append(out, BridgeNode{
			Domains:     domList,
			ID:          a.ID,
			DomainCount: len(domList),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].DomainCount != out[j].DomainCount {
			return out[i].DomainCount > out[j].DomainCount
		}
		return out[i].ID < out[j].ID
	})
	return out
}
