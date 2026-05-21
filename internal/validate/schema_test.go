package validate

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestSchemaCompiles ensures the embedded schema parses and compiles.
func TestSchemaCompiles(t *testing.T) {
	if _, err := loadSchema(); err != nil {
		t.Fatalf("schema failed to compile: %v", err)
	}
}

// TestSchemaInSync ensures internal/validate/schema.json (embedded) is
// semantically equivalent to the canonical schema/inventory.schema.json.
//
// Rationale: per sprint-1-closeout-2026-05-17 seq=12 Notary-bootstrap
// target #4 — Notary discipline upgrades verification from
// "compares-equal-bytes" to "validates-equivalent-meaning", catching
// drift only when the schemas actually differ. Key reordering, whitespace
// normalisation, or trailing-newline changes preserve semantics but would
// break a byte-equal check; this test is immune to such cosmetic changes.
// Update both files with `go generate ./...` if you change one.
func TestSchemaInSync(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "schema", "inventory.schema.json"))
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}

	var embeddedDoc, canonicalDoc interface{}
	if err := json.Unmarshal(rawSchema, &embeddedDoc); err != nil {
		t.Fatalf("parse embedded schema: %v", err)
	}
	if err := json.Unmarshal(canonical, &canonicalDoc); err != nil {
		t.Fatalf("parse canonical schema: %v", err)
	}

	if !reflect.DeepEqual(embeddedDoc, canonicalDoc) {
		t.Fatal("internal/validate/schema.json semantically diverged from schema/inventory.schema.json — re-copy with `go generate ./...`")
	}
}

// TestSchemaInSync_ByteEqual is a hygiene signal: it logs (but does not fail)
// when the two schema files are byte-different. Semantic equality is the
// load-bearing invariant (see TestSchemaInSync above); byte equality is
// desirable for diff hygiene but not required for correctness.
func TestSchemaInSync_ByteEqual(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "schema", "inventory.schema.json"))
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}
	if !bytes.Equal(rawSchema, canonical) {
		t.Logf("internal/validate/schema.json byte-different from schema/inventory.schema.json — semantically still in sync per TestSchemaInSync, but consider re-copying for hygiene (`go generate ./...`)")
	}
}

// TestValidate_AllFixtures validates every JSON file in testdata/
// against the schema. New fixtures must pass without modification.
func TestValidate_AllFixtures(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata")
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("read testdata/: %v", err)
	}
	var found int
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(fixturesDir, name)
			data, err := os.ReadFile(path) // #nosec G304 -- test reads a fixed-prefix testdata path
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if err := Inventory(data); err != nil {
				t.Fatalf("validate %s: %v", path, err)
			}
		})
		found++
	}
	if found == 0 {
		t.Fatal("no .json fixtures found in testdata/")
	}
}

func TestSchemaVersion(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"absent defaults to v0.1", `{"programme":"X","version":"0.1","axioms":[],"anchors":[],"chains":[]}`, "v0.1"},
		{"explicit v0.2", `{"programme":"X","version":"0.1","schema_version":"v0.2","axioms":[],"anchors":[],"chains":[]}`, "v0.2"},
		{"whitespace trimmed", `{"programme":"X","version":"0.1","schema_version":"  v0.2  ","axioms":[],"anchors":[],"chains":[]}`, "v0.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SchemaVersion([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
