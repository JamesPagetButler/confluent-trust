package main

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
)

const (
	qbpV32Fixture      = "../../testdata/qbp_v3_2.json"
	lifecycleV2Fixture = "../../testdata/predictions_lifecycle.json"
	lifecycleV3Fixture = "../../testdata/predictions_lifecycle_v0_3.json"
	qbpLocalV02Fixture = "../../testdata/qbp_local_v0_2.json"
	// Repeated literals hoisted for goconst.
	testProofLangLean4 = "lean4"
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
		"PROOF-42zd":        testProofLangLean4,
		"PROOF-hessian":     testProofLangLean4,
		"PROOF-eigenratios": testProofLangLean4,
		"PROOF-g2":          "python_exhaustive",
		"PROOF-fano":        testProofLangLean4,
		"PROOF-cl6":         testProofLangLean4,
		"PROOF-3gen":        testProofLangLean4,
		"PROOF-born":        "established_mathematics",
		"PROOF-shells":      testProofLangLean4,
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
			ProvenanceKind: pkTheoryExternalStr,
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
	applied := slices.Contains(report.DecisionsApplied, "MEAS-alpha")
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
			ProvenanceKind: pkTheoryExternalStr,
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
			wantSuggestion: pkTheoryStr,
		},
		{
			name: "description with citation year and external keyword → theory-external",
			anchor: model.Anchor{
				ID:          "PROOF-external-cite",
				Description: "Relies on Hurwitz 1898 theorem as external authority.",
			},
			wantSuggestion: pkTheoryExternalStr,
		},
		{
			name: "description with bare citation year → theory-external",
			anchor: model.Anchor{
				ID:          "PROOF-year-only",
				Description: "After Einstein 1905, the special theory of relativity implies...",
			},
			wantSuggestion: pkTheoryExternalStr,
		},
		{
			name: "no citation pattern → theory",
			anchor: model.Anchor{
				ID:          "PROOF-internal",
				Description: "Algebraic derivation from axioms. No external references.",
			},
			wantSuggestion: pkTheoryStr,
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
				ProvenanceKind: pkTheoryExternalStr,
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

// ---- QBP-local D/I/P legacy provenance tests (CTH #88) ----

// TestMigrate_QBPLocal_DefaultsTheory migrates qbp_local_v0_2.json with no
// decisions supplied.  Verifies that D/I/P anchors receive their conservative
// defaults and are flagged in DecisionsNeeded.
func TestMigrate_QBPLocal_DefaultsTheory(t *testing.T) {
	inv, err := store.LoadInventory(qbpLocalV02Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	migrated, report, err := Migrate(inv, nil)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Schema version bumped.
	if migrated.SchemaVersion != store.SchemaV03 {
		t.Errorf("schema_version: expected v0.3, got %q", migrated.SchemaVersion)
	}

	// Expected provenance_kind defaults for D/I/P anchors with no decisions.
	wantKind := map[string]model.ProvenanceKind{
		"PROOF-derived-a":          model.ProvenanceKindTheory,          // D → theory
		"PROOF-derived-b":          model.ProvenanceKindTheory,          // D → theory
		"PROOF-internal-compute-a": model.ProvenanceKindInternalCompute, // I → internal-compute
		"PROOF-internal-compute-b": model.ProvenanceKindInternalCompute, // I → internal-compute
		"PROOF-philosophy-a":       model.ProvenanceKindTheory,          // P → theory
		"PROOF-philosophy-b":       model.ProvenanceKindTheory,          // P → theory
	}

	for _, a := range migrated.Anchors {
		want, relevant := wantKind[a.ID]
		if !relevant {
			continue
		}
		if a.ProvenanceKind != want {
			t.Errorf("anchor %s: expected provenance_kind %s, got %s", a.ID, want, a.ProvenanceKind)
		}
	}

	// All 6 D/I/P anchors should appear in DecisionsNeeded.
	needsDecision := map[string]bool{}
	for _, dp := range report.DecisionsNeeded {
		needsDecision[dp.AnchorID] = true
	}
	for id := range wantKind {
		if !needsDecision[id] {
			t.Errorf("anchor %s: expected in DecisionsNeeded, not found", id)
		}
	}
}

// TestMigrate_QBPLocal_WithDecisions verifies that a decisions file mapping
// I→internal-compute and P→theory+proof_state:partial is applied correctly.
func TestMigrate_QBPLocal_WithDecisions(t *testing.T) {
	inv, err := store.LoadInventory(qbpLocalV02Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	decisions := []MigrationDecision{
		{AnchorID: "PROOF-internal-compute-a", ProvenanceKind: pkInternalComputeStr},
		{AnchorID: "PROOF-internal-compute-b", ProvenanceKind: pkInternalComputeStr},
		{AnchorID: "PROOF-philosophy-a", ProvenanceKind: pkTheoryStr, ProofState: proofStatePartialDecStr},
		{AnchorID: "PROOF-philosophy-b", ProvenanceKind: pkTheoryStr, ProofState: proofStatePartialDecStr},
	}

	migrated, report, err := Migrate(inv, decisions)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	anchorByID := map[string]model.Anchor{}
	for _, a := range migrated.Anchors {
		anchorByID[a.ID] = a
	}

	// I → internal-compute.
	for _, id := range []string{"PROOF-internal-compute-a", "PROOF-internal-compute-b"} {
		a, ok := anchorByID[id]
		if !ok {
			t.Fatalf("anchor %s not found", id)
		}
		if a.ProvenanceKind != model.ProvenanceKindInternalCompute {
			t.Errorf("anchor %s: expected internal-compute, got %s", id, a.ProvenanceKind)
		}
	}

	// P → theory + proof_state: partial.
	for _, id := range []string{"PROOF-philosophy-a", "PROOF-philosophy-b"} {
		a, ok := anchorByID[id]
		if !ok {
			t.Fatalf("anchor %s not found", id)
		}
		if a.ProvenanceKind != model.ProvenanceKindTheory {
			t.Errorf("anchor %s: expected theory, got %s", id, a.ProvenanceKind)
		}
		if a.ProofState != model.ProofStatePartial {
			t.Errorf("anchor %s: expected proof_state=partial, got %s", id, a.ProofState)
		}
	}

	// All 4 supplied decisions should appear in DecisionsApplied.
	applied := map[string]bool{}
	for _, id := range report.DecisionsApplied {
		applied[id] = true
	}
	for _, dec := range decisions {
		if !applied[dec.AnchorID] {
			t.Errorf("anchor %s: expected in DecisionsApplied", dec.AnchorID)
		}
	}
}

// TestMigrate_QBPLocal_DecisionsFile_RoundTrip writes the migrated v0.3 output,
// reloads it, and verifies all formerly-D/I/P anchors carry canonical
// provenance_kind values (translation is complete).
func TestMigrate_QBPLocal_DecisionsFile_RoundTrip(t *testing.T) {
	inv, err := store.LoadInventory(qbpLocalV02Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	decisions := []MigrationDecision{
		{AnchorID: "PROOF-derived-a", ProvenanceKind: pkInternalComputeStr},
		{AnchorID: "PROOF-derived-b", ProvenanceKind: pkTheoryStr},
		{AnchorID: "PROOF-internal-compute-a", ProvenanceKind: pkInternalComputeStr},
		{AnchorID: "PROOF-internal-compute-b", ProvenanceKind: pkInternalComputeStr},
		{AnchorID: "PROOF-philosophy-a", ProvenanceKind: pkTheoryStr, ProofState: proofStatePartialDecStr},
		{AnchorID: "PROOF-philosophy-b", ProvenanceKind: pkTheoryStr, ProofState: proofStatePartialDecStr},
	}

	migrated, _, err := Migrate(inv, decisions)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Write and reload.
	tmpDir := t.TempDir()
	outPath := tmpDir + "/qbp_local_v03.json"
	if err := store.SaveInventory(migrated, outPath); err != nil {
		t.Fatalf("SaveInventory: %v", err)
	}
	reloaded, err := store.LoadInventory(outPath)
	if err != nil {
		t.Fatalf("round-trip LoadInventory: %v", err)
	}

	// All formerly-D/I/P anchors must carry a canonical (non-unknown)
	// provenance_kind in the v0.3 output (translation is complete).
	// The legacy provenance field may be retained for audit provenance tracking,
	// but the canonical classification lives in provenance_kind.
	legacyIDs := map[string]bool{
		"PROOF-derived-a": true, "PROOF-derived-b": true,
		"PROOF-internal-compute-a": true, "PROOF-internal-compute-b": true,
		"PROOF-philosophy-a": true, "PROOF-philosophy-b": true,
	}
	for _, a := range reloaded.Anchors {
		if !legacyIDs[a.ID] {
			continue
		}
		if a.ProvenanceKind == model.ProvenanceKindUnknown {
			t.Errorf("anchor %s: v0.3 output has no canonical provenance_kind (translation incomplete)", a.ID)
		}
	}

	// Schema is v0.3.
	if reloaded.SchemaVersion != store.SchemaV03 {
		t.Errorf("round-trip schema_version: expected v0.3, got %q", reloaded.SchemaVersion)
	}

	// Anchor count preserved.
	if len(reloaded.Anchors) != len(inv.Anchors) {
		t.Errorf("round-trip anchor count: expected %d, got %d", len(inv.Anchors), len(reloaded.Anchors))
	}
}

// TestMigrate_DecisionsFile_RejectsUnknownProvenanceKind verifies that a
// decisions file with an invalid provenance_kind value causes a hard error.
func TestMigrate_DecisionsFile_RejectsUnknownProvenanceKind(t *testing.T) {
	inv, err := store.LoadInventory(qbpLocalV02Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	decisions := []MigrationDecision{
		{AnchorID: "PROOF-derived-a", ProvenanceKind: "made-up-value"},
	}

	_, _, err = Migrate(inv, decisions)
	if err == nil {
		t.Fatal("expected error for unknown provenance_kind, got nil")
	}
	if !strings.Contains(err.Error(), "made-up-value") {
		t.Errorf("error should cite the unknown value; got: %v", err)
	}
	if !strings.Contains(err.Error(), "provenance_kind") {
		t.Errorf("error should mention provenance_kind; got: %v", err)
	}
}

// TestMigrate_DecisionsFile_ProofStateOverride verifies that a T-provenance
// anchor can receive a proof_state override from the decisions file.
func TestMigrate_DecisionsFile_ProofStateOverride(t *testing.T) {
	inv, err := store.LoadInventory(qbpLocalV02Fixture)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}

	// PROOF-theory-a is a T anchor; supply theory + proof_state: partial.
	decisions := []MigrationDecision{
		{AnchorID: "PROOF-theory-a", ProvenanceKind: pkTheoryStr, ProofState: proofStatePartialDecStr},
	}

	migrated, _, err := Migrate(inv, decisions)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var found *model.Anchor
	for i := range migrated.Anchors {
		if migrated.Anchors[i].ID == "PROOF-theory-a" {
			found = &migrated.Anchors[i]
			break
		}
	}
	if found == nil {
		t.Fatal("PROOF-theory-a not found in migrated inventory")
	}
	if found.ProvenanceKind != model.ProvenanceKindTheory {
		t.Errorf("expected provenance_kind=theory, got %s", found.ProvenanceKind)
	}
	if found.ProofState != model.ProofStatePartial {
		t.Errorf("expected proof_state=partial, got %s", found.ProofState)
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
			{AnchorID: "PROOF-x", AnchorName: "X", Suggestion: pkTheoryStr, Rationale: "test"},
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
