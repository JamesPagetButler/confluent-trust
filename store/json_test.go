package store

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixturesDir = "../testdata"

func TestLoadInventory_AllFixtures(t *testing.T) {
	fixtures := []string{
		"minimal.json",
		"qbp_quantum_v0_1.json",
		"qbp_quantum_v0_2.json",
		"qbp_dm_fork.json",
		"qbp_v3_2.json",
		"predictions_lifecycle.json",
		"predictions_lifecycle_v0_3.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			inv, err := LoadInventory(filepath.Join(fixturesDir, name))
			if err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
			if inv.Programme == "" {
				t.Errorf("%s: empty programme after load", name)
			}
			for _, c := range inv.ConfluencePoints {
				if c.LegacyPathA != nil || c.LegacyPathB != nil {
					t.Errorf("%s: confluence %s has legacy fields after load", name, c.ID)
				}
			}
		})
	}
}

func TestLoadSave_RoundTripDeepEqual(t *testing.T) {
	fixtures := []string{
		"minimal.json",
		"qbp_quantum_v0_2.json",
		"qbp_dm_fork.json",
		"qbp_v3_2.json",
		"predictions_lifecycle.json",
		"predictions_lifecycle_v0_3.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			src := filepath.Join(fixturesDir, name)
			loaded, err := LoadInventory(src)
			if err != nil {
				t.Fatalf("first load: %v", err)
			}

			out := filepath.Join(t.TempDir(), name)
			if err := SaveInventory(loaded, out); err != nil {
				t.Fatalf("save: %v", err)
			}

			reloaded, err := LoadInventory(out)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			if !reflect.DeepEqual(loaded, reloaded) {
				t.Errorf("%s: round-trip diverged", name)
			}
		})
	}
}

func TestLoadInventory_LegacyConfluenceMigrates(t *testing.T) {
	// qbp_quantum_v0_2.json (the inventory file) actually still uses the
	// binary path_a/path_b shape internally. After load, those fields must
	// be normalized into Paths.
	inv, err := LoadInventory(filepath.Join(fixturesDir, "qbp_quantum_v0_2.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var seenPaths bool
	for _, c := range inv.ConfluencePoints {
		if c.LegacyPathA != nil || c.LegacyPathB != nil {
			t.Errorf("confluence %s: legacy fields not cleared", c.ID)
		}
		if len(c.Paths) > 0 {
			seenPaths = true
		}
	}
	if !seenPaths {
		t.Error("expected at least one confluence with normalized Paths")
	}
}

func TestLoadInventory_UnsupportedSchemaVersion(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.json")
	bad := []byte(`{"schema_version":"v9.9","programme":"X","version":"0.1","axioms":[],"anchors":[],"chains":[]}`)
	if err := writeRaw(tmp, bad); err != nil {
		t.Fatal(err)
	}
	_, err := LoadInventory(tmp)
	if err == nil {
		t.Fatal("expected error for unsupported schema version")
	}
	if !strings.Contains(err.Error(), "unsupported schema version") {
		t.Errorf("error %q does not mention unsupported schema version", err)
	}
}

func TestLoadInventory_RejectsMalformedJSON(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.json")
	if err := writeRaw(tmp, []byte(`{not-json`)); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadInventory(tmp); err == nil {
		t.Fatal("expected error on malformed JSON")
	}
}

func TestLoadMultiple(t *testing.T) {
	paths := []string{
		filepath.Join(fixturesDir, "minimal.json"),
		filepath.Join(fixturesDir, "qbp_dm_fork.json"),
	}
	invs, err := LoadMultiple(paths)
	if err != nil {
		t.Fatalf("LoadMultiple: %v", err)
	}
	if got, want := len(invs), 2; got != want {
		t.Errorf("got %d inventories, want %d", got, want)
	}

	// One bad path among good ones — error should aggregate.
	withBad := append([]string{}, paths...)
	withBad = append(withBad, filepath.Join(t.TempDir(), "does-not-exist.json"))
	if _, err := LoadMultiple(withBad); err == nil {
		t.Error("expected error when one path is missing")
	}
}

// TestLoadInventory_QBPv3_2_MatchesAnalysisReport pins the structural
// counts of testdata/qbp_v3_2.json to those documented in the QBP-CTH
// Analysis Report v3.2 §5.1 (in this repo at doc/QBP-CTH-Analysis-Report-v3_2.md).
// Encodes the "All v3.2 anchors preserved (count + ids match the report)"
// acceptance criterion from issue #52.
func TestLoadInventory_QBPv3_2_MatchesAnalysisReport(t *testing.T) {
	inv, err := LoadInventory(filepath.Join(fixturesDir, "qbp_v3_2.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got, want := inv.Programme, "QBP"; got != want {
		t.Errorf("programme: got %q want %q", got, want)
	}
	if got, want := inv.Version, "3.2"; got != want {
		t.Errorf("version: got %q want %q", got, want)
	}

	// Top-level counts per analysis report §5.1.
	checks := []struct {
		name string
		got  int
		want int
	}{
		{"axioms", len(inv.Axioms), 2},
		{"derived_principles", len(inv.DerivedPrinciples), 9},
		{"anchors", len(inv.Anchors), 59},
		{"inputs", len(inv.Inputs), 5},
		{"chains", len(inv.Chains), 21},
		{"confluence_points", len(inv.ConfluencePoints), 8},
		{"fork_points", len(inv.ForkPoints), 0},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %d want %d", c.name, c.got, c.want)
		}
	}

	// Tier breakdown per analysis report §5.1: Tier 1 = 24, Tier 2 = 29,
	// Tier 3 = 6. (Axioms are counted separately above as Tier 0.)
	tierCounts := map[int]int{}
	for _, a := range inv.Anchors {
		tierCounts[int(a.Tier)]++
	}
	tierChecks := []struct {
		tier int
		want int
	}{
		{1, 24},
		{2, 29},
		{3, 6},
	}
	for _, c := range tierChecks {
		if tierCounts[c.tier] != c.want {
			t.Errorf("Tier %d anchors: got %d want %d", c.tier, tierCounts[c.tier], c.want)
		}
	}

	// All 5 inputs are irreducible per report §5.1.
	var irreducible int
	for _, in := range inv.Inputs {
		if in.Status == "irreducible" {
			irreducible++
		}
	}
	if irreducible != 5 {
		t.Errorf("irreducible inputs: got %d want 5", irreducible)
	}

	// Confluence points must be N-ary normalized post-load (Paths populated,
	// legacy path_a/path_b cleared).
	for _, c := range inv.ConfluencePoints {
		if c.LegacyPathA != nil || c.LegacyPathB != nil {
			t.Errorf("confluence %s: legacy fields not cleared", c.ID)
		}
		if len(c.Paths) == 0 {
			t.Errorf("confluence %s: no normalized Paths", c.ID)
		}
	}
}

func TestSaveInventory_AtomicWrite(t *testing.T) {
	src := filepath.Join(fixturesDir, "minimal.json")
	inv, err := LoadInventory(src)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "out.json")
	if err := SaveInventory(inv, dst); err != nil {
		t.Fatalf("save: %v", err)
	}

	// .tmp staging file must not exist after a successful rename.
	if _, err := osStat(dst + ".tmp"); err == nil {
		t.Error("staging .tmp file should be cleaned up after rename")
	}
}
