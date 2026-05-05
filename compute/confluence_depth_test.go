// Issue #12 acceptance: anchors downstream of 3-way confluences have
// higher depth than those downstream of 2-way confluences. Encoded in
// TestAnchorConfluenceDepth_Acceptance below.
package compute

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func TestAnchorConfluenceDepth_EmptyInventory(t *testing.T) {
	got := AnchorConfluenceDepth(model.Inventory{})
	if len(got) != 0 {
		t.Errorf("empty inventory: got %v, want empty", got)
	}
}

func TestAnchorConfluenceDepth_NoConfluencesIsZero(t *testing.T) {
	inv := model.Inventory{
		Anchors: []model.Anchor{{ID: "X"}, {ID: "Y"}},
		Chains: []model.Chain{
			{ID: testChainC1, SourceIDs: []string{"X"}, TargetID: "Y"},
		},
	}
	got := AnchorConfluenceDepth(inv)
	for id, d := range got {
		if d != 0 {
			t.Errorf("%s depth = %d, want 0 (no confluences)", id, d)
		}
	}
}

// TestAnchorConfluenceDepth_Acceptance encodes the issue's acceptance
// line: an anchor downstream of a 3-way confluence has greater depth
// than an anchor downstream of a 2-way confluence.
func TestAnchorConfluenceDepth_Acceptance(t *testing.T) {
	// Topology:
	//   X → Z       (2-way confluence at Y, then Z below)
	//   X → Z
	//   X → W
	//   X → W
	//   X → W       (3-way confluence at W, then V below)
	// Result: depth(Z) = (2-1) = 1; depth(V) = (3-1) = 2.
	inv := model.Inventory{
		Anchors: []model.Anchor{
			{ID: "X"}, {ID: "Y"}, {ID: "Z"}, {ID: "W"}, {ID: "V"},
		},
		Chains: []model.Chain{
			{ID: "Cy1", SourceIDs: []string{"X"}, TargetID: "Y"},
			{ID: "Cy2", SourceIDs: []string{"X"}, TargetID: "Y"},
			{ID: "Cz", SourceIDs: []string{"Y"}, TargetID: "Z"},
			{ID: "Cw1", SourceIDs: []string{"X"}, TargetID: "W"},
			{ID: "Cw2", SourceIDs: []string{"X"}, TargetID: "W"},
			{ID: "Cw3", SourceIDs: []string{"X"}, TargetID: "W"},
			{ID: "Cv", SourceIDs: []string{"W"}, TargetID: "V"},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{
				ID: "CP-Y", AnchorID: "Y",
				Paths: []model.ChainRef{{ChainID: "Cy1"}, {ChainID: "Cy2"}},
			},
			{
				ID: "CP-W", AnchorID: "W",
				Paths: []model.ChainRef{{ChainID: "Cw1"}, {ChainID: "Cw2"}, {ChainID: "Cw3"}},
			},
		},
	}

	got := AnchorConfluenceDepth(inv)

	if got["Z"] != 1 {
		t.Errorf("depth(Z) = %d, want 1 (downstream of 2-way confluence at Y)", got["Z"])
	}
	if got["V"] != 2 {
		t.Errorf("depth(V) = %d, want 2 (downstream of 3-way confluence at W)", got["V"])
	}
	if got["V"] <= got["Z"] {
		t.Errorf("acceptance: depth(V)=%d should exceed depth(Z)=%d (3-way > 2-way)",
			got["V"], got["Z"])
	}
}

func TestAnchorConfluenceDepth_AccumulatesAcrossUpstream(t *testing.T) {
	// Anchor V has two upstream confluences (one 3-way, one 2-way).
	// depth(V) = (3-1) + (2-1) = 3.
	inv := model.Inventory{
		Anchors: []model.Anchor{{ID: "P"}, {ID: "Q"}, {ID: "V"}},
		Chains: []model.Chain{
			{ID: "Cp1", SourceIDs: []string{"X"}, TargetID: "P"},
			{ID: "Cp2", SourceIDs: []string{"X"}, TargetID: "P"},
			{ID: "Cp3", SourceIDs: []string{"X"}, TargetID: "P"},
			{ID: "Cq1", SourceIDs: []string{"Y"}, TargetID: "Q"},
			{ID: "Cq2", SourceIDs: []string{"Y"}, TargetID: "Q"},
			{ID: "Cv", SourceIDs: []string{"P", "Q"}, TargetID: "V"},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{
				ID: "CP-P", AnchorID: "P",
				Paths: []model.ChainRef{
					{ChainID: "Cp1"}, {ChainID: "Cp2"}, {ChainID: "Cp3"},
				},
			},
			{
				ID: "CP-Q", AnchorID: "Q",
				Paths: []model.ChainRef{{ChainID: "Cq1"}, {ChainID: "Cq2"}},
			},
		},
	}

	got := AnchorConfluenceDepth(inv)
	if want := 3; got["V"] != want {
		t.Errorf("depth(V) = %d, want %d (accumulates 3-way + 2-way upstream)",
			got["V"], want)
	}
}

func TestChainConfluenceDepth_TracksTargetID(t *testing.T) {
	inv := model.Inventory{
		Anchors: []model.Anchor{{ID: "A"}, {ID: "B"}},
		Chains: []model.Chain{
			{ID: "ChainA", SourceIDs: []string{"A"}, TargetID: "A"},
			{ID: "ChainB", SourceIDs: []string{"A"}, TargetID: "B"},
			{ID: "ChainB2", SourceIDs: []string{"A"}, TargetID: "B"},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{
				ID: "CP-B", AnchorID: "B",
				Paths: []model.ChainRef{{ChainID: "ChainB"}, {ChainID: "ChainB2"}},
			},
		},
	}
	got := ChainConfluenceDepth(inv)
	if got["ChainB"] != 1 {
		t.Errorf("ChainB depth = %d, want 1 (target B is 2-way confluence)", got["ChainB"])
	}
	if got["ChainA"] != 0 {
		t.Errorf("ChainA depth = %d, want 0", got["ChainA"])
	}
}

func TestAnchorConfluenceDepth_QBPQuantumV02(t *testing.T) {
	inv := loadFixture(t, "qbp_quantum_v0_2.json")
	got := AnchorConfluenceDepth(inv)
	if len(got) != len(inv.Anchors) {
		t.Errorf("expected one entry per anchor, got %d entries for %d anchors",
			len(got), len(inv.Anchors))
	}
	for id, d := range got {
		if d < 0 {
			t.Errorf("anchor %s: depth %d negative", id, d)
		}
	}
}
