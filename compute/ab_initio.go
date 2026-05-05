package compute

import (
	"math"
	"sort"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// abInitioFidelityTolerance is the |Δμ| window inside which two chain
// fidelities are treated as comparable; ties are broken by input count.
// Per Issue #20 / R7 the engine "prefers lower-deficit path only when
// fidelities are comparable".
const abInitioFidelityTolerance = 0.01

// abInitioMaxHops caps the BFS depth when counting upstream input
// dependencies for a chain. Realistic CTH inventories have chain depth
// far below 10; the cap keeps this O(chains) on pathological cycles.
const abInitioMaxHops = 10

// AbInitioCandidate is one chain's score for a multi-path target.
type AbInitioCandidate struct {
	ChainID    string
	Fidelity   float64
	Score      float64
	InputCount int
}

// AbInitioResult is one multi-path target's full ranking. BestChainID is
// the winner under the score-then-fewest-inputs rule (deterministic:
// when even input counts tie, the chain with the lower ID wins so the
// output is reproducible).
type AbInitioResult struct {
	Candidates  []AbInitioCandidate
	TargetID    string
	BestChainID string
	BestScore   float64
}

// AbInitioScore implements R7 from the Python engine: for every anchor
// that is the target of two or more chains, score each contributing
// chain by μ / (1 + input_count) where input_count is the number of
// distinct upstream input parameters reachable from the chain's sources
// (BFS, depth cap abInitioMaxHops).
//
// Tiebreaker (R7): when two chains' fidelities are within
// abInitioFidelityTolerance of each other, the chain with the lower
// input count wins ("prefer lower-deficit path only when fidelities are
// comparable"). When that also ties, the chain with the lower ID wins
// so the function is deterministic.
//
// The result list contains one AbInitioResult per multi-path target,
// sorted by TargetID.
func AbInitioScore(inv model.Inventory) []AbInitioResult {
	if len(inv.Chains) == 0 {
		return nil
	}

	// chainsByTarget: target_id → list of chain indices into inv.Chains.
	chainsByTarget := make(map[string][]int, len(inv.Chains))
	// chainsByID: chain id → chain (for upstream BFS).
	chainsByID := make(map[string]model.Chain, len(inv.Chains))
	// targetToChain: anchor target_id → chain producing it (for the
	// upstream traversal: when a frontier id is the target of some
	// chain, its sources become the next frontier).
	targetToChain := make(map[string]model.Chain, len(inv.Chains))
	for i, c := range inv.Chains {
		chainsByTarget[c.TargetID] = append(chainsByTarget[c.TargetID], i)
		chainsByID[c.ID] = c
		targetToChain[c.TargetID] = c
	}

	inputSet := inputIDs(inv)

	// Walk in inventory order so multi-path targets emerge deterministically;
	// final slice is sorted by TargetID below.
	results := make([]AbInitioResult, 0)
	seenTarget := make(map[string]struct{}, len(chainsByTarget))
	for _, c := range inv.Chains {
		if _, dup := seenTarget[c.TargetID]; dup {
			continue
		}
		idxs := chainsByTarget[c.TargetID]
		if len(idxs) < 2 {
			continue
		}
		seenTarget[c.TargetID] = struct{}{}

		candidates := make([]AbInitioCandidate, 0, len(idxs))
		for _, i := range idxs {
			cand := inv.Chains[i]
			mu := ChainFidelity(cand)
			ic := countUpstreamInputs(cand, targetToChain, inputSet)
			candidates = append(candidates, AbInitioCandidate{
				ChainID:    cand.ID,
				Fidelity:   mu,
				InputCount: ic,
				Score:      mu / float64(1+ic),
			})
		}

		best := pickBest(candidates)
		results = append(results, AbInitioResult{
			TargetID:    c.TargetID,
			BestChainID: best.ChainID,
			BestScore:   best.Score,
			Candidates:  candidates,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TargetID < results[j].TargetID
	})
	return results
}

// pickBest applies the R7 selection rule. Caller guarantees len ≥ 1.
//
//  1. Highest raw Score wins.
//  2. If two top scores are within abInitioFidelityTolerance on Fidelity,
//     the one with fewer InputCount wins (lower deficit).
//  3. If still tied, the lower ChainID wins (determinism).
func pickBest(cands []AbInitioCandidate) AbInitioCandidate {
	best := cands[0]
	for _, c := range cands[1:] {
		switch {
		case c.Score > best.Score+1e-12:
			// Strict score win — but if fidelities are comparable we
			// need to fall back to the deficit tiebreaker.
			if math.Abs(c.Fidelity-best.Fidelity) <= abInitioFidelityTolerance {
				if c.InputCount < best.InputCount {
					best = c
				} else if c.InputCount == best.InputCount && c.ChainID < best.ChainID {
					best = c
				}
				// else: keep best
			} else {
				best = c
			}
		case c.Score < best.Score-1e-12:
			// Strict loss — but again, comparable fidelities can flip.
			if math.Abs(c.Fidelity-best.Fidelity) <= abInitioFidelityTolerance {
				if c.InputCount < best.InputCount {
					best = c
				}
			}
		default:
			// Effectively equal score; resolve by inputs then id.
			if c.InputCount < best.InputCount {
				best = c
			} else if c.InputCount == best.InputCount && c.ChainID < best.ChainID {
				best = c
			}
		}
	}
	return best
}

// countUpstreamInputs runs a depth-capped BFS upstream from the chain's
// SourceIDs, counting distinct input parameters along the way. The
// frontier contains node IDs (axiom/anchor/input/derived-principle ids);
// when a frontier id matches an input, it is counted; when it matches
// the target of another chain in the inventory, that chain's source ids
// become the next frontier.
func countUpstreamInputs(
	c model.Chain,
	targetToChain map[string]model.Chain,
	inputSet map[string]struct{},
) int {
	if len(c.SourceIDs) == 0 {
		return 0
	}

	visited := make(map[string]struct{}, len(c.SourceIDs))
	frontier := append([]string(nil), c.SourceIDs...)
	count := 0

	for hop := 0; hop < abInitioMaxHops && len(frontier) > 0; hop++ {
		next := make([]string, 0)
		for _, id := range frontier {
			if _, seen := visited[id]; seen {
				continue
			}
			visited[id] = struct{}{}

			if _, isInput := inputSet[id]; isInput {
				count++
				continue
			}
			if upstream, ok := targetToChain[id]; ok {
				next = append(next, upstream.SourceIDs...)
			}
		}
		frontier = next
	}
	return count
}
