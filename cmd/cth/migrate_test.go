package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
)

const (
	qbpV32Fixture      = "../../testdata/qbp_v3_2.json"
	lifecycleV2Fixture = "../../testdata/predictions_lifecycle.json"
	lifecycleV3Fixture = "../../testdata/predictions_lifecycle_v0_3.json"
)

// TestMigrate_QBPv3_2_RoundTrip migrates qbp_v3_2.json (59 anchors, all provenance T)
// with no decisions supplied, then validates the resulting inventory.
func TestMigrate_QBPv3_2_RoundTrip(t *testing.T) {
	inv, err := store.LoadInventory(qbpV32Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	migrated, report, err := Migrate(inv, nil)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// (a) All 59 anchors present.
	if got := len(migrated.Anchors); got != 59 {
		t.Errorf("expected 59 anchors, got %d", got)
	}

	// (b) Schema version bumped to v0.3.
	if migrated.SchemaVersion != store.SchemaV03 {
		t.Errorf("expected schema_version %q, got %q", store.SchemaV03, migrated.SchemaVersion)
	}

	proofSysAnchors := map[string]string{
		"PROOF-hurwitz":     "established_mathematics",
		"PROOF-42zd":        "lean4",
		"PROOF-hessian":     "lean4",
		"PROOF-eigenratios": "lean4",
		"PROOF-g2":          "python_exhaustive",
		"PROOF-fano":        "lean4",
		"PROOF-cl6":         "lean4",
		"PROOF-3gen":        "lean4",
		"PROOF-born":        "established_mathematics",
		"PROOF-shells":      "lean4",
	}

	for _, a := range migrated.Anchors {
		// (c) Anchors with ProofSystem → ProvenanceKind = proof, ProofState = written.
		if proofSys, isProof := proofSysAnchors[a.ID]; isProof {
			if a.ProvenanceKind != model.ProvenanceKindProof {
				t.Errorf("anchor %s: expected ProvenanceKind=proof, got %s", a.ID, a.ProvenanceKind)
			}
			if a.ProofLanguage != proofSys {
				t.Errorf("anchor %s: expected ProofLanguage=%q, got %q", a.ID, proofSys, a.ProofLanguage)
			}
			if a.ProofState != model.ProofStateWritten {
				t.Errorf("anchor %s: expected ProofState=written, got %s", a.ID, a.ProofState)
			}
			continue
		}

		// (b) Non-proof T-provenance → ProvenanceKind = theory (safe default).
		if a.Provenance == model.ProvenanceTheoretical {
			if a.ProvenanceKind != model.ProvenanceKindTheory {
				t.Errorf("anchor %s: T-provenance expected theory, got %s", a.ID, a.ProvenanceKind)
			}
		}

		// (d) ID prefixes preserved.
		if a.ID == "" {
			t.Error("found anchor with empty ID")
		}
	}

	// All 10 proof-system anchors caused a decision-needed or mechanical entry.
	if report.AnchorCount != 59 {
		t.Errorf("report.AnchorCount: expected 59, got %d", report.AnchorCount)
	}

	// (e) Backwards-compat round-trip: serialise + re-parse with LoadInventory.
	tmpDir := t.TempDir()
	outPath := tmpDir + "/qbp_v3_2_migrated.json"
	if err := store.SaveInventory(migrated, outPath); err != nil {
		t.Fatalf("SaveInventory: %v", err)
	}
	reloaded, err := store.LoadInventory(outPath)
	if err != nil {
		t.Fatalf("round-trip LoadInventory: %v", err)
	}
	if len(reloaded.Anchors) != 59 {
		t.Errorf("round-trip: expected 59 anchors, got %d", len(reloaded.Anchors))
	}
}

// TestMigrate_WithDecisions verifies that a caller-supplied decisions JSON with
// one theory-external anchor is applied correctly.
func TestMigrate_WithDecisions(t *testing.T) {
	inv, err := store.LoadInventory(qbpV32Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	decisions := []MigrationDecision{
		{
			AnchorID:       "MEAS-alpha",
			ProvenanceKind: "theory-external",
			TheoryCitation: "PDG (2024). Review of Particle Physics.",
			TheoryDOI:      "10.1093/ptep/ptad058",
			TheoryURL:      "https://pdg.lbl.gov/",
		},
	}

	migrated, report, err := Migrate(inv, decisions)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Find the MEAS-alpha anchor.
	var found *model.Anchor
	for i := range migrated.Anchors {
		if migrated.Anchors[i].ID == "MEAS-alpha" {
			found = &migrated.Anchors[i]
			break
		}
	}
	if found == nil {
		t.Fatal("MEAS-alpha not found in migrated inventory")
	}

	if found.ProvenanceKind != model.ProvenanceKindTheoryExternal {
		t.Errorf("MEAS-alpha: expected ProvenanceKind=theory-external, got %s", found.ProvenanceKind)
	}
	if found.TheoryCitation != "PDG (2024). Review of Particle Physics." {
		t.Errorf("MEAS-alpha: unexpected TheoryCitation: %q", found.TheoryCitation)
	}
	if found.TheoryDOI != "10.1093/ptep/ptad058" {
		t.Errorf("MEAS-alpha: unexpected TheoryDOI: %q", found.TheoryDOI)
	}
	if found.TheoryURL != "https://pdg.lbl.gov/" {
		t.Errorf("MEAS-alpha: unexpected TheoryURL: %q", found.TheoryURL)
	}

	// Decision must appear in DecisionsApplied.
	var applied bool
	for _, id := range report.DecisionsApplied {
		if id == "MEAS-alpha" {
			applied = true
			break
		}
	}
	if !applied {
		t.Error("MEAS-alpha not listed in report.DecisionsApplied")
	}
}

// TestMigrate_WithDecisions_MissingCitation verifies that theory-external
// without a citation returns an error.
func TestMigrate_WithDecisions_MissingCitation(t *testing.T) {
	inv, err := store.LoadInventory(lifecycleV2Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}
	decisions := []MigrationDecision{
		{
			AnchorID:       "PROOF-derivation",
			ProvenanceKind: "theory-external",
			TheoryCitation: "", // missing
		},
	}
	_, _, err = Migrate(inv, decisions)
	if err == nil {
		t.Fatal("expected error for missing theory_citation, got nil")
	}
	if !strings.Contains(err.Error(), "theory_citation") {
		t.Errorf("error should mention theory_citation; got: %v", err)
	}
}

// TestMigrate_HeuristicSuggestions verifies that the heuristic correctly
// classifies synthetic anchors.
func TestMigrate_HeuristicSuggestions(t *testing.T) {
	tests := []struct {
		name           string
		wantSuggestion string
		anchor         model.Anchor
	}{
		{
			name: "proof_file set → theory",
			anchor: model.Anchor{
				ID:        "PROOF-with-file",
				ProofFile: "proofs/foo.lean",
			},
			wantSuggestion: "theory",
		},
		{
			name: "description with citation year and external keyword → theory-external",
			anchor: model.Anchor{
				ID:          "PROOF-external-cite",
				Description: "Relies on Hurwitz 1898 theorem as external authority.",
			},
			wantSuggestion: "theory-external",
		},
		{
			name: "description with bare citation year → theory-external",
			anchor: model.Anchor{
				ID:          "PROOF-year-only",
				Description: "After Einstein 1905, the special theory of relativity implies...",
			},
			wantSuggestion: "theory-external",
		},
		{
			name: "no citation pattern → theory",
			anchor: model.Anchor{
				ID:          "PROOF-internal",
				Description: "Algebraic derivation from axioms. No external references.",
			},
			wantSuggestion: "theory",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := suggestProvenance(tc.anchor)
			if got != tc.wantSuggestion {
				t.Errorf("suggestProvenance: expected %q, got %q", tc.wantSuggestion, got)
			}
		})
	}
}

// TestMigrate_PredictionsLifecycle migrates the lifecycle fixture and checks
// structural invariants.
func TestMigrate_PredictionsLifecycle(t *testing.T) {
	inv, err := store.LoadInventory(lifecycleV2Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	migrated, _, err := Migrate(inv, nil)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// (a) Status passes through unchanged.
	// Expected from fixture: coherent x3, contested x1, refuted x1, untested x1.
	statusCounts := map[string]int{}
	for _, a := range migrated.Anchors {
		statusCounts[a.Status.String()]++
	}
	wantStatuses := map[string]int{
		"coherent":  3,
		"contested": 1,
		"refuted":   1,
		"untested":  1,
	}
	for s, want := range wantStatuses {
		if got := statusCounts[s]; got != want {
			t.Errorf("status %q: expected %d anchors, got %d", s, want, got)
		}
	}

	// (b) ProofState is absent (zero value) for non-proof anchors (Invariant 4).
	for _, a := range migrated.Anchors {
		if a.ProvenanceKind != model.ProvenanceKindProof && a.ProofState != model.ProofStateUnknown {
			t.Errorf("anchor %s: non-proof anchor has proof_state %s (Invariant 4 violation)", a.ID, a.ProofState)
		}
	}

	// (c) Generated v0.3 round-trips through LoadInventory.
	tmpDir := t.TempDir()
	outPath := tmpDir + "/lifecycle_v03.json"
	if err := store.SaveInventory(migrated, outPath); err != nil {
		t.Fatalf("SaveInventory: %v", err)
	}
	reloaded, err := store.LoadInventory(outPath)
	if err != nil {
		t.Fatalf("round-trip LoadInventory: %v", err)
	}
	if len(reloaded.Anchors) != len(inv.Anchors) {
		t.Errorf("round-trip: anchor count mismatch: %d vs %d", len(reloaded.Anchors), len(inv.Anchors))
	}
}

// TestMigrate_Idempotent verifies that migrating an already-v0.3 inventory is
// a no-op with a warning in the report.
func TestMigrate_Idempotent(t *testing.T) {
	inv, err := store.LoadInventory(lifecycleV3Fixture)
	if err != nil {
		t.Fatalf("LoadInventory v0.3 fixture: %v", err)
	}
	// Confirm fixture is already v0.3.
	if inv.SchemaVersion != store.SchemaV03 {
		t.Fatalf("fixture is not v0.3; got %q", inv.SchemaVersion)
	}

	migrated, report, err := Migrate(inv, nil)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if migrated.SchemaVersion != store.SchemaV03 {
		t.Errorf("idempotent: schema_version should remain v0.3, got %q", migrated.SchemaVersion)
	}
	var hasAlreadyWarning bool
	for _, w := range report.Warnings {
		if strings.Contains(w, "already") {
			hasAlreadyWarning = true
			break
		}
	}
	if !hasAlreadyWarning {
		t.Errorf("idempotent: expected 'already' warning in report; got warnings: %v", report.Warnings)
	}
}

// TestRunMigrate_DefaultMode verifies that the default mode writes a v0.3 JSON
// file and a companion migration-report.md.
func TestRunMigrate_DefaultMode(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := tmpDir + "/lifecycle_v03.json"

	err := runMigrate([]string{lifecycleV2Fixture, "-o", outPath})
	if err != nil {
		t.Fatalf("runMigrate: %v", err)
	}

	// (a) v0.3 output file written.
	if _, serr := os.Stat(outPath); serr != nil {
		t.Fatalf("output file not created: %v", serr)
	}

	// (b) Companion report written alongside.
	reportPath := outPath + ".migration-report.md"
	if _, serr := os.Stat(reportPath); serr != nil {
		t.Fatalf("migration-report.md not created: %v", serr)
	}

	// (c) Output is a valid v0.3 inventory.
	reloaded, err := store.LoadInventory(outPath)
	if err != nil {
		t.Fatalf("reloaded output inventory: %v", err)
	}
	if reloaded.SchemaVersion != store.SchemaV03 {
		t.Errorf("output schema_version: expected v0.3, got %q", reloaded.SchemaVersion)
	}
}

// TestRunMigrate_CheckMode verifies that --check does NOT write any output file
// but emits the report to stdout.
func TestRunMigrate_CheckMode(t *testing.T) {
	// Copy the v0.2 fixture into a temp dir so the default output path
	// resolves inside the temp dir and cannot collide with the testdata tree.
	srcData, err := os.ReadFile(lifecycleV2Fixture)
	if err != nil {
		t.Fatalf("read lifecycle fixture: %v", err)
	}
	tmpDir := t.TempDir()
	tmpSrc := tmpDir + "/lifecycle.json"
	if err := os.WriteFile(tmpSrc, srcData, 0o644); err != nil { //nolint:gosec // G306: test writes non-sensitive inventory to t.TempDir()
		t.Fatalf("write tmp src: %v", err)
	}
	expectedOut := tmpDir + "/lifecycle_v0_3.json"

	stdoutContent := captureStdout(func() {
		err = runMigrate([]string{tmpSrc, "--check"})
		if err != nil {
			t.Errorf("runMigrate --check: %v", err)
		}
	})

	// (a) Report emitted to stdout.
	if !strings.Contains(stdoutContent, "Migration report") {
		t.Errorf("--check mode: expected 'Migration report' in stdout; got: %q", stdoutContent[:min(200, len(stdoutContent))])
	}

	// (b) No output JSON written.
	if _, serr := os.Stat(expectedOut); serr == nil {
		t.Errorf("--check mode wrote output file %s but should not", expectedOut)
		_ = os.Remove(expectedOut)
	}
}

// TestRunMigrate_OutputFlag verifies that -o writes to the specified path.
func TestRunMigrate_OutputFlag(t *testing.T) {
	outPath := t.TempDir() + "/custom_output.json"
	if err := runMigrate([]string{lifecycleV2Fixture, "-o", outPath}); err != nil {
		t.Fatalf("runMigrate -o: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not found at -o path: %v", err)
	}
}

// TestRunMigrate_MissingInventoryArg verifies that omitting the positional
// argument returns an error.
func TestRunMigrate_MissingInventoryArg(t *testing.T) {
	err := runMigrate([]string{})
	if err == nil {
		t.Fatal("expected error for missing inventory, got nil")
	}
	if !strings.Contains(err.Error(), "expects one inventory.json") {
		t.Errorf("error should mention inventory argument; got: %v", err)
	}
}

// TestRunMigrate_InvalidDecisionsFile verifies that a malformed JSON decisions
// file returns an error.
func TestRunMigrate_InvalidDecisionsFile(t *testing.T) {
	decPath := t.TempDir() + "/bad_decisions.json"
	if err := os.WriteFile(decPath, []byte("{not valid json"), 0o644); err != nil { //nolint:gosec // G306: test writes non-sensitive data to t.TempDir()
		t.Fatalf("write bad decisions file: %v", err)
	}
	outPath := t.TempDir() + "/out.json"
	err := runMigrate([]string{lifecycleV2Fixture, "--decisions", decPath, "-o", outPath})
	if err == nil {
		t.Fatal("expected error for invalid decisions JSON, got nil")
	}
}

// TestLoadDecisions_ValidFile verifies that a well-formed decisions JSON file
// is parsed correctly.
func TestLoadDecisions_ValidFile(t *testing.T) {
	df := decisionFile{
		Anchors: []MigrationDecision{
			{
				AnchorID:       "PROOF-foo",
				ProvenanceKind: "theory-external",
				TheoryCitation: "Hurwitz, A. (1898).",
				TheoryDOI:      "10.1000/placeholder",
				TheoryURL:      "https://example.org/",
			},
		},
	}
	data, err := json.Marshal(df)
	if err != nil {
		t.Fatalf("marshal test decisions: %v", err)
	}
	decPath := t.TempDir() + "/decisions.json"
	if err := os.WriteFile(decPath, data, 0o644); err != nil { //nolint:gosec // G306: test writes non-sensitive data to t.TempDir()
		t.Fatalf("write decisions file: %v", err)
	}

	loaded, err := LoadDecisions(decPath)
	if err != nil {
		t.Fatalf("LoadDecisions: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(loaded))
	}
	if loaded[0].AnchorID != "PROOF-foo" {
		t.Errorf("unexpected AnchorID: %q", loaded[0].AnchorID)
	}
	if loaded[0].TheoryCitation != "Hurwitz, A. (1898)." {
		t.Errorf("unexpected TheoryCitation: %q", loaded[0].TheoryCitation)
	}
}

// TestLoadDecisions_EmptyPath verifies that an empty path returns nil slice, nil error.
func TestLoadDecisions_EmptyPath(t *testing.T) {
	got, err := LoadDecisions("")
	if err != nil {
		t.Fatalf("LoadDecisions empty path: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(got))
	}
}

// TestFormatReport_ContainsRequiredSections verifies that FormatReport emits
// the key report sections.
func TestFormatReport_ContainsRequiredSections(t *testing.T) {
	report := MigrationReport{
		SourcePath:      "foo.json",
		OutputPath:      "foo_v03.json",
		AnchorCount:     5,
		MechanicalCount: 3,
		DecisionsNeeded: []DecisionPrompt{
			{AnchorID: "PROOF-x", AnchorName: "X", Suggestion: "theory", Rationale: "test"},
		},
		DecisionsApplied: []string{"PROOF-y"},
		Warnings:         []string{"anchor PROOF-z: proof_state set to written"},
	}
	md := FormatReport(report)

	requiredSubstrings := []string{
		"Migration report",
		"foo.json",
		"foo_v03.json",
		"Total anchors: 5",
		"Mechanically translated: 3",
		"Decisions still needed: 1",
		"PROOF-x",
		"## Decisions applied",
		"PROOF-y",
		"## Warnings",
		"PROOF-z",
		"## Re-run command",
	}
	for _, want := range requiredSubstrings {
		if !strings.Contains(md, want) {
			t.Errorf("FormatReport: missing %q", want)
		}
	}
}
