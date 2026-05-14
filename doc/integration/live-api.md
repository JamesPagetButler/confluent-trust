# LiveInventory — caller-side wiring sketch

**Relates to:** `store/live.go`, design doc `doc/design/live-inventory-api.md`, CTH #51, CTH #19, BMA #107, QBP #403, Contextus PR #11.

This document shows how the three primary consumers wire up `store.LiveInventory`.
All code below is pseudocode for orientation — it does not compile as-is.

---

## 1. BMA continuous-loop wiring (BMA #107)

BMA's L3 Beliefs layer holds a `*store.LiveInventory` open for the lifetime of the
cognitive cycle. Every NT_SIGNAL that produces a CTH anchor (per Wyrd PR #35 §4.1)
calls `AppendAnchor` or `UpdateAnchor`; status changes publish on the NATS topic
defined by CTH #19 (`cth.scoring.{anchor_id}.score_event`).

```go
// BMA-side pseudocode — internal/bma/beliefs/layer.go

nc, _ := nats.Connect(natsURL)

li, err := store.OpenLiveInventory("bma-data/inventory.json", &store.Hooks{
    OnAnchorChange: func(before, after *model.Anchor) {
        // Whitelist filtering already done by LiveInventory (design §2.1).
        // This hook fires only when {Status, MeasuredValue, MeasuredError,
        // DiscrepancyPct, LastTestedAt} change — never for Notes/Description.
        payload := buildScoreEvent(before, after) // BMA-internal proto
        topic := "cth.scoring." + after.ID + ".score_event"
        _ = nc.Publish(topic, payload)
    },
})
if err != nil {
    // BMA startup gate: inventory load failure is a fatal SE_HARDWARE_PROBE event.
    log.Fatal("beliefs layer: open inventory:", err)
}
defer li.Close()

// Cognitive cycle: M2 WDEvent → CTH ρ_net feedback loop (BMA #107).
for event := range wdEventCh {
    anchor := buildAnchorFromWDEvent(event) // maps NT_SIGNAL fields
    if err := li.AppendAnchor(anchor); err != nil {
        if errors.Is(err, model.ErrIDCollision) {
            // Already exists — update instead.
            _ = li.UpdateAnchor(anchor.ID, func(a *model.Anchor) error {
                a.Status = anchor.Status
                a.MeasuredValue = anchor.MeasuredValue
                return nil
            })
        }
    }
    // Snapshot for immediate ρ_net query (compute.NetCompressionDetail).
    snap := li.Snapshot()
    rhoNet := computeRhoNet(snap) // compute package
    publishToL3(rhoNet)
}
```

**Key points:**
- `OnAnchorChange` is the only hook BMA registers; chain/confluence hooks are nil.
- The hook publishes on `cth.scoring.{id}.score_event` per CTH #19 without any
  additional filtering — `LiveInventory` already applied the whitelist.
- `li.Snapshot()` is safe to call from the main loop after `UpdateAnchor` returns;
  the snapshot is always post-mutation consistent.

---

## 2. QBP federation-tenancy wiring (QBP #403 §5.3)

The QBP bookkeeper-write flow (`bma cth propose-update` / `bma cth status`) maps
to the first two bookkeeper ops (qbp-implementor toddle-design seq=14 Capability #7).
The bookkeeper holds a `*store.LiveInventory` opened on the QBP programme inventory;
`UpdateAnchor` is called when observation data arrives from the QBP experiment pipeline.

```go
// QBP bookkeeper pseudocode — qbp/bookkeeper/write.go

li, err := store.OpenLiveInventory("qbp-data/inventory.json", &store.Hooks{
    OnAnchorChange: func(before, after *model.Anchor) {
        // Publish a federation-wide reconciliation event so sibling tenants
        // can replay the PR7 reconciliation cycle.
        publishFederationEvent(after)
    },
})

// Op 1: bma cth status — read current anchor state.
snap := li.Snapshot()
printAnchorTable(snap.Anchors)

// Op 2: bma cth propose-update — update a prediction anchor with new measurement.
if err := li.UpdateAnchor("PRED-nv-center-decoherence", func(a *model.Anchor) error {
    measured := 42.7e-6 // seconds — hypothetical
    a.MeasuredValue = &measured
    a.Status = model.StatusCoherent
    a.MeasuredSource = "NV-Center run 2026-05-14"
    return nil
}); err != nil {
    return fmt.Errorf("bookkeeper: propose-update: %w", err)
}
// LiveInventory has fsynced to disk; the hook published the event.
// The PR7 reconciliation cycle reads the updated inventory via Snapshot().
```

**Key points:**
- The Walk-phase substrate swap (`OpenLiveInventoryWyrd`) is a one-line change
  per design §7; the `UpdateAnchor` call is identical. See CTH #60.
- `model.ErrNotFound` is the expected sentinel when an anchor has not yet been
  seeded; the bookkeeper creates it via `AppendAnchor` first.

---

## 3. Contextus closure-caching wiring (Contextus PR #11 v1.4)

Contextus PR #11 Spec v1.4 declares theories as conceptual scopes via
`ontology_uri: cth://anchor/<id>`. The `cth-derivation` membership predicate
computes the transitive closure of the derivation graph; when an anchor or chain
changes, the cached closure for that scope must be invalidated.

```go
// Contextus-side pseudocode — contextus/scope/cth_derivation.go

cache := newClosureCache() // TTL-on-mutate

li, err := store.OpenLiveInventory("shared-inventory.json", &store.Hooks{
    OnAnchorChange: func(before, after *model.Anchor) {
        // Any anchor status change invalidates the scope closure
        // for that anchor and all scopes whose closure includes it.
        cache.InvalidateTransitive(after.ID)
    },
    OnChainChange: func(before, after *model.Chain) {
        // Chain topology changes affect the derivation closure for every
        // anchor reachable via the modified chain — invalidate by chain ID.
        cache.InvalidateByChain(after.ID)
    },
    OnConfluenceChange: func(before, after *model.ConfluencePoint) {
        // Confluence membership determines "what counts as evidence";
        // invalidate the anchor scope this confluence converges on.
        cache.InvalidateTransitive(after.AnchorID)
    },
})

// Membership predicate: is anchor B in the derivation closure of scope A?
func IsMember(scopeAnchorID, candidateID string) bool {
    if closure, ok := cache.Get(scopeAnchorID); ok {
        return closure.Contains(candidateID)
    }
    snap := li.Snapshot()
    closure := computeClosure(snap, scopeAnchorID) // transitive chain walk
    cache.Set(scopeAnchorID, closure)
    return closure.Contains(candidateID)
}
```

**Key points:**
- `OnChainChange` and `OnConfluenceChange` fire on ALL field changes (design §2.1
  table row 2 and 3) — Contextus relies on this to catch topology changes that
  would otherwise silently stale the closure cache.
- `AppendAnchor` / `AppendChain` / `AppendConfluence` also fire hooks (before==nil)
  so new nodes are visible immediately, not just mutations to existing ones.

---

## 4. Walk-phase substrate swap (CTH #60)

Per design §7, at Walk the JSON file path swaps for a Wyrd hyperedge store.
The single-line change on the caller side:

```go
// Toddle (this PR)
li, err := store.OpenLiveInventory("bma-data/inventory.json", hooks)

// Walk (after CTH #60 lands the bridging adapter)
li, err := store.OpenLiveInventoryWyrd(graph, "BMA", hooks)
```

The returned `*store.LiveInventory` exposes the same `AppendAnchor` / `UpdateAnchor`
/ `Snapshot` / `Close` methods. All three wiring patterns above require no other
changes.

---

## 5. References

- `doc/design/live-inventory-api.md` §6 (NATS integration hook), §7 (Wyrd cutover)
- `store/live.go` — implementation
- CTH #19 — NATS topic `cth.scoring.{anchor_id}.score_event`
- BMA #107 — M2 WDEvent → CTH ρ_net feedback loop
- QBP #403 — Federation Tenancy v0.1 §5.3
- Contextus PR #11 — Spec v1.4 `cth-derivation` membership predicate
- CTH #60 — Wyrd↔CTH schema bridging (Walk-gate)
- Wyrd PR #35 §4.1 — CTHAnchor flow from NT_SIGNAL
