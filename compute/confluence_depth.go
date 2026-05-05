package compute

import "github.com/JamesPagetButler/confluent-trust/model"

// confluenceDepthMaxHops caps the upstream BFS depth when accumulating
// confluence depth contributions. Real CTH inventories are far below 10
// in chain depth; the cap prevents unbounded traversal on pathological
// cycles.
const confluenceDepthMaxHops = 10

// AnchorConfluenceDepth returns the per-anchor arity-weighted confluence
// depth per Theory v0.2 §4.7. For each anchor v, the depth is
//
//	depth(v) = Σ over (d ∈ deps(v) ∪ {v}) of (|R_d| − 1) when d is the
//	target of a confluence with N paths R_d
//
// In practice: an anchor downstream of a single 3-way confluence
// contributes 2 to its depth; downstream of two 2-way confluences
// contributes 1 + 1 = 2. The arity weight replaces the binary "is
// confluence target" count from the v0.1 metric.
//
// Anchors with no confluence-target dependencies have depth 0.
func AnchorConfluenceDepth(inv model.Inventory) map[string]int {
	if len(inv.Anchors) == 0 {
		return map[string]int{}
	}

	// Per-anchor confluence weight: the (arity − 1) contribution that an
	// anchor's *own* node carries when something downstream depends on it.
	weight := make(map[string]int, len(inv.ConfluencePoints))
	for _, cp := range inv.ConfluencePoints {
		w := len(cp.Paths) - 1
		if w < 0 {
			w = 0
		}
		weight[cp.AnchorID] += w
	}

	// Index chains by target_id so we can walk upstream from any anchor.
	chainByTarget := make(map[string][]model.Chain, len(inv.Chains))
	for _, c := range inv.Chains {
		chainByTarget[c.TargetID] = append(chainByTarget[c.TargetID], c)
	}

	out := make(map[string]int, len(inv.Anchors))
	for _, a := range inv.Anchors {
		out[a.ID] = depthFor(a.ID, weight, chainByTarget)
	}
	return out
}

// ChainConfluenceDepth returns the depth of each chain's TargetID. It is
// a thin convenience over AnchorConfluenceDepth + a chain-id remap.
func ChainConfluenceDepth(inv model.Inventory) map[string]int {
	anchorDepth := AnchorConfluenceDepth(inv)
	out := make(map[string]int, len(inv.Chains))
	for _, c := range inv.Chains {
		out[c.ID] = anchorDepth[c.TargetID]
	}
	return out
}

// depthFor sums confluence weights along the upstream BFS frontier from
// startID, including startID's own weight. Visits each node at most once
// to keep the traversal O(|inputs| + |chains|).
func depthFor(startID string, weight map[string]int, chainByTarget map[string][]model.Chain) int {
	visited := make(map[string]struct{})
	frontier := []string{startID}

	total := 0
	for hop := 0; hop < confluenceDepthMaxHops && len(frontier) > 0; hop++ {
		next := make([]string, 0)
		for _, id := range frontier {
			if _, dup := visited[id]; dup {
				continue
			}
			visited[id] = struct{}{}
			total += weight[id]
			for _, c := range chainByTarget[id] {
				next = append(next, c.SourceIDs...)
			}
		}
		frontier = next
	}
	return total
}
