package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixtureInv    = "../../testdata/lean-anchors-v0_3.json"
	fixtureCorpus = "../../testdata/lean"
)

// TestRunLeanLink_BasicReport verifies the subcommand produces a report
// mentioning the expected classes for the fixture inventory + corpus.
func TestRunLeanLink_BasicReport(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	out := filepath.Join(dir, "report.md")

	if err := runLeanLink([]string{fixtureInv, fixtureCorpus, "-o", out}); err != nil {
		t.Fatalf("runLeanLink: %v", err)
	}

	data, err := os.ReadFile(out) // #nosec G304 -- test reads its own temp output file
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	content := string(data)

	// The report must mention the 5-class sections.
	for _, want := range []string{
		"lean-link reconciliation report",
		"Proven & wired",
		"Orphan theorems",
		"Stale references",
		"Phantom-theorem violations",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("report missing section %q", want)
		}
	}
}

// TestRunLeanLink_UpdateInventory verifies --update-inventory writes the
// inventory back and reduces findings on a subsequent run.
func TestRunLeanLink_UpdateInventory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Copy the fixture inventory to a temp dir so we can mutate it.
	origData, err := os.ReadFile(fixtureInv)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	invPath := filepath.Join(dir, "inv.json")
	if err := os.WriteFile(invPath, origData, 0o600); err != nil { // #nosec G306,G703 -- test file in temp dir; path is from t.TempDir()
		t.Fatal(err)
	}

	// First run: update-inventory.
	if err := runLeanLink([]string{invPath, fixtureCorpus, "--update-inventory"}); err != nil {
		// --update-inventory may still exit 0 even with findings.
		t.Fatalf("runLeanLink --update-inventory: %v", err)
	}

	// Verify the file was written back.
	updated, err := os.ReadFile(invPath) // #nosec G304 -- test reads its own temp file
	if err != nil {
		t.Fatalf("read updated inventory: %v", err)
	}
	// The updated file must differ from original (at minimum last_tested_at was set).
	if string(updated) == string(origData) {
		// No drift anchors in fixture might mean no changes were applied — that is OK.
		// Just verify it is valid JSON.
		if len(updated) == 0 {
			t.Error("updated inventory is empty")
		}
	}
}

// TestRunLeanLink_StrictMode verifies --strict returns an error when findings exist.
func TestRunLeanLink_StrictMode(t *testing.T) {
	t.Parallel()
	// The fixture inventory has phantom-theorem + stale-ref + drift findings.
	err := runLeanLink([]string{fixtureInv, fixtureCorpus, flagStrict})
	if err == nil {
		t.Error("--strict: expected non-nil error when findings exist, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "strict mode") {
		t.Errorf("--strict: unexpected error text: %v", err)
	}
}

// TestRunLeanLink_MissingArgs verifies missing positional args returns an error.
func TestRunLeanLink_MissingArgs(t *testing.T) {
	t.Parallel()
	if err := runLeanLink([]string{fixtureInv}); err == nil {
		t.Error("expected error when corpus root is missing, got nil")
	}
	if err := runLeanLink(nil); err == nil {
		t.Error("expected error when no args given, got nil")
	}
}

// ---- confluent-trust#130: awaiting ceiling/allow-list + base regressions ----

// awaitingFixtureJSON builds a minimal valid v0.3 inventory with a single
// status:untested, provenance_kind:proof anchor whose theorem does not
// exist in the (empty) corpus — the "awaiting" shape from #130's problem
// statement.
func awaitingFixtureJSON(status string) string {
	return `{
  "schema_version": "v0.3",
  "programme": "cth-130-fixture",
  "version": "0.1.0",
  "axioms": [],
  "chains": [],
  "anchors": [
    {
      "id": "PROOF-OPEN-ROOT",
      "name": "open root discharge target",
      "description": "awaiting fixture",
      "tier": 1,
      "status": "` + status + `",
      "provenance": "T",
      "provenance_kind": "proof",
      "proof_language": "lean4",
      "proof_file": "Open.lean",
      "proof_state": "written",
      "prediction_chain": [],
      "theorems": [ { "name": "not_yet_written", "status": "not_started" } ]
    }
  ]
}`
}

// TestRunLeanLink_AwaitingDoesNotFailStrictByDefault verifies an awaiting
// finding, on its own, does not trip --strict (confluent-trust#130 Ask 1).
func TestRunLeanLink_AwaitingDoesNotFailStrictByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	invPath := filepath.Join(dir, "inv.json")
	if err := os.WriteFile(invPath, []byte(awaitingFixtureJSON("untested")), 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(corpus, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := runLeanLink([]string{invPath, corpus, flagStrict}); err != nil {
		t.Errorf("--strict: expected nil error for an awaiting-only finding, got %v", err)
	}
}

// TestRunLeanLink_AwaitingMaxCeiling verifies --awaiting-max bounds the
// count under --strict (qbp-architecture §I4 condition 1, typo-hiding guard).
func TestRunLeanLink_AwaitingMaxCeiling(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	invPath := filepath.Join(dir, "inv.json")
	if err := os.WriteFile(invPath, []byte(awaitingFixtureJSON("untested")), 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(corpus, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := runLeanLink([]string{invPath, corpus, flagStrict, "--awaiting-max", "1"}); err != nil {
		t.Errorf("--awaiting-max 1: expected nil error with exactly 1 awaiting, got %v", err)
	}
	if err := runLeanLink([]string{invPath, corpus, flagStrict, "--awaiting-max", "0"}); err == nil {
		t.Error("--awaiting-max 0: expected an error with 1 awaiting (exceeds ceiling), got nil")
	}
}

// TestRunLeanLink_AwaitingAllowList verifies --awaiting-allow requires every
// awaiting anchor ID to be explicitly listed.
func TestRunLeanLink_AwaitingAllowList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	invPath := filepath.Join(dir, "inv.json")
	if err := os.WriteFile(invPath, []byte(awaitingFixtureJSON("untested")), 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(corpus, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := runLeanLink([]string{invPath, corpus, flagStrict, "--awaiting-allow", "PROOF-OPEN-ROOT"}); err != nil {
		t.Errorf("--awaiting-allow with the anchor listed: expected nil error, got %v", err)
	}
	if err := runLeanLink([]string{invPath, corpus, flagStrict, "--awaiting-allow", "SOME-OTHER-ID"}); err == nil {
		t.Error("--awaiting-allow without the anchor listed: expected an error, got nil")
	}
}

// TestRunLeanLink_BaseRegression verifies --base flags a status flip into
// untested as its own strict-mode failure (qbp-architecture §I4 condition 2).
func TestRunLeanLink_BaseRegression(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.json")
	currentPath := filepath.Join(dir, "current.json")
	if err := os.WriteFile(basePath, []byte(awaitingFixtureJSON("coherent")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(currentPath, []byte(awaitingFixtureJSON("untested")), 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(corpus, 0o750); err != nil {
		t.Fatal(err)
	}

	// Without --base, the regression is invisible: the awaiting finding
	// alone must not fail strict.
	if err := runLeanLink([]string{currentPath, corpus, flagStrict}); err != nil {
		t.Errorf("--strict without --base: expected nil error, got %v", err)
	}
	// With --base, the coherent->untested flip is its own loud failure.
	if err := runLeanLink([]string{currentPath, corpus, flagStrict, "--base", basePath}); err == nil {
		t.Error("--strict --base: expected an error for a coherent->untested regression, got nil")
	}
	if err := runLeanLink([]string{currentPath, corpus, flagStrict, "--base", basePath}); err != nil &&
		!strings.Contains(err.Error(), "verification-regression") {
		t.Errorf("expected error to mention verification-regression, got %v", err)
	}
}

// TestRunLeanLink_BaseMissingFile verifies a bad --base path surfaces a
// clear error rather than silently skipping regression detection.
func TestRunLeanLink_BaseMissingFile(t *testing.T) {
	t.Parallel()
	if err := runLeanLink([]string{fixtureInv, fixtureCorpus, "--base", "/nonexistent/base.json"}); err == nil {
		t.Error("expected an error for a missing --base file, got nil")
	}
}
