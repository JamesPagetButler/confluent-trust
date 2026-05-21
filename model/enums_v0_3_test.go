package model

import (
	"encoding/json"
	"testing"
)

// TestStatusExtension_V03_RoundTrip verifies marshal/unmarshal round-trip for
// the four new Status values added in v0.3 (design §3).
func TestStatusExtension_V03_RoundTrip(t *testing.T) {
	cases := []struct {
		wantJSON string
		value    Status
	}{
		{`"killed"`, StatusKilled},
		{`"marginal"`, StatusMarginal},
		{`"converged"`, StatusConverged},
		{`"falsified"`, StatusFalsified},
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
			var got Status
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got != c.value {
				t.Errorf("round-trip drift: %v -> %s -> %v", c.value, b, got)
			}
		})
	}
}

// TestStatusExtension_V03_UnknownRejectsValue ensures unrecognised Status
// strings produce a descriptive error.
func TestStatusExtension_V03_UnknownRejectsValue(t *testing.T) {
	var s Status
	if err := json.Unmarshal([]byte(`"nonexistent-status"`), &s); err == nil {
		t.Error("expected error for unknown status value, got nil")
	}
}

// TestProvenanceKind_RoundTrip verifies marshal/unmarshal round-trip for all
// canonical ProvenanceKind values (design §2).
func TestProvenanceKind_RoundTrip(t *testing.T) {
	cases := []struct {
		wantJSON string
		value    ProvenanceKind
	}{
		{`"proof"`, ProvenanceKindProof},
		{`"theory"`, ProvenanceKindTheory},
		{`"theory-external"`, ProvenanceKindTheoryExternal},
		{`"experiment"`, ProvenanceKindExperiment},
		{`"hypothesis"`, ProvenanceKindHypothesis},
		{`"internal-compute"`, ProvenanceKindInternalCompute},
		{`"philosophy"`, ProvenanceKindPhilosophy},
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
			var got ProvenanceKind
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got != c.value {
				t.Errorf("round-trip drift: %v -> %s -> %v", c.value, b, got)
			}
		})
	}
}

// TestProvenanceKind_NullRoundTrip verifies that ProvenanceKindUnknown
// marshals to null and null unmarshals back to ProvenanceKindUnknown.
func TestProvenanceKind_NullRoundTrip(t *testing.T) {
	b, err := json.Marshal(ProvenanceKindUnknown)
	if err != nil {
		t.Fatalf("marshal unknown: %v", err)
	}
	if string(b) != jsonNull {
		t.Errorf("unknown provenance_kind: got %s, want null", b)
	}
	var got ProvenanceKind
	if err := json.Unmarshal([]byte(jsonNull), &got); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if got != ProvenanceKindUnknown {
		t.Errorf("null unmarshal: got %v, want ProvenanceKindUnknown", got)
	}
}

// TestProvenanceKind_UnknownRejectsValue ensures unrecognised provenance_kind
// strings produce a descriptive error.
func TestProvenanceKind_UnknownRejectsValue(t *testing.T) {
	var p ProvenanceKind
	if err := json.Unmarshal([]byte(`"not-a-real-kind"`), &p); err == nil {
		t.Error("expected error for unknown provenance_kind, got nil")
	}
}

// TestProofState_RoundTrip verifies marshal/unmarshal round-trip for all
// canonical ProofState values (design §4.1).
func TestProofState_RoundTrip(t *testing.T) {
	cases := []struct {
		wantJSON string
		value    ProofState
	}{
		{`"verified"`, ProofStateVerified},
		{`"partial"`, ProofStatePartial},
		{`"written"`, ProofStateWritten},
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
			var got ProofState
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got != c.value {
				t.Errorf("round-trip drift: %v -> %s -> %v", c.value, b, got)
			}
		})
	}
}

// TestProofState_NullRoundTrip verifies that ProofStateUnknown marshals to
// null (absent proof state) and null unmarshals back to ProofStateUnknown.
func TestProofState_NullRoundTrip(t *testing.T) {
	b, err := json.Marshal(ProofStateUnknown)
	if err != nil {
		t.Fatalf("marshal unknown: %v", err)
	}
	if string(b) != jsonNull {
		t.Errorf("unknown proof_state: got %s, want null", b)
	}
	var got ProofState
	if err := json.Unmarshal([]byte(jsonNull), &got); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if got != ProofStateUnknown {
		t.Errorf("null unmarshal: got %v, want ProofStateUnknown", got)
	}
}

// TestProofState_UnknownRejectsValue ensures unrecognised proof_state strings
// produce a descriptive error.
func TestProofState_UnknownRejectsValue(t *testing.T) {
	var s ProofState
	if err := json.Unmarshal([]byte(`"not-a-proof-state"`), &s); err == nil {
		t.Error("expected error for unknown proof_state, got nil")
	}
}

// TestTheoremStatus_RoundTrip verifies marshal/unmarshal round-trip for all
// canonical TheoremStatus values (design §4).
func TestTheoremStatus_RoundTrip(t *testing.T) {
	cases := []struct {
		wantJSON string
		value    TheoremStatus
	}{
		{`"verified"`, TheoremStatusVerified},
		{`"written"`, TheoremStatusWritten},
		{`"not_started"`, TheoremStatusNotStarted},
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
			var got TheoremStatus
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal %s: %v", b, err)
			}
			if got != c.value {
				t.Errorf("round-trip drift: %v -> %s -> %v", c.value, b, got)
			}
		})
	}
}

// TestTheoremStatus_NullRoundTrip verifies that TheoremStatusUnknown marshals
// to null and null unmarshals back to TheoremStatusUnknown.
func TestTheoremStatus_NullRoundTrip(t *testing.T) {
	b, err := json.Marshal(TheoremStatusUnknown)
	if err != nil {
		t.Fatalf("marshal unknown: %v", err)
	}
	if string(b) != jsonNull {
		t.Errorf("unknown theorem_status: got %s, want null", b)
	}
	var got TheoremStatus
	if err := json.Unmarshal([]byte(jsonNull), &got); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if got != TheoremStatusUnknown {
		t.Errorf("null unmarshal: got %v, want TheoremStatusUnknown", got)
	}
}

// TestTheoremStatus_UnknownRejectsValue ensures unrecognised theorem_status
// strings produce a descriptive error.
func TestTheoremStatus_UnknownRejectsValue(t *testing.T) {
	var s TheoremStatus
	if err := json.Unmarshal([]byte(`"not-a-theorem-status"`), &s); err == nil {
		t.Error("expected error for unknown theorem_status, got nil")
	}
}
