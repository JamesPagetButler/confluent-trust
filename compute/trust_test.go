package compute

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// p64 is a float64 pointer helper for test literals.
func p64(v float64) *float64 { return &v }

// ---- GlueAxisTrust ----

func TestGlueAxisTrust_Empty(t *testing.T) {
	out := GlueAxisTrust(nil)
	if out.Reproducibility != nil || out.Theory != nil ||
		out.Stats != nil || out.Method != nil || out.Independence != nil {
		t.Error("empty input: expected all-nil result")
	}
}

func TestGlueAxisTrust_SingleSection(t *testing.T) {
	sections := []model.AxisTrust{{
		Reproducibility: p64(0.7),
		Theory:          p64(0.8),
	}}
	out := GlueAxisTrust(sections)
	if out.Reproducibility == nil || *out.Reproducibility != 0.7 {
		t.Errorf("reproducibility: got %v, want 0.7", out.Reproducibility)
	}
	if out.Theory == nil || *out.Theory != 0.8 {
		t.Errorf("theory: got %v, want 0.8", out.Theory)
	}
	if out.Independence != nil {
		t.Errorf("independence: expected nil, got %v", *out.Independence)
	}
}

// TestGlueAxisTrust_MeetPoisoning checks the reproducibility axis: meet means
// the minimum wins. "One weak joint poisons the claim" — two anchors that agree
// but come from the same lab yield meet = min(0.6, 0.9) = 0.6, not average 0.75.
func TestGlueAxisTrust_MeetPoisoning_Reproducibility(t *testing.T) {
	sections := []model.AxisTrust{
		{Reproducibility: p64(0.9)},
		{Reproducibility: p64(0.6)},
	}
	out := GlueAxisTrust(sections)
	if out.Reproducibility == nil || *out.Reproducibility != 0.6 {
		t.Errorf("meet(0.9, 0.6): got %v, want 0.6", out.Reproducibility)
	}
}

// TestGlueAxisTrust_JoinElevation checks the theory axis: join means the
// maximum wins. Any strong theoretical anchor elevates the cluster.
func TestGlueAxisTrust_JoinElevation_Theory(t *testing.T) {
	sections := []model.AxisTrust{
		{Theory: p64(0.4)},
		{Theory: p64(0.95)},
		{Theory: p64(0.2)},
	}
	out := GlueAxisTrust(sections)
	if out.Theory == nil || *out.Theory != 0.95 {
		t.Errorf("join(0.4, 0.95, 0.2): got %v, want 0.95", out.Theory)
	}
}

func TestGlueAxisTrust_MeetStats(t *testing.T) {
	sections := []model.AxisTrust{
		{Stats: p64(0.8)},
		{Stats: p64(0.5)},
	}
	out := GlueAxisTrust(sections)
	if out.Stats == nil || *out.Stats != 0.5 {
		t.Errorf("meet stats: got %v, want 0.5", out.Stats)
	}
}

func TestGlueAxisTrust_MeetMethod(t *testing.T) {
	sections := []model.AxisTrust{
		{Method: p64(1.0)},
		{Method: p64(0.3)},
	}
	out := GlueAxisTrust(sections)
	if out.Method == nil || *out.Method != 0.3 {
		t.Errorf("meet method: got %v, want 0.3", out.Method)
	}
}

func TestGlueAxisTrust_MeetIndependence_MostConservative(t *testing.T) {
	sections := []model.AxisTrust{
		{Independence: p64(0.9)},
		{Independence: p64(0.1)},
		{Independence: p64(0.8)},
	}
	out := GlueAxisTrust(sections)
	if out.Independence == nil || *out.Independence != 0.1 {
		t.Errorf("meet independence: got %v, want 0.1", out.Independence)
	}
}

// TestGlueAxisTrust_AbsentAxisSkipped verifies that a nil axis in one section
// does not overwrite an existing accumulated value.
func TestGlueAxisTrust_AbsentAxisSkipped(t *testing.T) {
	sections := []model.AxisTrust{
		{Reproducibility: p64(0.8)},
		{Reproducibility: nil}, // absent — should not replace the accumulated 0.8
	}
	out := GlueAxisTrust(sections)
	if out.Reproducibility == nil || *out.Reproducibility != 0.8 {
		t.Errorf("absent axis: got %v, want 0.8", out.Reproducibility)
	}
}

func TestGlueAxisTrust_AllAxes_MixedOps(t *testing.T) {
	// Two full sections; verify each axis glues with its correct operation.
	s1 := model.AxisTrust{
		Reproducibility: p64(0.6),
		Theory:          p64(0.4),
		Stats:           p64(0.9),
		Method:          p64(0.7),
		Independence:    p64(0.8),
	}
	s2 := model.AxisTrust{
		Reproducibility: p64(0.9),
		Theory:          p64(0.95),
		Stats:           p64(0.5),
		Method:          p64(0.3),
		Independence:    p64(0.6),
	}
	out := GlueAxisTrust([]model.AxisTrust{s1, s2})

	check := func(name string, got *float64, want float64) {
		t.Helper()
		if got == nil || *got != want {
			t.Errorf("%s: got %v, want %g", name, got, want)
		}
	}
	check("reproducibility (meet)", out.Reproducibility, 0.6)
	check("theory (join)", out.Theory, 0.95)
	check("stats (meet)", out.Stats, 0.5)
	check("method (meet)", out.Method, 0.3)
	check("independence (meet)", out.Independence, 0.6)
}

// ---- ClusterStateFromAxes ----

func TestClusterStateFromAxes_NilIndependence_Nascent(t *testing.T) {
	at := model.AxisTrust{
		Reproducibility: p64(0.9),
		Theory:          p64(0.9),
		// Independence nil
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateNascent {
		t.Errorf("nil independence: got %v, want nascent", got)
	}
}

func TestClusterStateFromAxes_LowIndependence_Nascent(t *testing.T) {
	at := model.AxisTrust{
		Reproducibility: p64(0.9),
		Theory:          p64(0.9),
		Stats:           p64(0.9),
		Method:          p64(0.9),
		Independence:    p64(0.5), // below 0.7 threshold
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateNascent {
		t.Errorf("low independence: got %v, want nascent", got)
	}
}

func TestClusterStateFromAxes_IndependenceAboveMissingOtherAxes_Developing(t *testing.T) {
	at := model.AxisTrust{
		Independence: p64(0.8),
		// Reproducibility, Theory, Stats, Method all absent
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateDeveloping {
		t.Errorf("independence above threshold, others absent: got %v, want developing", got)
	}
}

func TestClusterStateFromAxes_IndependenceAboveSomeAxisBelow_Developing(t *testing.T) {
	at := model.AxisTrust{
		Reproducibility: p64(0.9),
		Theory:          p64(0.9),
		Stats:           p64(0.6), // below threshold
		Method:          p64(0.9),
		Independence:    p64(0.8),
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateDeveloping {
		t.Errorf("stats below threshold: got %v, want developing", got)
	}
}

func TestClusterStateFromAxes_AllAxesAboveThreshold_Confluent(t *testing.T) {
	at := model.AxisTrust{
		Reproducibility: p64(0.8),
		Theory:          p64(0.9),
		Stats:           p64(0.75),
		Method:          p64(0.85),
		Independence:    p64(0.9),
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateConfluent {
		t.Errorf("all above threshold: got %v, want confluent", got)
	}
}

func TestClusterStateFromAxes_ExactThreshold_Confluent(t *testing.T) {
	// Exactly at 0.7 on all axes should reach CONFLUENT (>= threshold).
	at := model.AxisTrust{
		Reproducibility: p64(0.7),
		Theory:          p64(0.7),
		Stats:           p64(0.7),
		Method:          p64(0.7),
		Independence:    p64(0.7),
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateConfluent {
		t.Errorf("exact threshold: got %v, want confluent", got)
	}
}

func TestClusterStateFromAxes_JustBelowThreshold_Nascent(t *testing.T) {
	// Independence just below threshold → NASCENT regardless of other axes.
	at := model.AxisTrust{
		Reproducibility: p64(1.0),
		Theory:          p64(1.0),
		Stats:           p64(1.0),
		Method:          p64(1.0),
		Independence:    p64(0.699),
	}
	if got := ClusterStateFromAxes(at); got != model.ClusterStateNascent {
		t.Errorf("independence just below threshold: got %v, want nascent", got)
	}
}
