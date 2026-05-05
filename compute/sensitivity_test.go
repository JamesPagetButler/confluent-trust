// Issue #8 acceptance: bracket reported for QBP v3.2; sensitivity ratio
// computed. QBP v3.2 fixture is intentionally absent from testdata/;
// structural acceptance is encoded on synthetic + qbp_quantum_v0_2.
package compute

import (
	"math"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func TestSensitivityBracket_OrderingHalfBaseDouble(t *testing.T) {
	// Halving axiom entropy lowers the ρ_net denominator → higher ρ_net.
	// Doubling raises the denominator → lower ρ_net. So
	// halfH > baseH > doubleH for any inventory with positive
	// confirmed information.
	delta := 0.0
	inv := model.Inventory{
		Axioms: []model.Axiom{{ID: testAxiomID}},
		Anchors: []model.Anchor{
			{
				ID: testAnchorM1, Tier: model.TierMeasurement,
				Status: model.StatusCoherent, DiscrepancyPct: &delta,
			},
		},
	}

	halfH, baseH, doubleH := SensitivityBracket(inv, nil)
	if !(halfH > baseH && baseH > doubleH) {
		t.Errorf("expected halfH > baseH > doubleH, got %v / %v / %v",
			halfH, baseH, doubleH)
	}
	for _, v := range []float64{halfH, baseH, doubleH} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			t.Errorf("non-finite bracket value: %v", v)
		}
	}
}

func TestSensitivityBracket_RespectsAxiomEntropyTable(t *testing.T) {
	// Two axioms; assigned table sets one to 2.0 (base assumption: 1.0).
	// At base, H_axioms = 2 + 1 = 3. At half, H_axioms = 1 + 0.5 = 1.5.
	// At double, H_axioms = 4 + 2 = 6. Confirm via direct ρ_net at base.
	delta := 0.0
	inv := model.Inventory{
		Axioms: []model.Axiom{{ID: testAxiomID}, {ID: "AX-2"}},
		Anchors: []model.Anchor{
			{
				ID: testAnchorM1, Tier: model.TierMeasurement,
				Status: model.StatusCoherent, DiscrepancyPct: &delta,
			},
		},
	}
	tab := map[string]float64{testAxiomID: 2.0}

	_, baseH, _ := SensitivityBracket(inv, tab)
	wantBase := 1.0 / (2.0 + 1.0) // I_confirmed = 1; H_axioms = 2 + default(1)
	if math.Abs(baseH-wantBase) > 1e-9 {
		t.Errorf("baseH = %v, want %v", baseH, wantBase)
	}
}

func TestSensitivityRatio_RobustWhenAboveHalf(t *testing.T) {
	if got := SensitivityRatio(1.0, 0.6); got <= 0.5 {
		t.Errorf("0.6/1.0 should be > 0.5, got %v", got)
	}
	if got := SensitivityRatio(1.0, 0.4); got >= 0.5 {
		t.Errorf("0.4/1.0 should be < 0.5, got %v", got)
	}
}

func TestSensitivityRatio_DegenerateReturnsZero(t *testing.T) {
	if got := SensitivityRatio(0, 0.5); got != 0 {
		t.Errorf("halfH=0 should return 0 (undefined), got %v", got)
	}
}

func TestSensitivityBracket_QBPQuantumV02(t *testing.T) {
	// Acceptance: bracket reported for QBP v3.2 (absent fixture; we use
	// the available v0.2 fixture). Triple must be finite and ordered.
	inv := loadFixture(t, "qbp_quantum_v0_2.json")
	half, base, double := SensitivityBracket(inv, nil)
	for _, v := range []float64{half, base, double} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			t.Errorf("non-finite value in qbp_quantum_v0_2 bracket: %v", v)
		}
	}
	if base > 0 && !(half >= base && base >= double) {
		t.Errorf("monotonicity broken: half=%v base=%v double=%v",
			half, base, double)
	}
}
