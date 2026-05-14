package compute

import (
	"errors"
	"math"
)

// ScorePrediction is the A18 §2.4 scoring glue: a clean API over CTH's
// existing residual-entropy + confirmatory-info primitives. Used by BMA
// #107 (M2 WDEvent → CTH ρ_net feedback loop) and Wyrd PR #35's
// NT_SIGNAL → CTH `PRED-*` flow (where a Wyrd Prediction's Score field
// is populated from this function when CTHAnchor != nil).

// PredictionKind discriminates the predicted-value shape. v0.1 admits
// scalar and categorical; KindProcess (MI on probability distributions)
// is deferred to v0.2 per addendum-18-walk seq=6 P7.
type PredictionKind uint8

// PredictionKind constants for KindScalar and KindCategorical.
const (
	KindScalar      PredictionKind = iota // predicted/observed both float64
	KindCategorical                       // predicted/observed both string
)

// ScoreRegime classifies the magnitude of a prediction-vs-observation delta.
// Thresholds chosen to map cleanly onto model.Status transitions:
//
//	ScoreRegimeLaminar    (delta <  1%)   → Status: coherent
//	ScoreRegimeLowSediment (1% ≤ delta < 10%)  → Status: coherent (annotated low-sediment)
//	ScoreRegimeModerate   (10% ≤ delta < 50%) → Status: contested
//	ScoreRegimeHeavy      (delta ≥ 50%)       → Status: refuted
type ScoreRegime uint8

// ScoreRegime constants ordered from lowest to highest discrepancy.
const (
	ScoreRegimeLaminar     ScoreRegime = iota // delta < 1%
	ScoreRegimeLowSediment                    // 1% ≤ delta < 10%
	ScoreRegimeModerate                       // 10% ≤ delta < 50%
	ScoreRegimeHeavy                          // delta ≥ 50%
)

// String returns the canonical regime name.
func (r ScoreRegime) String() string {
	switch r {
	case ScoreRegimeLaminar:
		return "laminar"
	case ScoreRegimeLowSediment:
		return "low_sediment"
	case ScoreRegimeModerate:
		return "moderate"
	case ScoreRegimeHeavy:
		return "heavy"
	default:
		return "unknown"
	}
}

// Score is the result of evaluating one prediction against one observation.
// Fields are ordered for optimal struct alignment (float64 first, then uint8).
type Score struct {
	Delta          float64        // Scalar: |predicted - observed| / |predicted|. Categorical: 0.0 on match, 1.0 on miss.
	DiscrepancyPct float64        // 100.0 * Delta. Suitable for direct assignment to model.Anchor.DiscrepancyPct.
	ConfirmInfo    float64        // Confirmatory bits per Theory v0.2 Def 8. Match (delta=0): 1.0. Otherwise: -log2(delta), clamped at 0.
	Kind           PredictionKind // Discriminates scalar vs categorical.
	Regime         ScoreRegime    // Classification of the delta magnitude.
}

// ScorePrediction evaluates a prediction against an observation and
// returns a Score. The `predicted` and `observed` arguments are
// type-asserted based on kind:
//
//	KindScalar      — both must be float64; otherwise ErrTypeMismatch
//	KindCategorical — both must be string;  otherwise ErrTypeMismatch
//
// For KindScalar, the function returns ErrZeroPrediction when predicted == 0
// (cannot compute relative delta against a zero baseline).
func ScorePrediction(kind PredictionKind, predicted, observed any) (Score, error) {
	switch kind {
	case KindScalar:
		pred, ok1 := predicted.(float64)
		obs, ok2 := observed.(float64)
		if !ok1 || !ok2 {
			return Score{}, ErrTypeMismatch
		}
		if pred == 0 {
			return Score{}, ErrZeroPrediction
		}
		delta := math.Abs(pred-obs) / math.Abs(pred)
		discrepancyPct := 100.0 * delta
		confirmInfo := scoreConfirmInfo(delta)
		regime := scoreClassifyRegime(delta)
		return Score{
			Kind:           KindScalar,
			Delta:          delta,
			DiscrepancyPct: discrepancyPct,
			ConfirmInfo:    confirmInfo,
			Regime:         regime,
		}, nil

	case KindCategorical:
		pred, ok1 := predicted.(string)
		obs, ok2 := observed.(string)
		if !ok1 || !ok2 {
			return Score{}, ErrTypeMismatch
		}
		if pred == obs {
			return Score{
				Kind:           KindCategorical,
				Delta:          0.0,
				DiscrepancyPct: 0.0,
				ConfirmInfo:    1.0,
				Regime:         ScoreRegimeLaminar,
			}, nil
		}
		return Score{
			Kind:           KindCategorical,
			Delta:          1.0,
			DiscrepancyPct: 100.0,
			ConfirmInfo:    0.0,
			Regime:         ScoreRegimeHeavy,
		}, nil

	default:
		return Score{}, ErrTypeMismatch
	}
}

// scoreConfirmInfo computes the confirmatory information in bits for a
// given delta per Theory v0.2 Def 8. When delta == 0 it returns 1.0
// (perfect match). Otherwise -log2(delta), clamped to [0, 1024].
func scoreConfirmInfo(delta float64) float64 {
	const ceilingBits = 1024.0
	if delta == 0 {
		return 1.0
	}
	v := -math.Log2(delta)
	if v < 0 {
		return 0.0
	}
	if v > ceilingBits {
		return ceilingBits
	}
	return v
}

// scoreClassifyRegime maps a scalar delta to a ScoreRegime per the
// thresholds in the ScoreRegime documentation.
func scoreClassifyRegime(delta float64) ScoreRegime {
	switch {
	case delta < 0.01:
		return ScoreRegimeLaminar
	case delta < 0.10:
		return ScoreRegimeLowSediment
	case delta < 0.50:
		return ScoreRegimeModerate
	default:
		return ScoreRegimeHeavy
	}
}

// ErrTypeMismatch is returned when predicted/observed type-assertion
// fails for the given Kind.
var ErrTypeMismatch = errors.New("compute/scoring: type mismatch for kind")

// ErrZeroPrediction is returned when a Scalar prediction is zero,
// which makes relative-delta computation ill-defined.
var ErrZeroPrediction = errors.New("compute/scoring: zero prediction (cannot compute relative delta)")
