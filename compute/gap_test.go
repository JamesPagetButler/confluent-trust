package compute

import (
	"math"
	"sort"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Test fixture builder for gap tests. Construction notes:
//
//   The Issue #9 acceptance line — "f(0) remains top eddy. INST-Jc has
//   proximity ≈ 0 (irreducible). Weighted ranking differs from unweighted
//   for at least one input." — references the QBP v3.2 fixture, which is
//   not yet committed (qbp_v3_2.json absent). We encode the *shape* of
//   that acceptance against a synthetic inventory:
//
//     - top-eddy preservation: an input with the smallest weighted gap
//       and largest η is the highest-ranked entry,
//     - INST-Jc analogue: an input fronted by a step_type=irreducible
//       chain has Proximity == 0, and
//     - weighted ≠ unweighted: a constructed input where the unweighted
//       (count-of-steps) ranking differs from the weighted ranking.
//
//   When qbp_v3_2.json is added, an integration test should pin the real
//   acceptance values; this synthetic test pins the algorithmic contract.

const (
	gapInputCheap      = "INST-cheap"
	gapInputModerate   = "INST-moderate"
	gapInputBlocked    = "INST-Jc"
	gapInputManyEasy   = "INST-many-easy"
	gapAnchorProven    = "P-proven"
	gapAnchorProvenAlt = "P-proven-alt"
)

// buildGapFixture constructs a synthetic inventory with four inputs:
//   - cheap:      one routine_lean step → fastest weighted gap.
//   - moderate:   two novel_tractable steps → mid weighted gap.
//   - blocked:    one irreducible step → unreachable, π_w == 0.
//   - many-easy:  three routine_lean steps → highest unweighted-count
//     distance, but lowest weighted distance among reachable
//     inputs (so unweighted and weighted disagree).
func buildGapFixture() model.Inventory {
	zero := 0
	provenAnchor := model.Anchor{
		ID:         gapAnchorProven,
		Tier:       model.TierProof,
		Status:     model.StatusCoherent,
		SorryCount: &zero,
	}
	provenAlt := model.Anchor{
		ID:         gapAnchorProvenAlt,
		Tier:       model.TierProof,
		Status:     model.StatusCoherent,
		SorryCount: &zero,
	}
	return model.Inventory{
		Programme: "synthetic",
		Version:   "0.0.1",
		Axioms:    []model.Axiom{{ID: testAxiomID}},
		Anchors:   []model.Anchor{provenAnchor, provenAlt},
		Inputs: []model.Input{
			{ID: gapInputCheap, Type: testInputType, Status: testInputStatus, SignificantFigures: 3},
			{ID: gapInputModerate, Type: testInputType, Status: testInputStatus, SignificantFigures: 3},
			{ID: gapInputBlocked, Type: testInputType, Status: testInputStatus, SignificantFigures: 3},
			{ID: gapInputManyEasy, Type: testInputType, Status: testInputStatus, SignificantFigures: 3},
		},
		Chains: []model.Chain{
			{
				ID:        "C-cheap",
				Name:      "cheap",
				SourceIDs: []string{gapInputCheap},
				TargetID:  gapAnchorProven,
				StepTypes: []string{StepRoutineLean}, // weight 0.1
				Steps:     1,
				Status:    model.StatusCoherent,
			},
			{
				ID:        "C-moderate",
				Name:      "moderate",
				SourceIDs: []string{gapInputModerate},
				TargetID:  gapAnchorProven,
				StepTypes: []string{StepNovelTractable, StepNovelTractable}, // weight 1.0
				Steps:     2,
				Status:    model.StatusCoherent,
			},
			{
				ID:        "C-blocked",
				Name:      "blocked",
				SourceIDs: []string{gapInputBlocked},
				TargetID:  gapAnchorProven,
				StepTypes: []string{StepIrreducible}, // weight +Inf
				Steps:     1,
				Status:    model.StatusCoherent,
			},
			{
				ID:        "C-many-easy",
				Name:      "many easy",
				SourceIDs: []string{gapInputManyEasy},
				TargetID:  gapAnchorProvenAlt,
				StepTypes: []string{StepRoutineLean, StepRoutineLean, StepRoutineLean}, // count 3, weight 0.3
				Steps:     3,
				Status:    model.StatusCoherent,
			},
		},
	}
}

func TestStepDifficulty_Table(t *testing.T) {
	cases := map[string]float64{
		StepRoutineLean:    0.1,
		StepNovelTractable: 0.5,
		StepOpenPlausible:  1.0,
		StepOpenUnknown:    5.0,
	}
	for label, want := range cases {
		if got := StepDifficulty(label); got != want {
			t.Errorf("StepDifficulty(%q) = %v, want %v", label, got, want)
		}
	}
	if got := StepDifficulty(StepIrreducible); !math.IsInf(got, 1) {
		t.Errorf("StepDifficulty(irreducible) = %v, want +Inf", got)
	}
}

func TestStepDifficulty_UnknownDefaultsToPlausible(t *testing.T) {
	if got := StepDifficulty("not_a_category"); got != 1.0 {
		t.Errorf("unknown default = %v, want 1.0 (open_plausible)", got)
	}
}

func TestWeightedGap_ReachesProvenAnchor(t *testing.T) {
	inv := buildGapFixture()
	gap, nearest := WeightedGap(gapInputCheap, inv)
	if math.Abs(gap-0.1) > 1e-9 {
		t.Errorf("cheap gap: got %v, want 0.1", gap)
	}
	if nearest != gapAnchorProven {
		t.Errorf("cheap nearest: got %q, want %q", nearest, gapAnchorProven)
	}

	gap, nearest = WeightedGap(gapInputModerate, inv)
	if math.Abs(gap-1.0) > 1e-9 {
		t.Errorf("moderate gap: got %v, want 1.0", gap)
	}
	if nearest != gapAnchorProven {
		t.Errorf("moderate nearest: got %q", nearest)
	}
}

func TestWeightedGap_IrreducibleIsUnreachable(t *testing.T) {
	inv := buildGapFixture()
	// Acceptance: INST-Jc has proximity ≈ 0 (irreducible) — encoded here
	// as: gap is +Inf and EddyProximity is exactly 0.
	gap, nearest := WeightedGap(gapInputBlocked, inv)
	if !math.IsInf(gap, 1) {
		t.Errorf("blocked gap: got %v, want +Inf", gap)
	}
	if nearest != "" {
		t.Errorf("blocked nearest: got %q, want empty", nearest)
	}
}

func TestEddyProximity_Irreducible(t *testing.T) {
	inv := buildGapFixture()
	// Acceptance assertion: irreducible-fronted input has π_w = 0.
	if got := EddyProximity(gapInputBlocked, inv); got != 0 {
		t.Errorf("blocked π_w: got %v, want 0", got)
	}
}

func TestEddyProximity_FiniteForReachable(t *testing.T) {
	inv := buildGapFixture()
	// η = InputEntropy(3); g_w(cheap) = 0.1 → π_w = 10 * η.
	want := InputEntropy(3) / 0.1
	if got := EddyProximity(gapInputCheap, inv); math.Abs(got-want) > 1e-9 {
		t.Errorf("cheap π_w: got %v, want %v", got, want)
	}
}

func TestEddyProximity_UnknownInputReturnsZero(t *testing.T) {
	inv := buildGapFixture()
	if got := EddyProximity("INST-does-not-exist", inv); got != 0 {
		t.Errorf("unknown id: got %v, want 0", got)
	}
}

func TestRankEddies_DescendingByProximity(t *testing.T) {
	inv := buildGapFixture()
	ranked := RankEddies(inv)
	if len(ranked) != 4 {
		t.Fatalf("len: got %d, want 4", len(ranked))
	}
	// Verify descending order.
	for i := 1; i < len(ranked); i++ {
		if ranked[i-1].Proximity < ranked[i].Proximity {
			t.Errorf("not descending at index %d: %v < %v",
				i, ranked[i-1].Proximity, ranked[i].Proximity)
		}
	}
	// Top entry must be a finite-gap eddy (the synthetic equivalent of
	// the "f(0) remains top eddy" acceptance line).
	if math.IsInf(ranked[0].Gap, 1) {
		t.Errorf("top eddy gap is +Inf — should be finite")
	}
	// Blocked input lands at the bottom with Proximity 0.
	last := ranked[len(ranked)-1]
	if last.InputID != gapInputBlocked {
		t.Errorf("last entry id: got %q, want %q", last.InputID, gapInputBlocked)
	}
	if last.Proximity != 0 {
		t.Errorf("last entry proximity: got %v, want 0", last.Proximity)
	}
}

func TestRankEddies_TopEddyIsCheapest(t *testing.T) {
	// The "f(0) remains top eddy" acceptance line — encoded for the
	// synthetic fixture as: the input with the smallest weighted gap
	// (highest π_w given equal η) is rank 0.
	inv := buildGapFixture()
	ranked := RankEddies(inv)
	if ranked[0].InputID != gapInputCheap {
		t.Errorf("top eddy: got %q, want %q", ranked[0].InputID, gapInputCheap)
	}
}

// unweightedGap counts step_types as the gap (legacy Python contract) so
// the test below can demonstrate weighted ≠ unweighted.
func unweightedGap(inputID string, inv model.Inventory) float64 {
	// Forward BFS on chain count.
	adj := chainsBySource(inv)
	anchors := anchorByID(inv)
	type entry struct {
		id   string
		dist float64
	}
	dist := map[string]float64{inputID: 0}
	frontier := []entry{{inputID, 0}}
	best := math.Inf(1)
	for len(frontier) > 0 {
		minIdx := 0
		for i := 1; i < len(frontier); i++ {
			if frontier[i].dist < frontier[minIdx].dist {
				minIdx = i
			}
		}
		cur := frontier[minIdx]
		frontier = append(frontier[:minIdx], frontier[minIdx+1:]...)
		if d, ok := dist[cur.id]; ok && cur.dist > d {
			continue
		}
		if cur.dist >= best {
			continue
		}
		if a, ok := anchors[cur.id]; ok && isProvenAnchor(a) {
			best = cur.dist
			continue
		}
		for _, c := range adj[cur.id] {
			steps := len(c.StepTypes)
			if steps == 0 {
				steps = c.Steps
			}
			nd := cur.dist + float64(steps)
			if nd >= best {
				continue
			}
			if old, seen := dist[c.TargetID]; !seen || nd < old {
				dist[c.TargetID] = nd
				frontier = append(frontier, entry{c.TargetID, nd})
			}
		}
	}
	return best
}

func TestRankEddies_WeightedDiffersFromUnweighted(t *testing.T) {
	// Acceptance: "Weighted ranking differs from unweighted for at least
	// one input." Construction:
	//   gapInputManyEasy has 3 routine_lean steps:
	//     unweighted distance = 3
	//     weighted distance   = 0.3
	//   gapInputModerate has 2 novel_tractable steps:
	//     unweighted distance = 2  (closer)
	//     weighted distance   = 1.0 (further)
	// So the unweighted ranker prefers gapInputModerate over gapInputManyEasy,
	// but the weighted ranker prefers gapInputManyEasy.

	inv := buildGapFixture()

	weightedRank := RankEddies(inv)
	weightedPos := map[string]int{}
	for i, r := range weightedRank {
		weightedPos[r.InputID] = i
	}

	type unwRow struct {
		id    string
		score float64 // η/unweighted-gap, or 0 for unreachable
	}
	rows := make([]unwRow, 0, len(inv.Inputs))
	for _, in := range inv.Inputs {
		uw := unweightedGap(in.ID, inv)
		var s float64
		if !math.IsInf(uw, 1) && uw > 0 {
			s = InputEntropy(in.SignificantFigures) / uw
		}
		rows = append(rows, unwRow{in.ID, s})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		return rows[i].id < rows[j].id
	})

	differs := false
	for i, r := range rows {
		if weightedRank[i].InputID != r.id {
			differs = true
			break
		}
	}
	if !differs {
		t.Fatalf("weighted ranking %v matches unweighted ranking %v — "+
			"fixture should force divergence", weightedRank, rows)
	}

	// Pin the specific divergence: gapInputManyEasy ranks above
	// gapInputModerate under weighted but below under unweighted.
	if weightedPos[gapInputManyEasy] >= weightedPos[gapInputModerate] {
		t.Errorf("weighted: many-easy rank %d, moderate rank %d — "+
			"many-easy should rank higher",
			weightedPos[gapInputManyEasy], weightedPos[gapInputModerate])
	}
	uwPos := map[string]int{}
	for i, r := range rows {
		uwPos[r.id] = i
	}
	if uwPos[gapInputManyEasy] <= uwPos[gapInputModerate] {
		t.Errorf("unweighted: many-easy rank %d, moderate rank %d — "+
			"moderate should rank higher under unweighted",
			uwPos[gapInputManyEasy], uwPos[gapInputModerate])
	}
}

func TestWeightedGap_NoChainsReturnsInf(t *testing.T) {
	inv := model.Inventory{
		Programme: "lonely",
		Version:   "0.0.1",
		Inputs:    []model.Input{{ID: "INST-orphan", Type: testInputType, Status: testInputStatus, SignificantFigures: 3}},
	}
	gap, nearest := WeightedGap("INST-orphan", inv)
	if !math.IsInf(gap, 1) {
		t.Errorf("orphan gap: got %v, want +Inf", gap)
	}
	if nearest != "" {
		t.Errorf("orphan nearest: got %q, want empty", nearest)
	}
}
