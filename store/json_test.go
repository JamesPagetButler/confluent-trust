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
