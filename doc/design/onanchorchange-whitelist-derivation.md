# OnAnchorChange Whitelist Derivation from NetCompressionDetail

**Status:** Sprint-1-closeout-2026-05-17 seq=12 — Notary-bootstrap target #2

**Issue:** CTH #50 — derive `LiveInventory.Hooks.OnAnchorChange` whitelist from
`compute.NetCompressionDetail`'s field-dependency graph.

**Conclusion:** DIVERGENCE — see §5.

---

## 1. Objective

`LiveInventory.Hooks.OnAnchorChange` fires on `UpdateAnchor` only when one of a
fixed set of anchor fields changes (design `live-inventory-api.md` §2.1 v0.2).
The v0.2 design asserts the whitelist is the "ρ_net-affecting subset per Theory
v0.2 §4.1" but does not derive it from the code.

This document replaces assertion-correctness with derived-correctness: walk
`compute.NetCompressionDetail`'s transitive call graph, enumerate every
`model.Anchor` field read, take the union, and compare to the current whitelist.

Any future addition of a new `model.Anchor` field-read inside the
`NetCompression` call graph must update both this document and
`anchorStatusFieldsChanged` in `store/live.go`.

---

## 2. Method

1. Start at `compute.NetCompression` (the function that returns
   `NetCompressionDetail`).
2. For each function in the transitive call graph that accepts a `model.Anchor`
   value or pointer, list every field read (`a.<Field>`).
3. Union across all callees → derived whitelist.
4. Compare to the current whitelist in `store.anchorStatusFieldsChanged`.

Scope: only functions reachable from `compute.NetCompression`. Functions
that take `model.Inventory` but never dereference an individual `model.Anchor`
field (e.g. `InformationDeficit`, `AxiomEntropySum`) are noted as "no Anchor
field reads".

Out of scope: `compute.ResidualEntropy`, `compute.ChainFidelity`,
`compute.BridgeCentrality`, `compute.AnchorConfluenceDepth`,
`compute.AbInitioScore`, `compute.RankEddies` — these are not called from
`NetCompression`.

---

## 3. Field-read inventory

### 3.1 `compute.NetCompression` (compression.go)

```
func NetCompression(inv model.Inventory, axiomEntropy map[string]float64) (float64, NetCompressionDetail)
```

Direct Anchor field reads inside `NetCompression`'s own body:

| Field | Location | Purpose |
|---|---|---|
| `a.PredictionChain` | Pass 1 + Pass 2 (`for _, dep := range a.PredictionChain`) | Counts consumers per input; allocates input-entropy cost per anchor |
| `a.ID` | `row.AnchorID = a.ID` | Identifies the per-anchor breakdown row |

`a.ID` is the anchor's identity key and is not mutable after creation; it does
not appear in the whitelist (changing it via `UpdateAnchor` would be an identity
swap, not a ρ_net-affecting mutation).

### 3.2 `compute.confirmedAnchors` (compression.go)

```
func confirmedAnchors(inv model.Inventory) []model.Anchor
```

Called by `NetCompression` as the first step.

| Field | Read | Purpose |
|---|---|---|
| `a.Tier` | `a.Tier == model.TierMeasurement` | Gates which anchors enter the confirmed set |
| `a.Status` | `a.Status == model.StatusCoherent` | Gates which anchors enter the confirmed set |

### 3.3 `compute.ConfirmatoryInfo` (entropy.go)

```
func ConfirmatoryInfo(a model.Anchor) float64
```

Called by `NetCompression` in Pass 2 for every confirmed anchor.

| Field | Read | Purpose |
|---|---|---|
| `a.Tier` | switch `a.Tier` (TierAxiom, TierProof, TierMeasurement, TierPrediction) | Selects the formula branch |
| `a.DiscrepancyPct` | `*a.DiscrepancyPct` | The only numeric input to the ι(v) formula |

### 3.4 `compute.AxiomEntropySum` (compression.go)

```
func AxiomEntropySum(inv model.Inventory, axiomEntropy map[string]float64) float64
```

Operates on `inv.Axioms` (type `model.Axiom`), not on `model.Anchor`. No Anchor
field reads.

### 3.5 `compute.AxiomEntropy` (entropy.go)

```
func AxiomEntropy(a model.Axiom, assigned map[string]float64) float64
```

Operates on `model.Axiom`, not `model.Anchor`. No Anchor field reads.

### 3.6 `compute.InformationDeficit` (compression.go)

```
func InformationDeficit(inv model.Inventory) float64
```

Operates on `inv.Inputs` (type `model.Input`). No Anchor field reads.

### 3.7 `compute.InputEntropy` (entropy.go)

```
func InputEntropy(sigFigures int) float64
```

Pure arithmetic. No Anchor field reads.

### 3.8 `compute.inputIDs` (compression.go)

```
func inputIDs(inv model.Inventory) map[string]struct{}
```

Operates on `inv.Inputs`. No Anchor field reads.

---

## 4. Derived field set

Union of all `model.Anchor` fields read by `NetCompression`'s transitive call
graph (excluding the immutable `a.ID`):

| Field | Read by | Whitelist-relevant? |
|---|---|---|
| `Tier` | `confirmedAnchors`, `ConfirmatoryInfo` | YES — gates both confirmed-set membership and formula branch |
| `Status` | `confirmedAnchors` | YES — gates confirmed-set membership |
| `DiscrepancyPct` | `ConfirmatoryInfo` | YES — sole numeric input to ι(v) formula |
| `PredictionChain` | `NetCompression` Pass 1 + Pass 2 | YES — controls which inputs are charged; changes input-cost allocation |

**Derived whitelist = `{Tier, Status, DiscrepancyPct, PredictionChain}`**

---

## 5. Comparison to current whitelist

Current implementation in `store.anchorStatusFieldsChanged`:

| Field | In current whitelist | In derived set | Verdict |
|---|---|---|---|
| `Status` | YES | YES | Match |
| `MeasuredValue` | YES | NO | Current-only: not directly read by NetCompression |
| `MeasuredError` | YES | NO | Current-only: not directly read by NetCompression |
| `DiscrepancyPct` | YES | YES | Match |
| `LastTestedAt` | YES | NO | Current-only: pure metadata, no math path |
| `Tier` | NO | YES | Derived-only: gates confirmedAnchors + ConfirmatoryInfo branch |
| `PredictionChain` | NO | YES | Derived-only: controls input-cost allocation in both passes |

**Result: DIVERGENCE**

Fields present in current whitelist but absent from derived set:
- `MeasuredValue` — not read by any function in NetCompression's call graph.
  Rationale for its current presence: `MeasuredValue` is a measurement result
  field that callers compute `DiscrepancyPct` from (via `compute.ScorePrediction`)
  before writing. Its change implies `DiscrepancyPct` will likely be updated in
  the same or a subsequent call. Per the heuristic in the implementation spec
  ("lean toward more inclusive whitelist"), this field is **retained** in the
  updated whitelist.
- `MeasuredError` — same rationale as `MeasuredValue`. Retained.
- `LastTestedAt` — purely temporal metadata; does not feed any ρ_net computation.
  No code path from `LastTestedAt` to `NetCompression` output exists. Per the
  heuristic, if ambiguous lean inclusive; however this is unambiguously NOT on
  any math path. **Dropped** from the updated whitelist.

Fields present in derived set but absent from current whitelist:
- `Tier` — changes to `Tier` alter confirmed-set membership
  (`confirmedAnchors` checks `a.Tier == TierMeasurement`) and select a different
  formula branch in `ConfirmatoryInfo`. In practice `Tier` is set at anchor
  creation and rarely changes, but the hook contract is "fire when ρ_net could
  change"; a Tier change is a Tier-0/1/2/3 reclassification. **Added** to the
  updated whitelist.
- `PredictionChain` — directly iterated in both passes of `NetCompression`.
  Adding or removing an input dependency changes input-cost allocation and
  therefore ρ_net. **Added** to the updated whitelist.

### 5.1 Updated whitelist

Applying the analysis above (retain ambiguous fields; drop unambiguous
non-math-path fields; add derived-only fields):

```
{Status, MeasuredValue, MeasuredError, DiscrepancyPct, Tier, PredictionChain}
```

`LastTestedAt` is removed. `Tier` and `PredictionChain` are added.

---

## 6. Conclusion

The derivation produces a DIVERGENCE:

1. `LastTestedAt` was in the current whitelist but has no path to any
   NetCompression computation. It is removed.
2. `Tier` and `PredictionChain` were absent from the current whitelist but are
   directly read by NetCompression's call graph. They are added.
3. `MeasuredValue` and `MeasuredError` are retained under the inclusive heuristic:
   they are upstream of `DiscrepancyPct` in the typical write workflow.

Updated `anchorStatusFieldsChanged` in `store/live.go` reflects these changes.

`store/live_whitelist_test.go` provides the regression test:
- Each whitelist field mutated alone → hook fires.
- `Notes` and `Description` (known non-whitelist) mutated alone → hook does NOT fire.
- `LastTestedAt` mutated alone → hook does NOT fire (confirmed removal).

---

## 7. Future maintenance rule

If any PR modifies `compute.NetCompression`, `compute.confirmedAnchors`,
`compute.ConfirmatoryInfo`, or adds a new helper called from them that reads a
new `model.Anchor` field, that PR must:

1. Update §3 (field-read inventory) of this document.
2. Update §4 (derived field set).
3. Update §5 (comparison table).
4. Update `anchorStatusFieldsChanged` in `store/live.go`.
5. Update `TestOnAnchorChange_WhitelistDerivation` in
   `store/live_whitelist_test.go`.
