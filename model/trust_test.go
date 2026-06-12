package model

import (
	"encoding/json"
	"testing"
)

// ptr64 returns a pointer to a float64 literal; used throughout trust tests.
func ptr64(v float64) *float64 { return &v }

// ---- AxisTrust.Validate ----

func TestAxisTrust_Validate_AllNil_Ok(t *testing.T) {
	at := AxisTrust{}
	if err := at.Validate(); err != nil {
		t.Errorf("expected valid for all-nil AxisTrust, got: %v", err)
	}
}

func TestAxisTrust_Validate_AllInRange_Ok(t *testing.T) {
	at := AxisTrust{
		Reproducibility: ptr64(0.8),
		Theory:          ptr64(1.0),
		Stats:           ptr64(0.5),
		Method:          ptr64(0.0),
		Independence:    ptr64(0.75),
	}
	if err := at.Validate(); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestAxisTrust_Validate_ReproducibilityNegative(t *testing.T) {
	at := AxisTrust{Reproducibility: ptr64(-0.1)}
	if err := at.Validate(); err == nil {
		t.Error("expected error for reproducibility < 0, got nil")
	}
}

func TestAxisTrust_Validate_TheoryAboveOne(t *testing.T) {
	at := AxisTrust{Theory: ptr64(1.001)}
	if err := at.Validate(); err == nil {
		t.Error("expected error for theory > 1.0, got nil")
	}
}

func TestAxisTrust_Validate_IndependenceOutOfRange(t *testing.T) {
	at := AxisTrust{Independence: ptr64(2.0)}
	if err := at.Validate(); err == nil {
		t.Error("expected error for independence > 1.0, got nil")
	}
}

// ---- Anchor carries AxisTrust + LocaleDomain + ClusterState ----

func TestAnchor_WithAxisTrust_Valid(t *testing.T) {
	a := baseAnchor("ANCHOR-trust-valid")
	a.AxisTrust = &AxisTrust{
		Reproducibility: ptr64(0.8),
		Independence:    ptr64(0.9),
	}
	a.LocaleDomain = "reproducibility"
	a.ClusterState = ClusterStateDeveloping
	if err := a.Validate(); err != nil {
		t.Errorf("expected valid anchor with axis_trust, got: %v", err)
	}
}

func TestAnchor_WithAxisTrust_InvalidRange_Rejected(t *testing.T) {
	a := baseAnchor("ANCHOR-trust-bad")
	a.AxisTrust = &AxisTrust{Stats: ptr64(-0.5)}
	if err := a.Validate(); err == nil {
		t.Error("expected error for out-of-range stats axis, got nil")
	}
}

func TestAnchor_NoAxisTrust_StillValid(t *testing.T) {
	// Existing anchors without AxisTrust must continue to pass Validate.
	a := baseAnchor("ANCHOR-no-trust")
	if err := a.Validate(); err != nil {
		t.Errorf("anchor without axis_trust: expected valid, got: %v", err)
	}
}

// ---- ClusterState enum round-trips ----

func TestClusterState_RoundTrip(t *testing.T) {
	cases := []struct {
		wantJSON string
		value    ClusterState
	}{
		{`"nascent"`, ClusterStateNascent},
		{`"developing"`, ClusterStateDeveloping},
		{`"confluent"`, ClusterStateConfluent},
	}
	for _, c := range cases {
		t.Run(c.wantJSON, func(t *testing.T) {
			b, err := json.Marshal(c.value)
			if err != nil {
				t.Fatalf("marshal %v: %v", c.value, err)
			}
			if string(b) != c.wantJSON {
				t.Errorf("marshal: got %s, want %s", b, c.wantJSON)
			}
			var got ClusterState
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got != c.value {
				t.Errorf("round-trip drift: %v -> %s -> %v", c.value, b, got)
			}
		})
	}
}

func TestClusterState_NullRoundTrip(t *testing.T) {
	b, err := json.Marshal(ClusterStateUnknown)
	if err != nil {
		t.Fatalf("marshal unknown: %v", err)
	}
	if string(b) != jsonNull {
		t.Errorf("unknown cluster_state: got %s, want null", b)
	}
	var got ClusterState
	if err := json.Unmarshal([]byte(jsonNull), &got); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if got != ClusterStateUnknown {
		t.Errorf("null unmarshal: got %v, want ClusterStateUnknown", got)
	}
}

func TestClusterState_UnknownRejectsValue(t *testing.T) {
	var c ClusterState
	if err := json.Unmarshal([]byte(`"not-a-cluster-state"`), &c); err == nil {
		t.Error("expected error for unknown cluster_state, got nil")
	}
}

// ---- AxisTrust JSON round-trip ----

func TestAxisTrust_JSONRoundTrip_Partial(t *testing.T) {
	orig := AxisTrust{
		Reproducibility: ptr64(0.6),
		Theory:          ptr64(0.9),
		// Stats, Method, Independence intentionally absent
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AxisTrust
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Reproducibility == nil || *got.Reproducibility != 0.6 {
		t.Errorf("reproducibility: got %v, want 0.6", got.Reproducibility)
	}
	if got.Theory == nil || *got.Theory != 0.9 {
		t.Errorf("theory: got %v, want 0.9", got.Theory)
	}
	if got.Stats != nil {
		t.Errorf("stats: expected nil (absent), got %v", *got.Stats)
	}
	if got.Independence != nil {
		t.Errorf("independence: expected nil (absent), got %v", *got.Independence)
	}
}
