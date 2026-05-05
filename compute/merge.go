package compute

import (
	"sort"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// BridgeEdge is one entry in MergeReport.Bridges: a domain boundary
// connecting a shared anchor to a programme-specific anchor in either
// merged side. Theory v0.2 §5.1.
type BridgeEdge struct {
	AnchorID string
	FromSide string
	ToSide   string
	Fidelity float64
}

// MergeReport summarises a programme merge. Per Theorem 2 the merge is
// "lossless" when every shared anchor is Tier 1 in both inputs (the
// bridge fidelity is 1.0 and no new entropy is introduced). DeficitKind
// classifies the merged residual deficit per Definition 19.
type MergeReport struct {
	SharedAnchorIDs    []string
	BridgeEdges        []BridgeEdge
	IncoherentMerges   []string
	TheoreticalDeficit float64
	EngineeringDeficit float64
	Lossless           bool
}

// MergeProgrammes combines two CTH inventories per Theory v0.2 §5.
// Shared anchors (same ID in both) are reduced via the §5.2 rules:
// Tier = min, Status = consensus (incoherent on disagreement), residual
// entropy = min. Bridge edges record cross-side connections; bridge
// fidelity is 1.0 when the shared anchor is Tier 1 in both programmes,
// otherwise downgraded by the lower-tier side's verification level.
//
// The merged Inventory uses Programme = "<a.Programme>+<b.Programme>"
// and SchemaVersion = "v0.2".
func MergeProgrammes(a, b model.Inventory) (model.Inventory, MergeReport) {
	merged := model.Inventory{
		Programme:        a.Programme + "+" + b.Programme,
		Version:          "merged",
		SchemaVersion:    "v0.2",
		ParentProgrammes: []string{a.Programme, b.Programme},
	}
	report := MergeReport{Lossless: true}

	merged.Axioms = mergeAxioms(a.Axioms, b.Axioms)
	merged.DerivedPrinciples = mergeDerivedPrinciples(a.DerivedPrinciples, b.DerivedPrinciples)

	mergedAnchors, sharedIDs, incoherent, allTier1Shared := mergeAnchors(a.Anchors, b.Anchors)
	merged.Anchors = mergedAnchors
	report.SharedAnchorIDs = sharedIDs
	report.IncoherentMerges = incoherent
	if !allTier1Shared {
		report.Lossless = false
	}
	if len(incoherent) > 0 {
		report.Lossless = false
	}

	merged.Inputs = mergeInputs(a.Inputs, b.Inputs)
	merged.Chains = append(append([]model.Chain{}, a.Chains...), b.Chains...)
	merged.ConfluencePoints = append(append([]model.ConfluencePoint{}, a.ConfluencePoints...), b.ConfluencePoints...)

	// Bridge edges: every shared anchor gets one bridge per side it has
	// neighbours on. Fidelity is 1.0 when the merged tier is TierProof
	// (Tier 1) on both sides; otherwise inherit the lower-tier side's
	// domain-boundary fidelity if any DomainBoundary at this anchor
	// names the appropriate cross-domain crossing, defaulting to 0.95.
	report.BridgeEdges = makeBridgeEdges(a, b, sharedIDs)
	for _, br := range report.BridgeEdges {
		if br.Fidelity < 1.0 {
			report.Lossless = false
		}
	}

	report.TheoreticalDeficit, report.EngineeringDeficit = classifyDeficit(merged)

	return merged, report
}

// mergeAxioms takes the union by ID. When an axiom is in both, A's row
// wins (deterministic; semantic differences should be rare).
func mergeAxioms(aAx, bAx []model.Axiom) []model.Axiom {
	seen := make(map[string]struct{}, len(aAx)+len(bAx))
	out := make([]model.Axiom, 0, len(aAx)+len(bAx))
	for _, x := range aAx {
		if _, dup := seen[x.ID]; dup {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
	}
	for _, x := range bAx {
		if _, dup := seen[x.ID]; dup {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
	}
	return out
}

func mergeDerivedPrinciples(aDP, bDP []model.DerivedPrinciple) []model.DerivedPrinciple {
	seen := make(map[string]struct{}, len(aDP)+len(bDP))
	out := make([]model.DerivedPrinciple, 0, len(aDP)+len(bDP))
	for _, x := range aDP {
		if _, dup := seen[x.ID]; dup {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
	}
	for _, x := range bDP {
		if _, dup := seen[x.ID]; dup {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
	}
	return out
}

// mergeAnchors applies the §5.2 reduction rules and reports which IDs
// were shared, which produced status conflicts, and whether every
// shared anchor was Tier 1 on both sides (Theorem 2 lossless precondition).
func mergeAnchors(aAnchors, bAnchors []model.Anchor) (
	merged []model.Anchor,
	sharedIDs []string,
	incoherent []string,
	allTier1Shared bool,
) {
	bByID := make(map[string]model.Anchor, len(bAnchors))
	for _, x := range bAnchors {
		bByID[x.ID] = x
	}
	allTier1Shared = true

	seen := make(map[string]struct{}, len(aAnchors)+len(bAnchors))
	merged = make([]model.Anchor, 0, len(aAnchors)+len(bAnchors))

	for _, a := range aAnchors {
		seen[a.ID] = struct{}{}
		if b, share := bByID[a.ID]; share {
			sharedIDs = append(sharedIDs, a.ID)
			r := mergeOneAnchor(a, b)
			if a.Status == model.StatusCoherent && b.Status == model.StatusCoherent &&
				r.Status == model.StatusIncoherent {
				incoherent = append(incoherent, a.ID)
			} else if a.Status != b.Status &&
				(a.Status == model.StatusIncoherent || b.Status == model.StatusIncoherent) {
				incoherent = append(incoherent, a.ID)
			}
			if a.Tier != model.TierProof || b.Tier != model.TierProof {
				allTier1Shared = false
			}
			merged = append(merged, r)
		} else {
			merged = append(merged, a)
		}
	}
	for _, b := range bAnchors {
		if _, dup := seen[b.ID]; dup {
			continue
		}
		merged = append(merged, b)
	}

	if len(sharedIDs) == 0 {
		// "All shared are Tier 1" is vacuously true with no shared
		// anchors; preserve allTier1Shared = true.
		_ = allTier1Shared
	}
	sort.Strings(sharedIDs)
	sort.Strings(incoherent)
	return merged, sharedIDs, incoherent, allTier1Shared
}

// mergeOneAnchor applies the §5.2 reduction to a single shared anchor.
func mergeOneAnchor(a, b model.Anchor) model.Anchor {
	// Tier = min (most trustworthy = lowest tier number, since
	// TierAxiom = 0 < TierProof = 1 < ... we follow the spec
	// language "more trustworthy" = lower number = earlier tier).
	tier := a.Tier
	if b.Tier < tier {
		tier = b.Tier
	}
	// Status: agreement or upgrade-to-incoherent on disagreement (when
	// either side declared incoherent).
	var status model.Status
	if a.Status == b.Status {
		status = a.Status
	} else {
		switch {
		case a.Status == model.StatusIncoherent || b.Status == model.StatusIncoherent:
			status = model.StatusIncoherent
		case a.Status == model.StatusUntested:
			status = b.Status
		case b.Status == model.StatusUntested:
			status = a.Status
		default:
			// Different but neither is incoherent: take A's by stable choice.
			status = a.Status
		}
	}
	// Residual entropy = min via DiscrepancyPct comparison: closer to 0
	// means smaller residual.
	out := a
	out.Tier = tier
	out.Status = status
	if a.DiscrepancyPct != nil && b.DiscrepancyPct != nil {
		if absLE(*b.DiscrepancyPct, *a.DiscrepancyPct) {
			out.DiscrepancyPct = b.DiscrepancyPct
		}
	} else if b.DiscrepancyPct != nil {
		out.DiscrepancyPct = b.DiscrepancyPct
	}
	if b.Description != "" && a.Description == "" {
		out.Description = b.Description
	}
	return out
}

func absLE(x, y float64) bool {
	ax, ay := x, y
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	return ax <= ay
}

func mergeInputs(aIn, bIn []model.Input) []model.Input {
	seen := make(map[string]struct{}, len(aIn)+len(bIn))
	out := make([]model.Input, 0, len(aIn)+len(bIn))
	for _, x := range aIn {
		if _, dup := seen[x.ID]; dup {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
	}
	for _, x := range bIn {
		if _, dup := seen[x.ID]; dup {
			continue
		}
		seen[x.ID] = struct{}{}
		out = append(out, x)
	}
	return out
}

// makeBridgeEdges builds one BridgeEdge per shared anchor for each side
// that retains it as a connection point. Per Theorem 2, fidelity is 1.0
// when both sides hold the anchor at TierProof; otherwise we look for a
// DomainBoundary record at this anchor in either inventory's chains and
// inherit its fidelity, defaulting to 0.95 (the §4.4 domain-boundary
// midpoint) when no record is present.
func makeBridgeEdges(a, b model.Inventory, sharedIDs []string) []BridgeEdge {
	if len(sharedIDs) == 0 {
		return nil
	}
	aTier := tierByID(a.Anchors)
	bTier := tierByID(b.Anchors)
	bridges := make([]BridgeEdge, 0, len(sharedIDs))
	for _, id := range sharedIDs {
		bridge := BridgeEdge{
			AnchorID: id,
			FromSide: a.Programme,
			ToSide:   b.Programme,
			Fidelity: 1.0,
		}
		if aTier[id] != model.TierProof || bTier[id] != model.TierProof {
			bridge.Fidelity = boundaryFidelityAt(id, a, b)
		}
		bridges = append(bridges, bridge)
	}
	return bridges
}

func tierByID(anchors []model.Anchor) map[string]model.Tier {
	out := make(map[string]model.Tier, len(anchors))
	for _, a := range anchors {
		out[a.ID] = a.Tier
	}
	return out
}

// boundaryFidelityAt returns the lowest DomainBoundary fidelity at id
// across both inventories' chain metadata, defaulting to 0.95 when no
// chain records a domain boundary at this anchor.
func boundaryFidelityAt(id string, a, b model.Inventory) float64 {
	const defaultBoundary = 0.95
	best := defaultBoundary
	for _, inv := range []model.Inventory{a, b} {
		for _, c := range inv.Chains {
			for _, db := range c.DomainBoundaries {
				if db.AtAnchorID != id {
					continue
				}
				if db.Fidelity < best {
					best = db.Fidelity
				}
			}
		}
	}
	return best
}

// classifyDeficit splits the merged deficit per Definition 19. Inputs
// with status "irreducible" or "unmeasurable" are theoretical deficit;
// "measurable" inputs are engineering deficit; unknown statuses fall
// back to engineering.
func classifyDeficit(inv model.Inventory) (theoretical, engineering float64) {
	for _, in := range inv.Inputs {
		bits := InputEntropy(in.SignificantFigures)
		switch in.Status {
		case "irreducible", "unmeasurable":
			theoretical += bits
		default:
			engineering += bits
		}
	}
	return theoretical, engineering
}
