package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout to a pipe for the duration of fn,
// then restores it and returns everything written.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

const lifecycleFixture = "../../testdata/predictions_lifecycle.json"

// TestRunScore_DefaultMode_LifecycleFixture verifies default mode against the
// lifecycle fixture: all five PRED-* IDs appear, regime labels are correct for
// the scored rows, and the untested row shows "—" in the regime column.
func TestRunScore_DefaultMode_LifecycleFixture(t *testing.T) {
	var err error
	out := captureStdout(func() {
		err = runScore([]string{lifecycleFixture})
	})
	if err != nil {
		t.Fatalf("runScore returned unexpected error: %v", err)
	}

	// (a) all five PRED-* anchor IDs appear
	wantIDs := []string{
		"PRED-laminar-match",
		"PRED-low-sediment-match",
		"PRED-moderate-discrepancy",
		"PRED-heavy-refutation",
		"PRED-untested-pending",
	}
	for _, id := range wantIDs {
		if !strings.Contains(out, id) {
			t.Errorf("output missing anchor ID %q", id)
		}
	}

	// (b) laminar anchor row shows regime "laminar"
	if !strings.Contains(out, "laminar") {
		t.Error("output missing regime label \"laminar\"")
	}

	// (c) heavy row shows regime "heavy"
	if !strings.Contains(out, "heavy") {
		t.Error("output missing regime label \"heavy\"")
	}

	// (d) untested row contains "—" in regime column
	// The untested row has the pattern: PRED-untested-pending … — (regime column)
	if !strings.Contains(out, "| PRED-untested-pending") {
		t.Error("output missing untested anchor row")
	}
	// Verify the row uses "—" for missing columns (at least four "—" on that line)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "PRED-untested-pending") {
			dashes := strings.Count(line, "—")
			if dashes < 4 {
				t.Errorf("untested row expected ≥4 '—' separators, got %d in line: %q", dashes, line)
			}
		}
	}
}

// TestRunScore_PredictionFlag_FiltersToOne verifies --prediction mode: output
// contains only the named anchor's full detail, including the correct regime
// and a discrepancy percentage.
func TestRunScore_PredictionFlag_FiltersToOne(t *testing.T) {
	var err error
	out := captureStdout(func() {
		err = runScore([]string{lifecycleFixture, "--prediction", "PRED-moderate-discrepancy"})
	})
	if err != nil {
		t.Fatalf("runScore returned unexpected error: %v", err)
	}

	// Must mention the anchor ID
	if !strings.Contains(out, "PRED-moderate-discrepancy") {
		t.Error("output missing anchor ID")
	}

	// Regime must be "moderate"
	if !strings.Contains(out, "moderate") {
		t.Error("output missing regime \"moderate\"")
	}

	// Discrepancy percentage: fixture value is 16.666…% → rendered as 16.6667%
	if !strings.Contains(out, "16.6667%") {
		t.Errorf("output missing expected discrepancy percentage; got:\n%s", out)
	}

	// Other anchor IDs must NOT appear (we're in single-anchor mode)
	unwantedIDs := []string{
		"PRED-laminar-match",
		"PRED-low-sediment-match",
		"PRED-heavy-refutation",
		"PRED-untested-pending",
	}
	for _, id := range unwantedIDs {
		if strings.Contains(out, id) {
			t.Errorf("output unexpectedly contains anchor %q in single-anchor mode", id)
		}
	}
}

// TestRunScore_PredictionFlag_NotFound verifies that requesting a non-existent
// anchor ID returns an error.
func TestRunScore_PredictionFlag_NotFound(t *testing.T) {
	err := runScore([]string{lifecycleFixture, "--prediction", "PRED-does-not-exist"})
	if err == nil {
		t.Fatal("expected error for missing anchor, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention \"not found\"; got: %v", err)
	}
}

// TestRunScore_RegimeFlag_GroupsAll verifies that --regime mode emits all four
// regime section headers plus an Untested section, and each named anchor
// appears under its expected section.
func TestRunScore_RegimeFlag_GroupsAll(t *testing.T) {
	var err error
	out := captureStdout(func() {
		err = runScore([]string{lifecycleFixture, "--regime"})
	})
	if err != nil {
		t.Fatalf("runScore returned unexpected error: %v", err)
	}

	// All four regime section headers must appear
	wantHeaders := []string{
		"## Laminar",
		"## Low Sediment",
		"## Moderate",
		"## Heavy",
		"## Untested",
	}
	for _, h := range wantHeaders {
		if !strings.Contains(out, h) {
			t.Errorf("output missing section header %q", h)
		}
	}

	// Verify each anchor lands in its expected section by checking ordering:
	// Laminar section should precede PRED-laminar-match, etc.
	anchorExpectedSection := map[string]string{
		"PRED-laminar-match":        "## Laminar",
		"PRED-low-sediment-match":   "## Low Sediment",
		"PRED-moderate-discrepancy": "## Moderate",
		"PRED-heavy-refutation":     "## Heavy",
		"PRED-untested-pending":     "## Untested",
	}
	for anchorID, sectionHeader := range anchorExpectedSection {
		sIdx := strings.Index(out, sectionHeader)
		aIdx := strings.Index(out, anchorID)
		if sIdx == -1 {
			t.Errorf("section header %q not found", sectionHeader)
			continue
		}
		if aIdx == -1 {
			t.Errorf("anchor %q not found in output", anchorID)
			continue
		}
		if aIdx < sIdx {
			t.Errorf("anchor %q appears before its section %q", anchorID, sectionHeader)
		}
		// Also check the anchor appears before the next section header (if any)
		nextHeaders := []string{"## Laminar", "## Low Sediment", "## Moderate", "## Heavy", "## Untested"}
		for _, nh := range nextHeaders {
			if nh == sectionHeader {
				continue
			}
			nhIdx := strings.Index(out, nh)
			if nhIdx == -1 {
				continue
			}
			// nhIdx must not fall between sIdx and aIdx for the anchor to be in the right section
			if nhIdx > sIdx && nhIdx < aIdx {
				t.Errorf("anchor %q appears after section %q, not its expected section %q",
					anchorID, nh, sectionHeader)
			}
		}
	}
}

// TestRunScore_BothFlagsConflict verifies that passing both --prediction and
// --regime together returns an error citing mutual exclusion.
func TestRunScore_BothFlagsConflict(t *testing.T) {
	err := runScore([]string{lifecycleFixture, "--prediction", "PRED-laminar-match", "--regime"})
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error message should mention \"mutually exclusive\"; got: %v", err)
	}
}

// TestRunScore_OutputFlag_WritesToFile verifies that -o writes markdown to disk
// and that the file is non-empty.
func TestRunScore_OutputFlag_WritesToFile(t *testing.T) {
	outPath := t.TempDir() + "/score-out.md"
	err := runScore([]string{lifecycleFixture, "-o", outPath})
	if err != nil {
		t.Fatalf("runScore returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath) //nolint:gosec // G304: test reads from t.TempDir(), not user-controlled input
	if err != nil {
		t.Fatalf("output file not found or unreadable: %v", err)
	}
	if len(data) == 0 {
		t.Error("output file is empty")
	}
	// Verify it looks like markdown (starts with "# ")
	content := string(data)
	if !strings.HasPrefix(content, "# ") {
		t.Errorf("output file does not start with a markdown header; got: %q", content[:min(40, len(content))])
	}
}

// TestRunScore_MissingInventoryArg verifies that calling score with no
// positional arguments returns an error.
func TestRunScore_MissingInventoryArg(t *testing.T) {
	err := runScore([]string{})
	if err == nil {
		t.Fatal("expected error for missing inventory argument, got nil")
	}
	if !strings.Contains(err.Error(), "expects one inventory.json") {
		t.Errorf("error message should mention inventory argument; got: %v", err)
	}
}

// min is a helper for pre-Go-1.21 compatibility if needed.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
