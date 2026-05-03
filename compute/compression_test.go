package compute

import (
	"math"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func loadInv(t *testing.T, path string) model.Inventory {
	t.Helper()
	// Inline the fixture by parsing raw JSON via the model package — the
	// store package is already covered by its own tests. We deliberately
	// avoid a dependency on store/ to keep compute/ stdlib-only.
	return model.Inventory{} // populated by callers below via literal construction
}

func TestInformationDeficit_Empty(t *testing.T) {
	inv := model.Inventory{}
	if got := InformationDeficit(inv); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestInformationDeficit_FromInputs(t *testing.T) {
	inv := model.Inventory{
		Inputs: []model.Input{
			{ID: "INST-A", Type: "input", Status: "measurable", SignificantFigures: 3},
			{ID: "INST-B", Type: "input", Status: "measurable", SignificantFigures: 2},
		},
	}
	want := InputEntropy(3) + InputEntropy(2)
	if got := InformationDeficit(inv); math.Abs(got-want) > 1e-9 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAxiomEntropySum(t *testing.T) {
	inv := model.Inventory{
		Axioms: []model.Axiom{
			{ID: "AX-1"}, {ID: "AX-2"}, {ID: "AX-3"},
		},
	}
	if got := AxiomEntropySum(inv, nil); got != 3.0*DefaultAxiomEntropyBits {
		t.Errorf("got %v, want %v", got, 3.0*DefaultAxiomEntropyBits)
	}

	tab := map[string]float64{"AX-1": 0.5, "AX-2": 1.5}
	if got := AxiomEntropySum(inv, tab); got != 0.5+1.5+DefaultAxiomEntropyBits {
		t.Errorf("got %v, want %v", got, 0.5+1.5+DefaultAxiomEntropyBits)
	}
}

func TestGrossCompression_DenomZeroReturnsZero(t *testing.T) {
	if got := GrossCompression(model.Inventory{}, nil); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestGrossCompression_StructuralAnchorsContribute(t *testing.T) {
	delta := 0.0
	inv := model.Inventory{
		Axioms: []model.Axiom{{ID: "AX-1"}},
		Anchors: []model.Anchor{
			{ID: "M-1", Tier: model.TierMeasurement, Status: model.StatusCoherent, DiscrepancyPct: &delta},
			{ID: "M-2", Tier: model.TierMeasurement, Status: model.StatusCoherent, DiscrepancyPct: &delta},
		},
	}
	// Each structural match contributes 1 bit; 2 anchors → I_confirmed = 2.
	// H_axioms = 1 (default), Δ = 0 → ρ_gross = 2.0.
	if got := GrossCompression(inv, nil); math.Abs(got-2.0) > 1e-9 {
		t.Errorf("got %v, want 2.0", got)
	}
}

func TestNetCompression_FractionalInputCostAllocation(t *testing.T) {
	delta := 0.0
	inv := model.Inventory{
		Axioms: []model.Axiom{{ID: "AX-1"}},
		Inputs: []model.Input{
			{ID: "INST-shared", Type: "input", Status: "measurable", SignificantFigures: 1},
			{ID: "INST-solo", Type: "input", Status: "measurable", SignificantFigures: 1},
		},
		Anchors: []model.Anchor{
			{ID: "M-1", Tier: model.TierMeasurement, Status: model.StatusCoherent, DiscrepancyPct: &delta,
				PredictionChain: []string{"INST-shared", "INST-solo"}},
			{ID: "M-2", Tier: model.TierMeasurement, Status: model.StatusCoherent, DiscrepancyPct: &delta,
				PredictionChain: []string{"INST-shared"}},
		},
	}

	netRho, detail := NetCompression(inv, nil)

	// 2 confirmed anchors × 1 bit each = 2 bits I_confirmed.
	if math.Abs(detail.IConfirmed-2.0) > 1e-9 {
		t.Errorf("IConfirmed: got %v, want 2.0", detail.IConfirmed)
	}

	// INST-shared: consumed by 2 anchors; per-anchor cost = 1*sf/2 = inputBits/2.
	// INST-solo: consumed by 1 anchor; per-anchor cost = inputBits.
	// Total input cost = inputBits (shared) + inputBits (solo) = 2 * inputBits.
	wantCost := 2.0 * inputBitsPerSigFig
	if math.Abs(detail.IInputCost-wantCost) > 1e-9 {
		t.Errorf("IInputCost: got %v, want %v", detail.IInputCost, wantCost)
	}

	// H_axioms = 1, Δ = 2 * inputBits, denom = 1 + 2*inputBits.
	wantDenom := DefaultAxiomEntropyBits + 2*inputBitsPerSigFig
	if math.Abs(detail.HAxioms+detail.InfoDeficit-wantDenom) > 1e-9 {
		t.Errorf("denom: got %v, want %v", detail.HAxioms+detail.InfoDeficit, wantDenom)
	}

	wantNet := (2.0 - wantCost) / wantDenom
	if math.Abs(netRho-wantNet) > 1e-9 {
		t.Errorf("net ρ: got %v, want %v", netRho, wantNet)
	}

	// Per-anchor breakdown: M-1 paid full INST-solo + half INST-shared.
	// M-2 paid only half INST-shared.
	if len(detail.PerAnchor) != 2 {
		t.Fatalf("PerAnchor len: got %d, want 2", len(detail.PerAnchor))
	}
	for _, row := range detail.PerAnchor {
		switch row.AnchorID {
		case "M-1":
			want := inputBitsPerSigFig + inputBitsPerSigFig/2
			if math.Abs(row.InputCost-want) > 1e-9 {
				t.Errorf("M-1 cost: got %v, want %v", row.InputCost, want)
			}
			if len(row.InputsUsed) != 2 {
				t.Errorf("M-1 InputsUsed len: got %d, want 2", len(row.InputsUsed))
			}
		case "M-2":
			want := inputBitsPerSigFig / 2
			if math.Abs(row.InputCost-want) > 1e-9 {
				t.Errorf("M-2 cost: got %v, want %v", row.InputCost, want)
			}
		}
	}
}

func TestNetCompression_NetEqualsGrossWithNoInputs(t *testing.T) {
	delta := 0.0
	inv := model.Inventory{
		Axioms: []model.Axiom{{ID: "AX-1"}},
		Anchors: []model.Anchor{
			{ID: "M-1", Tier: model.TierMeasurement, Status: model.StatusCoherent, DiscrepancyPct: &delta},
		},
	}
	netRho, detail := NetCompression(inv, nil)
	if math.Abs(netRho-detail.GrossRho) > 1e-9 {
		t.Errorf("net %v should equal gross %v with no inputs", netRho, detail.GrossRho)
	}
	if detail.IInputCost != 0 {
		t.Errorf("input cost: got %v, want 0", detail.IInputCost)
	}
}

func TestCompressionVelocity(t *testing.T) {
	tests := []struct {
		name string
		prev VersionSnapshot
		curr VersionSnapshot
		want float64
	}{
		{"positive accel", VersionSnapshot{Rho: 0.5, AnchorCount: 10}, VersionSnapshot{Rho: 0.7, AnchorCount: 20}, (0.7 - 0.5) / 10.0},
		{"stalled", VersionSnapshot{Rho: 0.5, AnchorCount: 10}, VersionSnapshot{Rho: 0.5, AnchorCount: 12}, 0.0},
		{"epicycles", VersionSnapshot{Rho: 0.5, AnchorCount: 10}, VersionSnapshot{Rho: 0.4, AnchorCount: 20}, (0.4 - 0.5) / 10.0},
		{"no progress", VersionSnapshot{Rho: 0.5, AnchorCount: 10}, VersionSnapshot{Rho: 0.7, AnchorCount: 10}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompressionVelocity(tt.prev, tt.curr); math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// Silence the unused-helper warning for loadInv (kept as scaffolding for
// future fixture-based tests once known_values.go is wired in).
var _ = loadInv
