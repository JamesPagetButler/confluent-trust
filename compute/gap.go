package compute

import (
	"math"
	"sort"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Step-category labels for the §2.7 Definition 16 difficulty table.
// Exported so inventory authors can reference them rather than embedding
// raw strings in their data.
const (
	StepRoutineLean       = "routine_lean"
	StepNovelTractable    = "novel_tractable"
	StepOpenPlausible     = "open_plausible"
	StepOpenUnknown       = "open_unknown"
	StepIrreducible       = "irreducible"
	defaultStepDifficulty = 1.0
)

// stepDifficultyTable encodes the §2.7 Definition 16 weights. An entry of
// math.Inf(+1) marks an irreducible step (no path can cross it).
var stepDifficultyTable = map[string]float64{
	StepRoutineLean:    0.1,
	StepNovelTractable: 0.5,
	StepOpenPlausible:  1.0,
	StepOpenUnknown:    5.0,
	StepIrreducible:    math.Inf(1),
}

// StepDifficulty returns w(e) for a derivation step type per the §2.7
// difficulty table. Unknown labels return defaultStepDifficulty (1.0,
// "open_plausible") so callers do not silently get free crossings.
func StepDifficulty(stepCategory string) float64 {
	if v, ok := stepDifficultyTable[stepCategory]; ok {
		return v
	}
	return defaultStepDifficulty
}

// EddyRanking is one row of the eddy-proximity ranking returned by
// RankEddies. Field order is chosen for govet -fieldalignment.
type EddyRanking struct {
	NearestProven string
	InputID       string
	Gap           float64
	Proximity     float64
}

// chainWeight returns the total weighted difficulty of a chain. When
// step_types is populated, sum StepDifficulty over each step. When absent,
// fall back to len(step_types)*1.0 if non-zero, else c.Steps*1.0 (default
// "open_plausible" per step).
func chainWeight(c model.Chain) float64 {
	if len(c.StepTypes) > 0 {
		var w float64
		for _, st := range c.StepTypes {
			w += StepDifficulty(st)
		}
		return w
	}
	if c.Steps > 0 {
		return float64(c.Steps) * defaultStepDifficulty
	}
	return defaultStepDifficulty
}

// anchorByID indexes inv.Anchors so the BFS can decide whether a node
// satisfies the "proven" stop condition without rescanning.
func anchorByID(inv model.Inventory) map[string]model.Anchor {
	m := make(map[string]model.Anchor, len(inv.Anchors))
	for _, a := range inv.Anchors {
		m[a.ID] = a
	}
	return m
}

// chainsBySource indexes inv.Chains by every entry in source_ids — the
// adjacency list for the forward BFS in WeightedGap.
func chainsBySource(inv model.Inventory) map[string][]model.Chain {
	m := make(map[string][]model.Chain, len(inv.Chains))
	for _, c := range inv.Chains {
		for _, src := range c.SourceIDs {
			m[src] = append(m[src], c)
		}
	}
	return m
}

// isProvenAnchor reports whether a is a complete Tier-1 proof anchor —
// the BFS stop condition for WeightedGap.
func isProvenAnchor(a model.Anchor) bool {
	if a.Tier != model.TierProof {
		return false
	}
	if a.Status != model.StatusCoherent {
		return false
	}
	if a.SorryCount != nil && *a.SorryCount != 0 {
		return false
	}
	return true
}

// WeightedGap returns g_w(input) per Definition 16: the minimum sum of
// step-difficulty weights along any path from inputID forward through
// chains to a complete Tier-1 proof anchor. The second return is the ID
// of the proven anchor reached. When unreachable, returns (+Inf, "").
//
// The algorithm is Dijkstra (uniform-cost search) over the chain
// hypergraph: each chain edge contributes chainWeight(c) to the cost of
// reaching c.TargetID from any of its source IDs. Irreducible step types
// produce +Inf weights that the relaxation naturally excludes.
func WeightedGap(inputID string, inv model.Inventory) (float64, string) {
	anchors := anchorByID(inv)
	adj := chainsBySource(inv)

	// dist[id] = minimum weighted distance from inputID to id.
	dist := map[string]float64{inputID: 0}
	// Simple priority queue via sorted slice — inventories are O(100)
	// nodes; a heap is overkill and would complicate fieldalignment.
	type entry struct {
		id   string
		dist float64
	}
	frontier := []entry{{inputID, 0}}

	bestProven := ""
	bestDist := math.Inf(1)

	for len(frontier) > 0 {
		// Pop minimum.
		minIdx := 0
		for i := 1; i < len(frontier); i++ {
			if frontier[i].dist < frontier[minIdx].dist {
				minIdx = i
			}
		}
		cur := frontier[minIdx]
		frontier = append(frontier[:minIdx], frontier[minIdx+1:]...)

		// Stale entry check.
		if d, ok := dist[cur.id]; ok && cur.dist > d {
			continue
		}
		if cur.dist >= bestDist {
			// Can no longer improve the answer.
			continue
		}

		// Goal test: complete Tier-1 proof.
		if a, ok := anchors[cur.id]; ok && isProvenAnchor(a) {
			if cur.dist < bestDist {
				bestDist = cur.dist
				bestProven = cur.id
			}
			continue
		}

		// Relax outgoing chains.
		for _, c := range adj[cur.id] {
			w := chainWeight(c)
			if math.IsInf(w, 1) {
				continue
			}
			nd := cur.dist + w
			if nd >= bestDist {
				continue
			}
			if old, seen := dist[c.TargetID]; !seen || nd < old {
				dist[c.TargetID] = nd
				frontier = append(frontier, entry{c.TargetID, nd})
			}
		}
	}

	if math.IsInf(bestDist, 1) {
		return math.Inf(1), ""
	}
	return bestDist, bestProven
}

// EddyProximity returns the Definition 17 weighted eddy proximity:
//
//	π_w(input) = η(input) / g_w(input)
//
// where η is the input's residual entropy (bits) and g_w is its weighted
// gap. When the gap is +Inf (unreachable, or guarded by an irreducible
// step), π_w is 0 — the input is irreducibly stuck and cannot benefit
// from incremental work.
func EddyProximity(inputID string, inv model.Inventory) float64 {
	var in model.Input
	found := false
	for _, candidate := range inv.Inputs {
		if candidate.ID == inputID {
			in = candidate
			found = true
			break
		}
	}
	if !found {
		return 0
	}
	gap, _ := WeightedGap(inputID, inv)
	if math.IsInf(gap, 1) {
		return 0
	}
	if gap <= 0 {
		// Gap of zero would mean the input itself is already proven —
		// not meaningful; return 0 to keep the ranking stable.
		return 0
	}
	return InputEntropy(in.SignificantFigures) / gap
}

// RankEddies returns one EddyRanking per input, sorted by Proximity
// descending (highest-value eddy first). Tiebreaks: InputID ascending.
// Inputs guarded by irreducible steps surface at the bottom with
// Proximity=0 and Gap=+Inf.
func RankEddies(inv model.Inventory) []EddyRanking {
	out := make([]EddyRanking, 0, len(inv.Inputs))
	for _, in := range inv.Inputs {
		gap, nearest := WeightedGap(in.ID, inv)
		var prox float64
		if !math.IsInf(gap, 1) && gap > 0 {
			prox = InputEntropy(in.SignificantFigures) / gap
		}
		out = append(out, EddyRanking{
			InputID:       in.ID,
			Gap:           gap,
			NearestProven: nearest,
			Proximity:     prox,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Proximity != out[j].Proximity {
			return out[i].Proximity > out[j].Proximity
		}
		return out[i].InputID < out[j].InputID
	})
	return out
}
