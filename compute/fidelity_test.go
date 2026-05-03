package compute

import (
	"math"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func TestStepFidelity_Table(t *testing.T) {
	cases := map[string]float64{
		"lean4_proof":         1.000,
		"established_math":    1.000,
		"standard_physics":    0.999,
		"numerical_verified":  0.999,
		"domain_boundary":     0.95,
		"semi_empirical":      0.95,
		"unproven_conjecture": 0.50,
	}
	for label, want := range cases {
		t.Run(label, func(t *testing.T) {
			if got := StepFidelity(label); got != want {
				t.Errorf("StepFidelity(%q) = %v, want %v", label, got, want)
			}
		})
	}
}

func TestStepFidelity_CaseAndWhitespaceInsensitive(t *testing.T) {
	if got := StepFidelity("  Lean4_Proof  "); got != 1.0 {
		t.Errorf("normalized lookup failed: %v", got)
	}
}

func TestStepFidelity_UnknownReturns1(t *testing.T) {
	if got := StepFidelity("not_a_step_type"); got != 1.0 {
		t.Errorf("unknown step: got %v, want 1.0 (caller decides whether to flag)", got)
	}
}

func TestChainFidelity_ExplicitOverride(t *testing.T) {
	f := 0.85
	c := model.Chain{
		ID: "C-1", Fidelity: &f,
		// StepTypes intentionally set; Fidelity must win.
		StepTypes: []string{"unproven_conjecture", "unproven_conjecture"},
	}
	if got := ChainFidelity(c); got != 0.85 {
		t.Errorf("got %v, want 0.85 (explicit Fidelity wins)", got)
	}
}

func TestChainFidelity_FromStepTypes(t *testing.T) {
	c := model.Chain{
		ID:        "C-2",
		StepTypes: []string{"standard_physics", "domain_boundary", "established_math"},
	}
	want := 0.999 * 0.95 * 1.0
	if got := ChainFidelity(c); math.Abs(got-want) > 1e-9 {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestChainFidelity_EmptyChainReturns1(t *testing.T) {
	if got := ChainFidelity(model.Chain{ID: "C-empty"}); got != 1.0 {
		t.Errorf("got %v, want 1.0", got)
	}
}

func TestChainFidelity_ClampsExplicitOutOfRange(t *testing.T) {
	hi, lo := 1.5, -0.2
	if got := ChainFidelity(model.Chain{ID: "C-hi", Fidelity: &hi}); got != 1.0 {
		t.Errorf("over-1 not clamped to 1: %v", got)
	}
	if got := ChainFidelity(model.Chain{ID: "C-lo", Fidelity: &lo}); got != 0.0 {
		t.Errorf("sub-0 not clamped to 0: %v", got)
	}
}

func TestClassifyFidelityRegime(t *testing.T) {
	tests := []struct {
		want string
		mu   float64
	}{
		{"laminar", 1.000},
		{"laminar", 0.999},
		{"low_sediment", 0.998},
		{"low_sediment", 0.95},
		{"low_sediment", 0.90},
		{"moderate", 0.899},
		{"moderate", 0.70},
		{"heavy", 0.699},
		{"heavy", 0.50},
		{"heavy", math.NaN()},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := ClassifyFidelityRegime(tt.mu); got != tt.want {
				t.Errorf("μ=%v: got %q, want %q", tt.mu, got, tt.want)
			}
		})
	}
}
