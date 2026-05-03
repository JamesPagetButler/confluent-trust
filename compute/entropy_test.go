package compute

import (
	"math"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int) *int           { return &v }

const (
	eps         = 1e-6
	testAxiomID = "AX-1"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) <= eps
}

func TestAxiomEntropy(t *testing.T) {
	a := model.Axiom{ID: "AXIOM-1"}

	if got := AxiomEntropy(a, nil); got != DefaultAxiomEntropyBits {
		t.Errorf("default: got %v, want %v", got, DefaultAxiomEntropyBits)
	}

	tab := map[string]float64{"AXIOM-1": 2.5}
	if got := AxiomEntropy(a, tab); got != 2.5 {
		t.Errorf("table: got %v, want %v", got, 2.5)
	}

	if got := AxiomEntropy(model.Axiom{ID: "AXIOM-Z"}, tab); got != DefaultAxiomEntropyBits {
		t.Errorf("missing-from-table: got %v, want default %v", got, DefaultAxiomEntropyBits)
	}
}

func TestResidualEntropy(t *testing.T) {
	tab := map[string]float64{testAxiomID: 3.0}

	tests := []struct {
		name   string
		anchor model.Anchor
		want   float64
	}{
		{
			name:   "tier 0 axiom from table",
			anchor: model.Anchor{ID: testAxiomID, Tier: model.TierAxiom},
			want:   3.0,
		},
		{
			name:   "tier 0 axiom default",
			anchor: model.Anchor{ID: "AX-9", Tier: model.TierAxiom},
			want:   DefaultAxiomEntropyBits,
		},
		{
			name:   "tier 1 complete proof",
			anchor: model.Anchor{ID: "P-1", Tier: model.TierProof, SorryCount: ptrInt(0)},
			want:   0.0,
		},
		{
			name:   "tier 1 incomplete proof has residual entropy",
			anchor: model.Anchor{ID: "P-2", Tier: model.TierProof, SorryCount: ptrInt(2)},
			want:   2.0,
		},
		{
			name:   "tier 2 exact match",
			anchor: model.Anchor{ID: "M-1", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(0.0)},
			want:   0.0,
		},
		{
			name:   "tier 2 with delta 0.01 (1%)",
			anchor: model.Anchor{ID: "M-2", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(1.0)},
			want:   -math.Log2(1 - 0.01),
		},
		{
			name:   "tier 2 with delta 0.5 (50%)",
			anchor: model.Anchor{ID: "M-3", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(50.0)},
			want:   1.0,
		},
		{
			name:   "tier 2 missing discrepancy treated as zero",
			anchor: model.Anchor{ID: "M-4", Tier: model.TierMeasurement},
			want:   0.0,
		},
		{
			name:   "tier 3 untested prediction returns zero baseline",
			anchor: model.Anchor{ID: "PRED-1", Tier: model.TierPrediction},
			want:   0.0,
		},
		{
			name:   "tier 2 with delta beyond 1.0 is capped, not infinite",
			anchor: model.Anchor{ID: "M-bad", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(150.0)},
			want:   1024.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResidualEntropy(tt.anchor, tab)
			if !approxEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if math.IsInf(got, 0) || math.IsNaN(got) {
				t.Errorf("got non-finite %v", got)
			}
		})
	}
}

func TestConfirmatoryInfo(t *testing.T) {
	tests := []struct {
		name   string
		anchor model.Anchor
		want   float64
	}{
		{
			name:   "tier 0 axiom carries no confirmation",
			anchor: model.Anchor{ID: testAxiomID, Tier: model.TierAxiom},
			want:   0.0,
		},
		{
			name:   "tier 1 proof carries no confirmation",
			anchor: model.Anchor{ID: "P-1", Tier: model.TierProof},
			want:   0.0,
		},
		{
			name:   "tier 2 structural match = 1 bit",
			anchor: model.Anchor{ID: "M-1", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(0.0)},
			want:   1.0,
		},
		{
			name:   "tier 2 missing discrepancy = 0 bits",
			anchor: model.Anchor{ID: "M-2", Tier: model.TierMeasurement},
			want:   0.0,
		},
		{
			name:   "tier 2 delta 0.01 = log2(100) ≈ 6.644 bits",
			anchor: model.Anchor{ID: "M-3", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(1.0)},
			want:   -math.Log2(0.01),
		},
		{
			name:   "tier 2 delta 0.5 = 1 bit",
			anchor: model.Anchor{ID: "M-4", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(50.0)},
			want:   1.0,
		},
		{
			name:   "tier 2 delta 0.9 < 1 bit floor",
			anchor: model.Anchor{ID: "M-5", Tier: model.TierMeasurement, DiscrepancyPct: ptrFloat(90.0)},
			want:   -math.Log2(0.9),
		},
		{
			name:   "tier 3 untested = 0 bits",
			anchor: model.Anchor{ID: "PRED-1", Tier: model.TierPrediction},
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConfirmatoryInfo(tt.anchor)
			if !approxEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntropyConfirmatoryDuality(t *testing.T) {
	// Theory v0.2 §2.5 Remark: a Tier-2 anchor with δ=0 has
	//   η(v) = 0  (no remaining uncertainty)
	//   ι(v) = 1  (one binary question answered correctly)
	// These are *not* contradictory.
	anchor := model.Anchor{
		ID: "M-zero", Tier: model.TierMeasurement,
		DiscrepancyPct: ptrFloat(0.0),
	}
	if got := ResidualEntropy(anchor, nil); got != 0.0 {
		t.Errorf("eta: got %v, want 0", got)
	}
	if got := ConfirmatoryInfo(anchor); got != 1.0 {
		t.Errorf("iota: got %v, want 1", got)
	}
}

func TestInputEntropy(t *testing.T) {
	tests := []struct {
		sf   int
		want float64
	}{
		{1, inputBitsPerSigFig},
		{3, 3 * inputBitsPerSigFig},
		{0, 3 * inputBitsPerSigFig},  // default
		{-1, 3 * inputBitsPerSigFig}, // default
	}
	for _, tt := range tests {
		if got := InputEntropy(tt.sf); !approxEqual(got, tt.want) {
			t.Errorf("InputEntropy(%d): got %v, want %v", tt.sf, got, tt.want)
		}
	}
}
