package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "testdata", name)
	data, err := os.ReadFile(path) // #nosec G304 -- test reads a fixed-prefix testdata path
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// TestInventory_RoundTrip is Issue #2's acceptance test:
// load → unmarshal → marshal → unmarshal → deep-equal.
// We use the round-tripped output (not the original bytes) as the
// equality target because the canonical serialization drops legacy
// path_a/path_b fields and zero-valued optionals.
func TestInventory_RoundTrip(t *testing.T) {
	fixtures := []string{
		"minimal.json",
		"qbp_quantum_v0_2.json",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			raw := loadFixture(t, name)

			var first Inventory
			if err := json.Unmarshal(raw, &first); err != nil {
				t.Fatalf("first unmarshal: %v", err)
			}
			first.NormalizeConfluences()

			marshalled, err := json.Marshal(first)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var second Inventory
			if err := json.Unmarshal(marshalled, &second); err != nil {
				t.Fatalf("second unmarshal: %v", err)
			}
			second.NormalizeConfluences()

			if !reflect.DeepEqual(first, second) {
				t.Errorf("round-trip diverged for %s", name)
			}
		})
	}
}

func TestInventory_Validate_QBPQuantumV02(t *testing.T) {
	raw := loadFixture(t, "qbp_quantum_v0_2.json")
	var inv Inventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := inv.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestInventory_NormalizeConfluences_BinaryToNary(t *testing.T) {
	raw := loadFixture(t, "qbp_quantum_v0_2.json")
	var inv Inventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Pre-normalization: the v0.2 fixture currently uses path_a/path_b.
	var hadLegacy bool
	for _, c := range inv.ConfluencePoints {
		if c.LegacyPathA != nil || c.LegacyPathB != nil {
			hadLegacy = true
			break
		}
	}
	inv.NormalizeConfluences()

	for _, c := range inv.ConfluencePoints {
		if c.LegacyPathA != nil || c.LegacyPathB != nil {
			t.Errorf("confluence %s still has legacy fields after normalize", c.ID)
		}
	}
	if !hadLegacy {
		t.Logf("note: fixture had no legacy binary confluences to migrate")
	}
}

func TestAxiom_DerivableInvariant(t *testing.T) {
	good := Axiom{ID: "AXIOM-1", Name: "x", Statement: "y", Derivable: false}
	if err := good.Validate(); err != nil {
		t.Errorf("non-derivable axiom rejected: %v", err)
	}

	bad := Axiom{ID: "AXIOM-2", Name: "x", Statement: "y", Derivable: true}
	if err := bad.Validate(); err == nil {
		t.Error("derivable=true without derived_from_axioms should fail validation")
	}

	merged := Axiom{
		ID: "AXIOM-3", Name: "x", Statement: "y",
		Derivable: true, DerivedFromAxioms: []string{"AXIOM-1", "AXIOM-2"},
	}
	if err := merged.Validate(); err != nil {
		t.Errorf("merge-style axiom with parents rejected: %v", err)
	}
}

func TestForkPoint_Invariants(t *testing.T) {
	tests := []struct {
		name string
		fork ForkPoint
		ok   bool
	}{
		{
			name: "valid two-branch fork",
			fork: ForkPoint{
				ID: "FORK-1", BranchNodeID: "PROOF-X", Question: "?",
				Branches: []Branch{
					{ID: "A", Name: "minimal", Hypothesis: "no x", Burden: BurdenMinimal},
					{ID: "B", Name: "extended", Hypothesis: "x exists", Burden: BurdenExtended},
				},
			},
			ok: true,
		},
		{
			name: "rejects single branch",
			fork: ForkPoint{
				ID: "FORK-2", BranchNodeID: "PROOF-X", Question: "?",
				Branches: []Branch{
					{ID: "A", Name: "minimal", Hypothesis: "no x", Burden: BurdenMinimal},
				},
			},
			ok: false,
		},
		{
			name: "rejects no minimal",
			fork: ForkPoint{
				ID: "FORK-3", BranchNodeID: "PROOF-X", Question: "?",
				Branches: []Branch{
					{ID: "A", Name: "extended-1", Hypothesis: "x", Burden: BurdenExtended},
					{ID: "B", Name: "extended-2", Hypothesis: "y", Burden: BurdenExtended},
				},
			},
			ok: false,
		},
		{
			name: "rejects duplicate branch id",
			fork: ForkPoint{
				ID: "FORK-4", BranchNodeID: "PROOF-X", Question: "?",
				Branches: []Branch{
					{ID: "A", Name: "minimal", Hypothesis: "no x", Burden: BurdenMinimal},
					{ID: "A", Name: "extended", Hypothesis: "x", Burden: BurdenExtended},
				},
			},
			ok: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fork.Validate()
			if tt.ok && err != nil {
				t.Errorf("expected ok, got %v", err)
			}
			if !tt.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestStatusEnum_RoundTrip(t *testing.T) {
	values := []Status{
		StatusCoherent, StatusUntested, StatusIncoherent, StatusContested, StatusRefuted,
	}
	for _, v := range values {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %v: %v", v, err)
		}
		var got Status
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if got != v {
			t.Errorf("round-trip drift: %v -> %s -> %v", v, b, got)
		}
	}
}
