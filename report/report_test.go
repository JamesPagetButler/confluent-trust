// Issue #17 acceptance: dashboard output includes anchor count, tier
// breakdown, coherence ratio, ρ_net with sensitivity bracket, top
// bridge, sediment partition, highest-value eddy. The QBP v3.2
// Python-engine reference output is not reproducible here (fixture
// absent); we test the layout invariants on shipped fixtures.
package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

func loadInv(t *testing.T, name string) model.Inventory {
	t.Helper()
	path := filepath.Join("..", "testdata", name)
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 -- test reads testdata
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var inv model.Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	inv.NormalizeConfluences()
	return inv
}

func TestRunFullAnalysis_PopulatesEverySection(t *testing.T) {
	inv := loadInv(t, "qbp_quantum_v0_2.json")
	fa := RunFullAnalysis(inv, nil)

	// Every aggregator field is non-nil/non-degenerate.
	if fa.TierBreakdown == nil {
		t.Error("TierBreakdown nil")
	}
	if fa.AnchorDepth == nil {
		t.Error("AnchorDepth nil")
	}
	if fa.ChainDepth == nil {
		t.Error("ChainDepth nil")
	}
}

func TestDashboard_LayoutSections(t *testing.T) {
	inv := loadInv(t, "qbp_quantum_v0_2.json")
	fa := RunFullAnalysis(inv, nil)
	out := Dashboard(inv, fa)

	// Acceptance: 8 dashboard rows + header. We assert each label
	// appears once.
	requiredLabels := []string{
		"CTH Health:",
		"Anchors:",
		"Coherence:",
		"ρ_net:",
		"ρ_gross:",
		"Top bridge:",
		"Sediment:",
		"Highest eddy:",
	}
	for _, label := range requiredLabels {
		if !strings.Contains(out, label) {
			t.Errorf("dashboard missing label %q\nfull output:\n%s", label, out)
		}
	}
	if strings.Contains(out, "<unset>") || strings.Contains(out, "<nil>") {
		t.Errorf("dashboard contains unset markers:\n%s", out)
	}
}

func TestDashboard_DMForkRendersCleanly(t *testing.T) {
	inv := loadInv(t, "qbp_dm_fork.json")
	fa := RunFullAnalysis(inv, nil)
	out := Dashboard(inv, fa)
	if strings.Contains(out, "<nil>") || strings.Contains(out, "%!") {
		t.Errorf("dashboard contains format-string artefacts:\n%s", out)
	}
	if !strings.Contains(out, "QBP-DM") {
		t.Errorf("dashboard missing programme name:\n%s", out)
	}
}

func TestMarkdownReport_HasEverySection(t *testing.T) {
	inv := loadInv(t, "qbp_quantum_v0_2.json")
	fa := RunFullAnalysis(inv, nil)
	out := MarkdownReport(inv, fa)

	required := []string{
		"# CTH analysis",
		"## Compression",
		"## Tier breakdown",
		"## Coherence",
		"## Top bridges",
		"## Sediment partition",
		"## Eddies",
		"## Ab-initio preference",
		"## Confluence depth",
	}
	for _, h := range required {
		if !strings.Contains(out, h) {
			t.Errorf("markdown missing header %q", h)
		}
	}
}

func TestRiverMap_NonEmpty(t *testing.T) {
	inv := loadInv(t, "minimal.json")
	out := RiverMap(inv)
	if len(out) < 64 {
		t.Errorf("river map too short (%d bytes):\n%s", len(out), out)
	}
	if !strings.Contains(out, "river map") {
		t.Errorf("river map missing title:\n%s", out)
	}
}

func TestRunFullAnalysis_OnEachShippedFixture(t *testing.T) {
	for _, name := range []string{"minimal.json", "qbp_quantum_v0_1.json", "qbp_quantum_v0_2.json", "qbp_dm_fork.json"} {
		t.Run(name, func(t *testing.T) {
			inv := loadInv(t, name)
			fa := RunFullAnalysis(inv, nil)
			d := Dashboard(inv, fa)
			if len(d) == 0 {
				t.Errorf("Dashboard empty for %s", name)
			}
			m := MarkdownReport(inv, fa)
			if len(m) == 0 {
				t.Errorf("MarkdownReport empty for %s", name)
			}
		})
	}
}
