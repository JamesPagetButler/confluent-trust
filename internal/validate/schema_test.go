package validate

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestSchemaCompiles ensures the embedded schema parses and compiles.
func TestSchemaCompiles(t *testing.T) {
	if _, err := loadSchema(); err != nil {
		t.Fatalf("schema failed to compile: %v", err)
	}
}

// TestSchemaInSync ensures internal/validate/schema.json (embedded)
// matches the canonical schema/inventory.schema.json byte-for-byte.
// Update both with `go generate ./...` if you change one.
func TestSchemaInSync(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "schema", "inventory.schema.json"))
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}
	if !bytes.Equal(rawSchema, canonical) {
		t.Fatal("internal/validate/schema.json drifted from schema/inventory.schema.json — re-copy")
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
			data, err := os.ReadFile(path)
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
