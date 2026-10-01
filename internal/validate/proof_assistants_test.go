package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// confluent-trust#108 AC1 — the additive proof_assistants schema field.
// AC2–AC6 (the derived PA engine) are tested in internal/pa; these cover the
// schema layer only: additivity + the closed assistant enum + required trio.

// paAnchorMarker is a unique line in testdata/minimal.json (the PROOF-newton-2nd
// anchor's description) used to inject a proof_assistants list for these tests.
const paAnchorMarker = `"description": "Stated for the unit-test baseline; not a real proof.",`

func minimalWithPA(t *testing.T, paJSON string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "minimal.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, paAnchorMarker) {
		t.Fatal("minimal.json injection marker not found; fixture changed")
	}
	return []byte(strings.Replace(s, paAnchorMarker, paAnchorMarker+"\n      "+paJSON, 1))
}

// TestSchema_AdditiveOldRecordsValidate: a record with NO proof_assistants (every
// pre-0.3.5 anchor) still validates. Mutation that must fail: making
// proof_assistants required, or otherwise breaking old records.
func TestSchema_AdditiveOldRecordsValidate(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "minimal.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Inventory(b); err != nil {
		t.Fatalf("pre-0.3.5 record (no proof_assistants) must still validate under 0.3.5: %v", err)
	}
}

// TestSchema_AcceptsValidProofAssistant: a well-formed entry validates.
func TestSchema_AcceptsValidProofAssistant(t *testing.T) {
	data := minimalWithPA(t, `"proof_assistants": [{"assistant": "lean4", "evidence_ref": "proofs/x.lean@abc1234#t", "trust_check": "pass"}],`)
	if err := Inventory(data); err != nil {
		t.Fatalf("a valid proof_assistants entry must validate: %v", err)
	}
}

// TestSchema_RejectsUnknownAssistant: the assistant enum is closed
// {lean4,coq,agda}. Mutations that must fail: widen the enum to any string; or
// remove proof_assistants from the schema (without the $ref the enum is not
// enforced and this record would wrongly validate).
func TestSchema_RejectsUnknownAssistant(t *testing.T) {
	data := minimalWithPA(t, `"proof_assistants": [{"assistant": "isabelle", "evidence_ref": "x@sha#t", "trust_check": "pass"}],`)
	if err := Inventory(data); err == nil {
		t.Fatal("unknown assistant (isabelle) must be rejected — closed enum {lean4,coq,agda}")
	}
}

// TestSchema_RejectsProofAssistantMissingRequired: assistant, evidence_ref and
// trust_check are each required on an entry.
func TestSchema_RejectsProofAssistantMissingRequired(t *testing.T) {
	data := minimalWithPA(t, `"proof_assistants": [{"assistant": "lean4"}],`)
	if err := Inventory(data); err == nil {
		t.Fatal("a proof_assistants entry missing evidence_ref/trust_check must be rejected")
	}
}

// TestSchema_RejectsBadTrustCheck: trust_check is the closed enum {pass,fail}.
func TestSchema_RejectsBadTrustCheck(t *testing.T) {
	data := minimalWithPA(t, `"proof_assistants": [{"assistant": "coq", "evidence_ref": "x@sha#t", "trust_check": "maybe"}],`)
	if err := Inventory(data); err == nil {
		t.Fatal("trust_check outside {pass,fail} must be rejected")
	}
}
