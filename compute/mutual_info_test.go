package compute

import (
	"math"
	"testing"
)

func TestPairwiseMI_PerfectAgreementIsFiniteLarge(t *testing.T) {
	// §4.5: when predA == predB, the formula would diverge without ε.
	got := PairwiseMI(1.0, 1.0, 0.1, 0.1)
	if math.IsInf(got, 0) || math.IsNaN(got) {
		t.Fatalf("got non-finite %v", got)
	}
	if got < 10 {
		t.Errorf("perfect agreement should produce a large MI; got %v", got)
	}
}

func TestPairwiseMI_LargeDisagreementIsSmall(t *testing.T) {
	// 100 σ apart with σ=1 → MI should be small.
	got := PairwiseMI(0.0, 100.0, 1.0, 1.0)
	if got > 0.05 {
		t.Errorf("large disagreement should produce small MI; got %v", got)
	}
}

func TestPairwiseMI_DegenerateChannelsReturnZero(t *testing.T) {
	if got := PairwiseMI(0, 1, 0, 0); got != 0 {
		t.Errorf("zero sigmas should return 0; got %v", got)
	}
}

func TestNaryMI_TwoPathsMatchesPairwise(t *testing.T) {
	preds := []float64{1.0, 1.05}
	sigs := []float64{0.1, 0.1}
	pairwise := PairwiseMI(preds[0], preds[1], sigs[0], sigs[1])
	nary := NaryMI(preds, sigs)
	if math.Abs(nary-pairwise) > 1e-9 {
		t.Errorf("N=2: got %v, want %v (= pairwise)", nary, pairwise)
	}
}

func TestNaryMI_ThreePathsAgreeing_ExceedsPairwiseSum(t *testing.T) {
	// Issue #6 acceptance: 3-way > sum of pairwise when all three agree.
	preds := []float64{1.0, 1.001, 0.999}
	sigs := []float64{0.1, 0.1, 0.1}

	var pairwiseSum float64
	for i := 0; i < len(preds); i++ {
		for j := i + 1; j < len(preds); j++ {
			pairwiseSum += PairwiseMI(preds[i], preds[j], sigs[i], sigs[j])
		}
	}
	nary := NaryMI(preds, sigs)
	if nary <= pairwiseSum {
		t.Errorf("3-way (%v) should exceed pairwise sum (%v) when all paths agree", nary, pairwiseSum)
	}
}

func TestNaryMI_DisagreementCollapsesSynergy(t *testing.T) {
	// When paths disagree strongly, the synergy bonus decays toward zero
	// and N-ary MI is dominated by (small) pairwise contributions.
	preds := []float64{0.0, 100.0, -100.0}
	sigs := []float64{1.0, 1.0, 1.0}
	got := NaryMI(preds, sigs)
	if got > 0.5 {
		t.Errorf("strong disagreement should produce tiny MI; got %v", got)
	}
}

func TestNaryMI_ReturnsZeroForUnderspecified(t *testing.T) {
	if got := NaryMI([]float64{1.0}, []float64{0.1}); got != 0 {
		t.Errorf("N=1 should return 0; got %v", got)
	}
	if got := NaryMI([]float64{1, 2}, []float64{0.1}); got != 0 {
		t.Errorf("mismatched lengths should return 0; got %v", got)
	}
}

func TestCappedMI(t *testing.T) {
	tests := []struct {
		name string
		caps []float64
		mi   float64
		want float64
	}{
		{"no caps disables", nil, 5.0, 5.0},
		{"cap binds", []float64{2.5, 4.0}, 10.0, 2.5},
		{"cap loose", []float64{6.0}, 4.0, 4.0},
		{"single cap", []float64{1.5}, 3.0, 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CappedMI(tt.mi, tt.caps); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStructuralMI_IsInteger(t *testing.T) {
	tests := []struct {
		name        string
		want        float64
		minCapacity float64
		arity       int
	}{
		{"3-way structural with loose cap", 3.0, 100.0, 3},
		{"3-way structural capped at 2", 2.0, 2.0, 3},
		{"2-way structural", 2.0, math.Inf(1), 2},
		{"zero arity", 0.0, math.Inf(1), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StructuralMI(tt.arity, tt.minCapacity)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipelineNoInfiniteValues(t *testing.T) {
	// Issue #6 acceptance: no infinite values across all combinators.
	preds := []float64{1.0, 1.0, 1.0, 1.0, 1.0}
	sigs := []float64{0.01, 0.01, 0.01, 0.01, 0.01}

	pw := PairwiseMI(preds[0], preds[1], sigs[0], sigs[1])
	nary := NaryMI(preds, sigs)
	capped := CappedMI(nary, []float64{8.0})
	struc := StructuralMI(5, math.Inf(1))

	for _, v := range []float64{pw, nary, capped, struc} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			t.Errorf("unexpected non-finite value %v", v)
		}
	}
}
