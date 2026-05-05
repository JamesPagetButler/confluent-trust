package report

import (
	"github.com/JamesPagetButler/confluent-trust/compute"
	"github.com/JamesPagetButler/confluent-trust/model"
)

// FullAnalysis aggregates every compute output the dashboard, markdown
// report, and CLI need from a single inventory pass. RunFullAnalysis
// produces this in one call.
type FullAnalysis struct {
	AnchorDepth       map[string]int
	TierBreakdown     map[model.Tier]int
	ChainDepth        map[string]int
	TopBridge         string
	HighestValueEddy  string
	AbInitio          []compute.AbInitioResult
	EddyRanking       []compute.EddyRanking
	BridgeNodes       []compute.BridgeNode
	Sediment          compute.SedimentReport
	Compression       compute.NetCompressionDetail
	SensitivityHalfH  float64
	SensitivityBaseH  float64
	SensitivityDouble float64
	GrossRho          float64
	NetRho            float64
	CoherenceRatio    float64
}

// RunFullAnalysis runs every primitive over inv and returns a single
// struct callers can render. Pass axiomEntropy=nil to use defaults.
func RunFullAnalysis(inv model.Inventory, axiomEntropy map[string]float64) FullAnalysis {
	netRho, detail := compute.NetCompression(inv, axiomEntropy)
	half, base, double := compute.SensitivityBracket(inv, axiomEntropy)

	bridges := compute.BridgeCentrality(inv, true)
	var topBridge string
	if len(bridges) > 0 {
		topBridge = bridges[0].ID
	}

	eddies := compute.RankEddies(inv)
	var topEddy string
	if len(eddies) > 0 {
		topEddy = eddies[0].InputID
	}

	return FullAnalysis{
		Compression:       detail,
		Sediment:          compute.DetectSedimentPartitions(inv),
		BridgeNodes:       bridges,
		EddyRanking:       eddies,
		AbInitio:          compute.AbInitioScore(inv),
		AnchorDepth:       compute.AnchorConfluenceDepth(inv),
		ChainDepth:        compute.ChainConfluenceDepth(inv),
		TierBreakdown:     tierBreakdown(inv),
		HighestValueEddy:  topEddy,
		TopBridge:         topBridge,
		SensitivityHalfH:  half,
		SensitivityBaseH:  base,
		SensitivityDouble: double,
		GrossRho:          detail.GrossRho,
		NetRho:            netRho,
		CoherenceRatio:    coherenceRatio(inv, axiomEntropy),
	}
}

func tierBreakdown(inv model.Inventory) map[model.Tier]int {
	out := map[model.Tier]int{
		model.TierAxiom:       len(inv.Axioms),
		model.TierProof:       0,
		model.TierMeasurement: 0,
		model.TierPrediction:  0,
	}
	for _, a := range inv.Anchors {
		out[a.Tier]++
	}
	return out
}

// coherenceRatio replicates the local compute package definition (which
// is an unexported helper there) so report/ can stay independent of any
// internal compute structure.
func coherenceRatio(inv model.Inventory, axiomEntropy map[string]float64) float64 {
	var total, incoherent float64
	for _, a := range inv.Axioms {
		total += compute.AxiomEntropy(a, axiomEntropy)
	}
	for _, a := range inv.Anchors {
		eta := compute.ResidualEntropy(a, axiomEntropy)
		total += eta
		if a.Status == model.StatusIncoherent {
			incoherent += eta
		}
	}
	if total <= 0 {
		return 1.0
	}
	return 1.0 - incoherent/total
}
