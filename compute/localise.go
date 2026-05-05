package compute

import (
	"sort"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// LocalisationResult is the output of LocaliseIncoherence (Theory v0.2
// Method 6). Field order is chosen for govet -fieldalignment.
type LocalisationResult struct {
	AnchorID               string
	SegmentStart           string
	SegmentEnd             string
	LastCoherentConfluence string
	WeakestLinkID          string
	SegmentLength          int
}

// confluenceTargets returns the set of anchor IDs that are the
// AnchorID of at least one ConfluencePoint in the inventory. These are
// the "checkpoints" used to bound the localisation segment.
func confluenceTargets(inv model.Inventory) map[string]struct{} {
	m := make(map[string]struct{}, len(inv.ConfluencePoints))
	for _, cp := range inv.ConfluencePoints {
		m[cp.AnchorID] = struct{}{}
	}
	return m
}

// chainsByTarget indexes inv.Chains by TargetID — used to look up the
// incoming chain(s) for a given anchor when computing the weakest link.
func chainsByTarget(inv model.Inventory) map[string][]model.Chain {
	m := make(map[string][]model.Chain, len(inv.Chains))
	for _, c := range inv.Chains {
		m[c.TargetID] = append(m[c.TargetID], c)
	}
	return m
}

// minIncomingFidelity returns the minimum ChainFidelity over the chains
// whose TargetID is anchorID. Returns (1.0, false) when no chains target
// the anchor (e.g. an axiom or an input).
func minIncomingFidelity(anchorID string, byTarget map[string][]model.Chain) (float64, bool) {
	chains := byTarget[anchorID]
	if len(chains) == 0 {
		return 1.0, false
	}
	mu := ChainFidelity(chains[0])
	for _, c := range chains[1:] {
		if v := ChainFidelity(c); v < mu {
			mu = v
		}
	}
	return mu, true
}

// LocaliseIncoherence implements Method 6: starting from anchorID,
// walk backward through the predecessor graph (each anchor's
// prediction_chain) and identify
//
//   - the last coherent confluence-target encountered (a "checkpoint"
//     where multiple chains independently agreed),
//   - the segment from that checkpoint forward to anchorID, and
//   - the anchor in that segment whose incoming chain has the smallest
//     ChainFidelity (the "weakest link").
//
// When no coherent confluence exists in the ancestry, SegmentStart is
// the deepest reachable ancestor and LastCoherentConfluence is empty.
// When the segment contains only the failing anchor itself,
// WeakestLinkID equals AnchorID.
func LocaliseIncoherence(anchorID string, inv model.Inventory) LocalisationResult {
	anchors := anchorByID(inv)
	checkpoints := confluenceTargets(inv)
	byTarget := chainsByTarget(inv)

	// Backward BFS recording the ancestor order we first saw each id.
	visited := map[string]struct{}{anchorID: {}}
	// parent[i] is the immediate descendant we walked from to reach i —
	// the ancestor-of relationship in reverse, used to reconstruct the
	// path from a chosen checkpoint forward to anchorID.
	parent := map[string]string{}
	queue := []string{anchorID}
	order := []string{anchorID}

	var lastCheckpoint string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		// Checkpoint test: a coherent anchor that is also a confluence
		// target. We accept the *first* such ancestor we hit (BFS order
		// = nearest-first walking backward), which matches the "last
		// coherent confluence on the path from origin to failure".
		if cur != anchorID {
			if a, ok := anchors[cur]; ok && a.Status == model.StatusCoherent {
				if _, isCheckpoint := checkpoints[cur]; isCheckpoint {
					if lastCheckpoint == "" {
						lastCheckpoint = cur
					}
				}
			}
		}

		a, ok := anchors[cur]
		if !ok {
			continue
		}
		for _, predID := range a.PredictionChain {
			if _, seen := visited[predID]; seen {
				continue
			}
			visited[predID] = struct{}{}
			parent[predID] = cur
			queue = append(queue, predID)
			order = append(order, predID)
		}
	}

	res := LocalisationResult{
		AnchorID:               anchorID,
		SegmentEnd:             anchorID,
		LastCoherentConfluence: lastCheckpoint,
	}

	// Determine SegmentStart and the segment-anchor list.
	var segment []string
	switch {
	case lastCheckpoint != "":
		res.SegmentStart = lastCheckpoint
		// Walk forward via parent[] from the checkpoint to anchorID.
		segment = []string{lastCheckpoint}
		for cur := lastCheckpoint; cur != anchorID; {
			next, ok := parent[cur]
			if !ok {
				// Defensive: parent chain broken; bail out with what
				// we have so we never loop forever.
				break
			}
			segment = append(segment, next)
			cur = next
		}
	case len(order) > 1:
		// No checkpoint: SegmentStart is the deepest ancestor reached
		// (last seen by BFS). Segment is the parent chain from that
		// ancestor up to anchorID.
		deepest := order[len(order)-1]
		res.SegmentStart = deepest
		segment = []string{deepest}
		for cur := deepest; cur != anchorID; {
			next, ok := parent[cur]
			if !ok {
				break
			}
			segment = append(segment, next)
			cur = next
		}
	default:
		// Singleton segment: the failing anchor alone.
		res.SegmentStart = anchorID
		segment = []string{anchorID}
	}

	res.SegmentLength = len(segment) - 1
	if res.SegmentLength < 0 {
		res.SegmentLength = 0
	}

	// WeakestLinkID: minimum incoming-chain fidelity within the segment.
	// Singleton segment ⇒ WeakestLinkID == anchorID.
	if len(segment) == 1 {
		res.WeakestLinkID = anchorID
		return res
	}

	bestFid := 1.1 // larger than any clamped fidelity
	var candidates []string
	for _, id := range segment {
		mu, hasIncoming := minIncomingFidelity(id, byTarget)
		if !hasIncoming {
			continue
		}
		if mu < bestFid {
			bestFid = mu
			candidates = []string{id}
		} else if mu == bestFid {
			candidates = append(candidates, id)
		}
	}
	switch len(candidates) {
	case 0:
		// No segment anchor has an incoming chain — fall back to the
		// failing anchor.
		res.WeakestLinkID = anchorID
	case 1:
		res.WeakestLinkID = candidates[0]
	default:
		sort.Strings(candidates)
		res.WeakestLinkID = candidates[0]
	}
	return res
}
