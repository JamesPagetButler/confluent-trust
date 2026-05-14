package compute

import (
	"errors"
	"math"
	"testing"
)

// TestScorePrediction_Scalar_NonZeroDelta covers the two scalar sub-cases:
//   - predicted=42.0, observed=41.7 → delta≈0.00714 (<1%) → RegimeLaminar
//   - predicted=42.0, observed=35.0 → delta≈0.1667 (10%–50%) → RegimeModerate
func TestScorePrediction_Scalar_NonZeroDelta(t *testing.T) {
	// Sub-case 1: small delta stays Laminar
	s, err := ScorePrediction(KindScalar, 42.0, 41.7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantDelta1 := 0.3 / 42.0 // ≈ 0.007142
	if math.Abs(s.Delta-wantDelta1) > 1e-9 {
		t.Errorf("Delta: got %v, want %v", s.Delta, wantDelta1)
	}
	if s.Delta == 0 {
		t.Errorf("expected non-zero Delta")
	}
	if s.Regime != ScoreRegimeLaminar {
		t.Errorf("Regime: got %v, want %v", s.Regime, ScoreRegimeLaminar)
	}
	wantDisc1 := 100.0 * wantDelta1
	if math.Abs(s.DiscrepancyPct-wantDisc1) > 1e-9 {
		t.Errorf("DiscrepancyPct: got %v, want %v", s.DiscrepancyPct, wantDisc1)
	}

	// Sub-case 2: moderate delta (10%–50%) → RegimeModerate
	s2, err := ScorePrediction(KindScalar, 42.0, 35.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantDelta2 := 7.0 / 42.0 // ≈ 0.16667
	if math.Abs(s2.Delta-wantDelta2) > 1e-9 {
		t.Errorf("Delta: got %v, want %v", s2.Delta, wantDelta2)
	}
	if s2.Regime != ScoreRegimeModerate {
		t.Errorf("Regime: got %v, want %v", s2.Regime, ScoreRegimeModerate)
	}
}

func TestScorePrediction_Categorical_Miss(t *testing.T) {
	s, err := ScorePrediction(KindCategorical, testCatA, testCatB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Delta != 1.0 {
		t.Errorf("Delta: got %v, want 1.0", s.Delta)
	}
	if s.Regime != ScoreRegimeHeavy {
		t.Errorf("Regime: got %v, want %v", s.Regime, ScoreRegimeHeavy)
	}
	if s.ConfirmInfo != 0.0 {
		t.Errorf("ConfirmInfo: got %v, want 0.0", s.ConfirmInfo)
	}
	if s.DiscrepancyPct != 100.0 {
		t.Errorf("DiscrepancyPct: got %v, want 100.0", s.DiscrepancyPct)
	}
	if s.Kind != KindCategorical {
		t.Errorf("Kind: got %v, want KindCategorical", s.Kind)
	}
}

func TestScorePrediction_Categorical_Match(t *testing.T) {
	s, err := ScorePrediction(KindCategorical, testCatA, testCatA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Delta != 0.0 {
		t.Errorf("Delta: got %v, want 0.0", s.Delta)
	}
	if s.Regime != ScoreRegimeLaminar {
		t.Errorf("Regime: got %v, want %v", s.Regime, ScoreRegimeLaminar)
	}
	if s.ConfirmInfo != 1.0 {
		t.Errorf("ConfirmInfo: got %v, want 1.0", s.ConfirmInfo)
	}
	if s.DiscrepancyPct != 0.0 {
		t.Errorf("DiscrepancyPct: got %v, want 0.0", s.DiscrepancyPct)
	}
	if s.Kind != KindCategorical {
		t.Errorf("Kind: got %v, want KindCategorical", s.Kind)
	}
}

func TestScorePrediction_TypeMismatch_Scalar(t *testing.T) {
	_, err := ScorePrediction(KindScalar, testCatA, 41.7)
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("got error %v, want ErrTypeMismatch", err)
	}
}

func TestScorePrediction_ZeroPrediction(t *testing.T) {
	_, err := ScorePrediction(KindScalar, 0.0, 1.0)
	if !errors.Is(err, ErrZeroPrediction) {
		t.Errorf("got error %v, want ErrZeroPrediction", err)
	}
}

func TestScoreRegime_String_Roundtrip(t *testing.T) {
	cases := []struct {
		want   string
		regime ScoreRegime
	}{
		{scoreRegimeStrLaminar, ScoreRegimeLaminar},
		{scoreRegimeStrLowSediment, ScoreRegimeLowSediment},
		{scoreRegimeStrModerate, ScoreRegimeModerate},
		{scoreRegimeStrHeavy, ScoreRegimeHeavy},
	}
	for _, tc := range cases {
		if got := tc.regime.String(); got != tc.want {
			t.Errorf("ScoreRegime(%d).String() = %q, want %q", tc.regime, got, tc.want)
		}
	}
}

// TestScorePrediction_RegimeBoundaries sweeps through delta values and
// verifies the regime classifier applies the correct thresholds:
//
//	delta < 0.01  → ScoreRegimeLaminar
//	0.01 ≤ delta < 0.10 → ScoreRegimeLowSediment
//	0.10 ≤ delta < 0.50 → ScoreRegimeModerate
//	delta ≥ 0.50  → ScoreRegimeHeavy
//
// We drive via ScorePrediction(KindScalar, 1.0, 1.0-delta) so that
// the computed delta equals the target delta exactly.
func TestScorePrediction_RegimeBoundaries(t *testing.T) {
	cases := []struct {
		delta      float64
		wantRegime ScoreRegime
	}{
		{0.0, ScoreRegimeLaminar},
		{0.005, ScoreRegimeLaminar},
		{0.05, ScoreRegimeLowSediment},
		{0.25, ScoreRegimeModerate},
		{0.5, ScoreRegimeHeavy},
		{0.9, ScoreRegimeHeavy},
	}
	for _, tc := range cases {
		// predicted=1.0, observed=1.0-delta → |pred-obs|/|pred| == delta.
		s, err := ScorePrediction(KindScalar, 1.0, 1.0-tc.delta)
		if err != nil {
			t.Fatalf("delta=%v: unexpected error: %v", tc.delta, err)
		}
		if math.Abs(s.Delta-tc.delta) > 1e-12 {
			t.Errorf("delta=%v: Score.Delta=%v, want %v", tc.delta, s.Delta, tc.delta)
		}
		if s.Regime != tc.wantRegime {
			t.Errorf("delta=%v: Regime=%v, want %v", tc.delta, s.Regime, tc.wantRegime)
		}
	}
}
