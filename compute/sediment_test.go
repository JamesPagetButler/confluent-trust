// Issue #11 acceptance: REBCO identified as dirty-only domain. Sharp
// partition detected. The QBP v3.2 fixture is absent from testdata/
// pending Issue #11-followup; the structural assertions below test the
// partition algebra on the fixtures we have, plus a hand-built two-chain
// inventory that encodes the REBCO acceptance pattern (one clean-only
// domain coexisting with one dirty-only domain).
package compute

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Hand-built domain labels for the sediment acceptance test. Each appears
// in three or more spots across this file (struct literal, CleanOnly
// assertion, DirtyOnly assertion, DomainCounts lookup) so they trip
// goconst as raw strings — hoist to constants here.
const (
	sedDomainClean = "ALG"
	sedDomainDirty = "REBCO"
)

func TestDetectSedimentPartitions_v0_2Fixture(t *testing.T) {
	// Structural property: every chain in the fixture appears in exactly
	// one partition, and the four partitions sum to len(inv.Chains).
	inv := loadInventory(t, "qbp_quantum_v0_2.json")

	report := DetectSedimentPartitions(inv)
	total := len(report.Laminar.ChainIDs) +
		len(report.LowSediment.ChainIDs) +
		len(report.Moderate.ChainIDs) +
		len(report.Heavy.ChainIDs)
	if total != len(inv.Chains) {
		t.Fatalf("partition total %d != chain count %d", total, len(inv.Chains))
	}

	// No chain id appears in two partitions.
	seen := make(map[string]string, len(inv.Chains))
	for _, p := range []SedimentPartition{report.Laminar, report.LowSediment, report.Moderate, report.Heavy} {
		for _, id := range p.ChainIDs {
			if prev, dup := seen[id]; dup {
				t.Errorf("chain %s in both %s and %s", id, prev, p.Regime)
			}
			seen[id] = p.Regime
		}
	}

	// Per-partition DomainCount totals must equal that partition's chain
	// count — every chain credits exactly one domain.
	for _, p := range []SedimentPartition{report.Laminar, report.LowSediment, report.Moderate, report.Heavy} {
		var sum int
		for _, n := range p.DomainCounts {
			sum += n
		}
		if sum != len(p.ChainIDs) {
			t.Errorf("%s: domain count sum %d != chain count %d",
				p.Regime, sum, len(p.ChainIDs))
		}
	}

	// SharpPartition is a boolean — exercise the field, no assertion on
	// its truth value for this fixture (the canonical REBCO acceptance
	// is encoded in TestDetectSedimentPartitions_AcceptanceREBCO below).
	_ = report.SharpPartition
}

// TestDetectSedimentPartitions_AcceptanceREBCO encodes the issue #11
// acceptance line: REBCO identified as dirty-only domain, sharp
// partition detected. Built inline because qbp_v3_2.json (where REBCO
// originates) is intentionally absent from testdata/.
func TestDetectSedimentPartitions_AcceptanceREBCO(t *testing.T) {
	clean := 1.0
	dirty := 0.5

	inv := model.Inventory{
		Chains: []model.Chain{
			{
				ID: "C-clean", TargetID: sedDomainClean + "-su2",
				SourceIDs: []string{testAxiomID}, Fidelity: &clean,
			},
			{
				ID: "C-dirty", TargetID: sedDomainDirty + "-tape",
				SourceIDs: []string{testAxiomID}, Fidelity: &dirty,
			},
		},
	}

	report := DetectSedimentPartitions(inv)

	if !report.SharpPartition {
		t.Fatalf("expected SharpPartition=true for clean+dirty mix, got false")
	}
	if !slices.Contains(report.CleanOnlyDomains, sedDomainClean) {
		t.Errorf("CleanOnlyDomains = %v; want to contain %q",
			report.CleanOnlyDomains, sedDomainClean)
	}
	if !slices.Contains(report.DirtyOnlyDomains, sedDomainDirty) {
		t.Errorf("DirtyOnlyDomains = %v; want to contain %q (acceptance)",
			report.DirtyOnlyDomains, sedDomainDirty)
	}

	if n := len(report.Laminar.ChainIDs); n != 1 || report.Laminar.ChainIDs[0] != "C-clean" {
		t.Errorf("laminar partition = %v, want [C-clean]", report.Laminar.ChainIDs)
	}
	if n := len(report.Heavy.ChainIDs); n != 1 || report.Heavy.ChainIDs[0] != "C-dirty" {
		t.Errorf("heavy partition = %v, want [C-dirty]", report.Heavy.ChainIDs)
	}
	if report.Laminar.DomainCounts[sedDomainClean] != 1 {
		t.Errorf("laminar DomainCounts[%q] = %d, want 1",
			sedDomainClean, report.Laminar.DomainCounts[sedDomainClean])
	}
	if report.Heavy.DomainCounts[sedDomainDirty] != 1 {
		t.Errorf("heavy DomainCounts[%q] = %d, want 1",
			sedDomainDirty, report.Heavy.DomainCounts[sedDomainDirty])
	}
}

// TestDetectSedimentPartitions_NoSharpWhenAllClean: a sanity check that
// SharpPartition does not fire when the whole inventory sits in one half.
func TestDetectSedimentPartitions_NoSharpWhenAllClean(t *testing.T) {
	clean := 1.0
	inv := model.Inventory{
		Chains: []model.Chain{
			{
				ID: testChainC1, TargetID: sedDomainClean + "-a",
				SourceIDs: []string{testAxiomID}, Fidelity: &clean,
			},
			{
				ID: testChainC2, TargetID: sedDomainClean + "-b",
				SourceIDs: []string{testAxiomID}, Fidelity: &clean,
			},
		},
	}
	report := DetectSedimentPartitions(inv)
	if report.SharpPartition {
		t.Errorf("SharpPartition fired for all-clean inventory")
	}
	if len(report.DirtyOnlyDomains) != 0 {
		t.Errorf("DirtyOnlyDomains = %v, want empty", report.DirtyOnlyDomains)
	}
}

// loadInventory reads a fixture JSON file into a model.Inventory. We
// don't use store.LoadInventory because it would introduce a test-time
// dependency from compute/ on store/.
func loadInventory(t *testing.T, name string) model.Inventory {
	t.Helper()
	path := filepath.Join("..", "testdata", name)
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var inv model.Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return inv
}
