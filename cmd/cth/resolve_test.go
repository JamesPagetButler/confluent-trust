package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
)

// qbpFixture is the path to the primary E2E test fixture (59 anchors,
// PROOF-* / MEAS-* / PRED-* / DERIV-* / INPUT-* prefixes).
const qbpFixture = "../../testdata/qbp_v3_2.json"

// resolveAnchorID is a known PROOF-* anchor ID in qbp_v3_2.json.
const resolveAnchorID = "PROOF-hurwitz"

// resolveAxiomID is a known AXIOM-* ID in qbp_v3_2.json.
const resolveAxiomID = "AXIOM-1"

// resolveDerivedID is a known DERIV-* ID in qbp_v3_2.json.
const resolveDerivedID = "DERIV-holographic"

// loadQBPFixture is a helper that loads the qbp_v3_2 inventory or fails the test.
func loadQBPFixture(t *testing.T) model.Inventory {
	t.Helper()
	inv, err := store.LoadInventory(qbpFixture)
	if err != nil {
		t.Fatalf("loadQBPFixture: %v", err)
	}
	return inv
}

// TestResolveAnchor_FindsAnchor verifies that a known PROOF-* anchor is found
// with the expected fields.
func TestResolveAnchor_FindsAnchor(t *testing.T) {
	inv := loadQBPFixture(t)

	got, ok := ResolveAnchor(inv, resolveAnchorID)
	if !ok {
		t.Fatalf("ResolveAnchor(%q): expected ok=true, got false", resolveAnchorID)
	}
	if got.ID != resolveAnchorID {
		t.Errorf("ID: got %q, want %q", got.ID, resolveAnchorID)
	}
	// PROOF-hurwitz name is "Hurwitz theorem: only 4 normed division algebras"
	if !strings.Contains(got.Name, "Hurwitz") {
		t.Errorf("Name should contain \"Hurwitz\"; got %q", got.Name)
	}
	// Tier 1 (TierProof) per fixture
	if got.Tier != model.TierProof {
		t.Errorf("Tier: got %d, want %d (TierProof)", got.Tier, model.TierProof)
	}
}

// TestResolveAnchor_FindsAxiom verifies that a known AXIOM-* ID is resolved to
// a synthesized Anchor with Tier == TierAxiom (0).
func TestResolveAnchor_FindsAxiom(t *testing.T) {
	inv := loadQBPFixture(t)

	got, ok := ResolveAnchor(inv, resolveAxiomID)
	if !ok {
		t.Fatalf("ResolveAnchor(%q): expected ok=true, got false", resolveAxiomID)
	}
	if got.ID != resolveAxiomID {
		t.Errorf("ID: got %q, want %q", got.ID, resolveAxiomID)
	}
	// AXIOM-1 name is "Information is preserved"
	if !strings.Contains(got.Name, "Information") {
		t.Errorf("Name should contain \"Information\"; got %q", got.Name)
	}
	if got.Tier != model.TierAxiom {
		t.Errorf("Tier: got %d, want %d (TierAxiom)", got.Tier, model.TierAxiom)
	}
}

// TestResolveAnchor_FindsDerivedPrinciple verifies that a known DERIV-* ID is
// resolved to a synthesized Anchor with Tier == TierProof (1) and
// PredictionChain populated from DerivedFrom.
func TestResolveAnchor_FindsDerivedPrinciple(t *testing.T) {
	inv := loadQBPFixture(t)

	got, ok := ResolveAnchor(inv, resolveDerivedID)
	if !ok {
		t.Fatalf("ResolveAnchor(%q): expected ok=true, got false", resolveDerivedID)
	}
	if got.ID != resolveDerivedID {
		t.Errorf("ID: got %q, want %q", got.ID, resolveDerivedID)
	}
	// DERIV-holographic name is "Holographic principle"
	if !strings.Contains(got.Name, "Holographic") {
		t.Errorf("Name should contain \"Holographic\"; got %q", got.Name)
	}
	if got.Tier != model.TierProof {
		t.Errorf("Tier: got %d, want %d (TierProof)", got.Tier, model.TierProof)
	}
	// DERIV-holographic has DerivedFrom: ["AXIOM-1", "AXIOM-2"]
	if len(got.PredictionChain) == 0 {
		t.Error("PredictionChain should be populated from DerivedFrom; got empty slice")
	}
}

// TestResolveAnchor_NotFound verifies that a non-existent ID returns ok=false
// and a zero-value Anchor.
func TestResolveAnchor_NotFound(t *testing.T) {
	inv := loadQBPFixture(t)

	got, ok := ResolveAnchor(inv, "PROOF-does-not-exist")
	if ok {
		t.Fatal("expected ok=false for unknown ID, got true")
	}
	if got.ID != "" {
		t.Errorf("expected zero-value Anchor on not-found; got ID=%q", got.ID)
	}
}

// TestRunResolve_HappyPath verifies that runResolve writes valid JSON for a
// known anchor and that the output contains the anchor ID.
func TestRunResolve_HappyPath(t *testing.T) {
	var err error
	out := captureStdout(func() {
		err = runResolve([]string{qbpFixture, resolveAnchorID})
	})
	if err != nil {
		t.Fatalf("runResolve returned unexpected error: %v", err)
	}
	if !strings.Contains(out, resolveAnchorID) {
		t.Errorf("output should contain anchor ID %q; got:\n%s", resolveAnchorID, out)
	}
	// Output must be valid JSON
	var a model.Anchor
	if err := json.Unmarshal([]byte(out), &a); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput:\n%s", err, out)
	}
}

// TestRunResolve_NotFound verifies that runResolve returns an error citing
// "not found" and the missing ID when the anchor does not exist.
func TestRunResolve_NotFound(t *testing.T) {
	const missingID = "PROOF-does-not-exist"
	err := runResolve([]string{qbpFixture, missingID})
	if err == nil {
		t.Fatal("expected error for missing anchor, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention \"not found\"; got: %v", err)
	}
	if !strings.Contains(err.Error(), missingID) {
		t.Errorf("error should cite the missing ID %q; got: %v", missingID, err)
	}
}

// TestRunResolve_MissingArgs verifies that runResolve returns an error when
// called with zero or one positional argument.
func TestRunResolve_MissingArgs(t *testing.T) {
	cases := [][]string{
		{},
		{qbpFixture},
	}
	for _, args := range cases {
		err := runResolve(args)
		if err == nil {
			t.Errorf("runResolve(%v): expected error, got nil", args)
		}
	}
}

// TestRunResolve_OutputFlag verifies that -o writes valid JSON to the specified
// path and that the file can be parsed back as a model.Anchor.
func TestRunResolve_OutputFlag(t *testing.T) {
	outPath := t.TempDir() + "/resolved-anchor.json"
	err := runResolve([]string{qbpFixture, resolveAnchorID, "-o", outPath})
	if err != nil {
		t.Fatalf("runResolve returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath) //nolint:gosec // G304: test reads from t.TempDir(), not user-controlled input
	if err != nil {
		t.Fatalf("output file not found or unreadable: %v", err)
	}
	if len(data) == 0 {
		t.Error("output file is empty")
	}
	// Must parse back as model.Anchor
	var a model.Anchor
	if err := json.Unmarshal(data, &a); err != nil {
		t.Errorf("output file is not valid JSON Anchor: %v\ncontent:\n%s", err, data)
	}
	if a.ID != resolveAnchorID {
		t.Errorf("parsed anchor ID: got %q, want %q", a.ID, resolveAnchorID)
	}
}
