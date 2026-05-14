# Live Inventory Update API — design surface (CTH #51)

**Status:** §I4 design surface (Loop 1 Reference). Implementation deferred to a follow-up PR after §I4 sign-off.

**Revision:** v0.2 — adds §2.1 *Hook semantics — Append fires + per-kind filter* resolving the T8 + T9 design clarifications flagged by @contextus-impl §I4 review on PR #62. Per beekeeper direction 2026-05-14: design-surface gaps resolve in a follow-up design PR, not the impl PR. See `## Changelog` at the bottom.

**Issue:** [#51](https://github.com/JamesPagetButler/confluent-trust/issues/51) — feat: live inventory update API — append/mutate anchors as theory advances

## 0. §I4 invariant — design-doc-as-S-01-review-surface

This document is the §I4 review surface per the federation's design-doc-then-impl pattern (precedent: Wyrd PR #35 `scoutquery.md`, Wyrd PR #39 `tier-immunity-salience.md`). The implementation PR ships only after §I4 sign-off lands here.

**Named reviewers (per D5 reviewer-list pattern):**

- `@bma-implementor` — **primary consumer**. BMA's continuous-loop cognitive cycle is the main caller; `AppendAnchor` is invoked from BMA's L3 Beliefs layer per BMA #107 (M2 WDEvent → CTH ρ_net feedback loop).
- `@qbp-implementor` — **secondary consumer**. Federation tenancy v0.1 (QBP #403) §5.3 specifies the live-update flow; PR7 reconciliation cycles consume this API.
- `@contextus-impl` — **consumer (closure caching)**. Contextus PR #11 Spec v1.4 theory-as-scope's `cth-derivation` membership predicate depends on TTL-on-mutate live events from this API.
- `@wyrd-implementor` — **substrate-cutover consultative**. At Walk, the underlying substrate swaps from JSON files to Wyrd hyperedges (per CTH #60 schema bridging); the API contract must remain substrate-agnostic.
- `@beekeeper` — review & approve.

## 1. Motivation

CTH v0.1.0 ships `store.LoadInventory` and `store.SaveInventory` for one-shot read/write. The workflow that produced QBP's `testdata/qbp_v3_2.json` is *"edit JSON file by hand, then run `cth analyse`"*. This is acceptable for human-curated programme inventories but breaks down under three concurrent forcing functions:

1. **BMA continuous-loop at Toddle** — workspace-phase-architecture §2.4 names CTH as L3 Beliefs substrate. BMA's continuous cognitive cycle produces theory artifacts continuously; each NT_SIGNAL with a CTHAnchor (per Wyrd PR #35 §4.1) becomes a new or updated anchor in the CTH inventory. Reload-and-save-whole-file cannot sustain 7-day endurance.
2. **Federation tenancy live-update flow** — QBP Federation Tenancy v0.1 §5.3 (PR #403) specifies that tenant predictions write back to the inventory as observations arrive. This is the bookkeeper-role write surface per qbp-implementor's first-2-bookkeeper-ops scope (`bma cth propose-update` / commit-on-approval).
3. **CTH-derivation membership invalidation** — Contextus PR #11 v1.4 declares theories as conceptual scopes via `ontology_uri: cth://anchor/<id>`. The CTH-derivation-closure membership recomputes when anchors change; this requires a TTL-on-mutate event surface from CTH.

The §0.11 principle (*"Trust anchors are CTH anchors. η = CTH ρ_net"*) means ρ_net must be queryable continuously, not just at offline-analysis time.

## 2. API surface

New file: `store/live.go`. New type:

```go
package store

import (
    "sync"
    "github.com/JamesPagetButler/confluent-trust/model"
)

// LiveInventory wraps a model.Inventory with concurrent append/mutate
// operations and atomic persistence. It is the long-lived handle that
// BMA, QBP, and other federation tenants hold across continuous-loop
// cycles.
//
// LiveInventory is safe for concurrent use by multiple goroutines.
// Snapshot() acquires a read lock; append/update methods acquire a
// write lock; the lock window covers validation, in-memory mutation,
// and disk persistence.
type LiveInventory struct {
    inv   *model.Inventory  // protected by mu
    path  string            // JSON file path (Toddle); ignored at Walk-cutover
    mu    sync.RWMutex
    hooks *Hooks            // optional change callbacks; see §6
}

// Hooks carry caller-supplied callbacks fired after a successful commit.
// All hooks are invoked outside the critical section (post-fsync, after
// the write lock is released) so callers do not deadlock by re-entering
// LiveInventory from a hook. Hooks must not panic; a panicking hook is
// logged and recovered, never propagated.
//
// The NATS event publish surface (cth.scoring.{anchor_id}.score_event
// per CTH #19) is wired by callers via these hooks; LiveInventory itself
// has no NATS dependency.
type Hooks struct {
    OnAnchorChange     func(before *model.Anchor, after *model.Anchor)
    OnChainChange      func(before *model.Chain, after *model.Chain)
    OnConfluenceChange func(before *model.ConfluencePoint, after *model.ConfluencePoint)
}

// OpenLiveInventory loads an inventory from disk and returns a handle
// suitable for live append/mutate. The file is loaded with full schema
// validation (same path as store.LoadInventory). The hooks argument may
// be nil (no callbacks).
//
// At Walk, this constructor is shadowed by OpenLiveInventoryWyrd; see §7.
func OpenLiveInventory(path string, hooks *Hooks) (*LiveInventory, error)

// AppendAnchor adds a new anchor. The anchor.ID must not collide with
// any existing anchor or axiom; on collision the method returns
// model.ErrIDCollision wrapped. The single anchor is schema-validated
// before insertion; the full Inventory.Validate() runs before fsync.
//
// On successful commit, Hooks.OnAnchorChange fires with before == nil
// and after pointing at the appended anchor (deep copy). See §2.1 for
// the full hook-semantics contract.
func (li *LiveInventory) AppendAnchor(a model.Anchor) error

// AppendChain, AppendConfluence, AppendInput follow the same contract.
func (li *LiveInventory) AppendChain(c model.Chain) error
func (li *LiveInventory) AppendConfluence(cp model.ConfluencePoint) error
func (li *LiveInventory) AppendInput(in model.Input) error

// UpdateAnchor applies a caller-supplied mutator to the anchor identified
// by id. The mutator runs inside the write lock; it must not call other
// LiveInventory methods (deadlock). Mutator errors abort the update; the
// in-memory state is rolled back; disk state is unchanged.
//
// If the mutator changes any of {Status, MeasuredValue, MeasuredError,
// DiscrepancyPct, LastTestedAt}, Hooks.OnAnchorChange fires with the
// before/after snapshot. Other field changes do not fire the hook
// (avoids spurious NATS publishes on routine bookkeeping). The
// anchor-specific whitelist is NOT applied to OnChainChange or
// OnConfluenceChange; see §2.1.
func (li *LiveInventory) UpdateAnchor(id string, mutator func(*model.Anchor) error) error

// UpdateChain has the same shape as UpdateAnchor, with one semantic
// difference: OnChainChange fires on ALL field changes (no whitelist
// filter). See §2.1 for rationale.
func (li *LiveInventory) UpdateChain(id string, mutator func(*model.Chain) error) error

// Snapshot returns a deep copy of the current inventory state. Callers
// may compute on the snapshot without holding any LiveInventory lock;
// concurrent mutations to LiveInventory do not affect the returned
// value.
//
// Deep copy is intentional. The snapshot is the unit of consumption
// for compute.NetCompressionDetail and report.RunFullAnalysis; both
// expect a consistent point-in-time view. A pointer-with-token API
// is a v0.2 ergonomics consideration; v0.1 ships deep-copy semantics
// only.
func (li *LiveInventory) Snapshot() model.Inventory

// Close releases the LiveInventory. After Close, all method calls
// return model.ErrClosed wrapped. Close does not fsync — every append
// and update already fsynced — but it does ensure the .tmp staging
// file (if any) is removed.
func (li *LiveInventory) Close() error
```

The supporting `model.ErrIDCollision` and `model.ErrClosed` sentinel errors land alongside this PR's implementation; they are referenced in `model/errors.go` (new file, ~12 lines).

### 2.1 Hook semantics — Append fires + per-kind filter

**(Added v0.2 — resolves PR #62 T8 + T9 design clarifications flagged by @contextus-impl §I4 review.)**

Two semantic questions were left implicit in v0.1 and are pinned here:

**(a) `Append*` methods fire hooks with `before == nil`.** Every successful `AppendAnchor`, `AppendChain`, `AppendConfluence` invokes the corresponding `OnAnchorChange` / `OnChainChange` / `OnConfluenceChange` hook with `before == nil` and `after` pointing at the just-appended record (deep copy, same isolation as `Snapshot`). This is symmetric with `Update*` (which always supplies both `before` and `after`). The single hook surface covers both lifecycle events; consumers register one callback per kind, not a separate "watch-new-IDs" surface.

`AppendInput` does NOT fire a hook (no `OnInputChange` defined; inputs are programme-level metadata, not anchor-graph state).

**(b) The per-field filter is anchor-specific. Chain and confluence hooks fire on all field changes.**

| Hook | Fires on Append? | Fires on Update for which fields? |
|---|---|---|
| `OnAnchorChange` | yes (`before == nil`) | only when `{Status, MeasuredValue, MeasuredError, DiscrepancyPct, LastTestedAt}` change. Other field changes (e.g., `Notes`, `Description`) do not fire. |
| `OnChainChange` | yes (`before == nil`) | **all field changes fire**, no whitelist. Chain topology IS the derivation surface; any change matters to closure-caching consumers (e.g., Contextus `cth-derivation` membership predicate). |
| `OnConfluenceChange` | yes (`before == nil`) | **all field changes fire**, no whitelist. Same rationale: confluence membership in the derivation closure determines the "what counts as evidence" set; consumers must observe every change. |

**Rationale for the asymmetry**: anchor records have many fields (`Notes`, `Description`, `References`, `Tags`, citation links, etc.) that change frequently as humans curate the inventory without affecting ρ_net or downstream consumers. The whitelist filter (`{Status, MeasuredValue, …}`) is the ρ_net-affecting subset per Theory v0.2 §4.1. Chain and confluence records, by contrast, have a small fixed shape where every field matters to some consumer; filtering would just guess wrong.

**Hook fire order**: hooks for sequential successful mutations fire in commit order. The implementation queues hook dispatch outside the critical section but preserves the order in which mutations linearised under the write lock. Consumers replaying mutations to reconstruct state can rely on the order; they need not add their own reconciliation pass for ordering.

## 3. Concurrency contract

- **Single-writer-or-multi-reader** at the LiveInventory level via `sync.RWMutex`. `Snapshot()` acquires RLock; all append/update methods acquire Lock.
- **Per-call atomicity**: each `AppendXxx` and `UpdateXxx` acquires the lock, validates, mutates in memory, fsyncs to disk (via the existing `.tmp + rename` pattern in `store.SaveInventory`), and releases. Callers see either the pre-state or the post-state, never a torn intermediate.
- **No nested locks**: hooks run outside the critical section (post-release) so hook implementations may safely call `Snapshot()` if they need read access. Hooks may NOT call append/update methods recursively — that produces a write-after-write race the API does not protect against. Document this in `Hooks` doc comments.
- **Validation under lock**: schema validation runs inside the critical section. This bounds the cost (single-record validation is O(1) wrt inventory size) and guarantees the on-disk state never violates the schema.

The §5 (PR #26) "valid but not single-snapshot" framing applies to `Snapshot()` the same way it applies to Wyrd's `query.API.NeighborNodes`: a snapshot is always internally consistent; back-to-back snapshots may differ in ways concurrent appends produce.

## 4. Atomicity contract

- **Disk-side**: per-mutation `.tmp + rename` (same path as `store.SaveInventory`). `os.Rename` is atomic on POSIX filesystems; the file is either the pre-state or the post-state on any reader (including a CTH instance in another process). This matches QBP's curation workflow expectations.
- **In-memory side**: mutation, validation, and the disk write are inside one lock window. If `Inventory.Validate()` fails after the in-memory mutation, the mutation is rolled back **before** releasing the lock. The disk file is never touched on the failure path.
- **No batching at v0.1**: every append fsyncs. This is durable but slow under high write rates. A `FlushInterval` knob (batch writes for ≤N ms, then fsync) is a v0.2 ergonomics consideration; deferred unless profiling shows it's load-bearing.
- **Failure modes documented**: if the rename fails (disk full, permission denied), the in-memory mutation is rolled back and the error is returned to the caller. The `.tmp` file is removed; the canonical file is untouched.

## 5. Cache invalidation

The `model.Inventory.Health` block carries cached `ρ_net`, `coherence_ratio`, `compression_velocity`, etc. Live mutations invalidate these.

**v0.1 strategy: clear-on-write.** Every successful append or update zeroes `inv.Health` (sets to nil). Subsequent `Snapshot()` returns the inventory with `Health == nil`; the caller (typically `report.RunFullAnalysis`) recomputes.

This is conservative — even a non-anchor-affecting update (e.g., changing an anchor's `notes` field) invalidates Health. v0.2 may add a finer-grained "which-fields-affect-which-metrics" map; v0.1 keeps it simple and correct.

## 6. NATS integration hook (CTH #19)

The `cth.scoring.{anchor_id}.score_event` NATS topic carved into CTH #19 is the federation-wide read-side for anchor status changes. LiveInventory itself has no NATS dependency. Instead, callers register a `Hooks.OnAnchorChange` callback that publishes on the topic.

BMA's continuous-loop wiring (per BMA #107):

```go
// Pseudocode; BMA-side
nc, _ := nats.Connect(...)
li, _ := store.OpenLiveInventory("bma-data/inventory.json", &store.Hooks{
    OnAnchorChange: func(before, after *model.Anchor) {
        if before.Status != after.Status {
            payload := buildScoreEvent(before, after)
            nc.Publish("cth.scoring."+after.ID+".score_event", payload)
        }
    },
})
```

This keeps:
- **LiveInventory substrate-agnostic** — no NATS import in `store/`.
- **#19 scope-contained** — the wire format and topic structure live with the NATS integration code.
- **Caller-side filtering** — only emit when status changes; routine bookkeeping doesn't spam the bus.

## 7. Wyrd-cutover substrate-swap path (Walk)

Per CTH #60 (Wyrd↔CTH schema bridging) and workspace-phase-architecture §2.7 Toddle→Walk exit gate, the underlying substrate swaps from JSON files to Wyrd hyperedges at Walk. The LiveInventory API contract is designed to survive that swap with a **single constructor switch on the caller side**:

```go
// Toddle (this PR)
li, _ := store.OpenLiveInventory("bma-data/inventory.json", hooks)

// Walk (after CTH #60 lands the bridging adapter)
li, _ := store.OpenLiveInventoryWyrd(graph, "BMA", hooks)
```

The returned `*LiveInventory` exposes the same `AppendAnchor` / `UpdateAnchor` / `Snapshot` methods. Internally, the Walk constructor swaps the disk-write-path for `graph.AddNodeWithCapability` calls (per wyrd-implementor toddle-design seq=11). The `*LiveInventory` field set may grow a `substrate` discriminator; callers don't observe it.

This is the one-line substrate-swap promise from cth-implementor toddle-design seq=20.

## 8. What v0.1 ships (this design + impl PR)

| Surface | Status |
|---|---|
| `store/live.go` | New file — LiveInventory + Hooks + OpenLiveInventory |
| `model/errors.go` | New file — ErrIDCollision + ErrClosed sentinels |
| `store/live_test.go` | New file — concurrent-append race tests, fsync verification, rollback on validation failure, hook invocation, deep-copy semantics |
| `doc/design/live-inventory-api.md` | This document |
| `doc/integration/live-api.md` | Caller-side wiring sketch (BMA, QBP, Contextus consumption patterns) |

**Not in v0.1:**
- Wyrd-backed constructor (CTH #60 dependency — Walk-gate work)
- FlushInterval batching (deferred; profile first)
- Pointer-with-token snapshot API (deferred to v0.2)
- Watch-channel surface (`func (li *LiveInventory) Watch() <-chan ChangeEvent`) — deferred; hooks cover the v0.1 use cases

## 9. Migration path

1. Land this design doc — §I4 sign-off from named reviewers.
2. Open implementation PR with `store/live.go` + `model/errors.go` + tests.
3. BMA #107 (M2 WDEvent → CTH ρ_net feedback loop) wires `Hooks.OnAnchorChange` and the NATS publish per #19.
4. QBP federation-tenancy v0.1 §5.3 (PR #403) cites this API for its bookkeeper-write flow.
5. Contextus PR #11 v1.4 wires `Hooks.OnAnchorChange` for `cth-derivation` membership TTL invalidation.
6. (Walk-α) CTH #60 lands the Wyrd-bridging adapter; `OpenLiveInventoryWyrd` ships; callers swap one line.

Steps 1–2 are in scope for this issue. Steps 3–5 are consumer-side work in sibling repos; step 6 is the Walk-gate milestone.

## 10. Open questions for review

1. **Hooks vs Watch-channel as the change-notification surface.** Hooks are simpler (caller-supplied callback runs in caller's goroutine context); Watch returns a buffered channel the caller drains. My lean: **hooks** at v0.1 — they match BMA's expected wiring pattern (per BMA #107) and avoid backpressure semantics. Pushback if anyone needs Watch for fan-out.

2. **Snapshot deep-copy vs pointer-with-token.** Deep-copy is safe but allocates; pointer-with-token (caller receives a `*model.Inventory` plus a release function) is cheaper but caller must release. My lean: **deep-copy at v0.1**; v0.2 add a `SnapshotRef()` variant returning the cheaper shape. Pushback if BMA's L3 read frequency makes the alloc cost load-bearing at Toddle.

3. ~~**Per-field status-change filtering for OnAnchorChange.**~~ **RESOLVED v0.2.** The anchor-side whitelist `{Status, MeasuredValue, MeasuredError, DiscrepancyPct, LastTestedAt}` ships at v0.1 per the original spec. **Chain and confluence hooks fire on all field changes** (no whitelist) per §2.1 — chain/confluence records have small fixed shape where every field matters to some consumer. **Append* methods fire hooks with `before == nil`** per §2.1, symmetric with Update*. Hook fire order is commit order, queued outside the critical section.

4. **`AppendAxiom` and `AppendDerivedPrinciple` absent from v0.1.** Axioms are conventionally write-once-at-programme-init; derived principles change rarely. My lean: **omit at v0.1**, add via SaveInventory full-rewrite path until a real use case surfaces. Pushback if QBP's PR7 reconciliation cycle needs them.

5. **Mutator pattern vs typed-update methods.** `UpdateAnchor(id, func(*model.Anchor) error)` is flexible but lets the mutator do arbitrary things including breaking invariants. Typed methods (`UpdateAnchorStatus(id, newStatus)`, `UpdateAnchorMeasurement(id, value, error, source, at)`) are safer but expand the surface. My lean: **mutator pattern at v0.1** with `Inventory.Validate()` running post-mutation under-lock as the safety teeth. Pushback if anyone wants typed teeth at v0.1.

## Cross-references

- **CTH #51** — this issue (live inventory update API)
- **CTH #19** — NATS event integration; carries `cth.scoring.{anchor_id}.score_event` (read-side for §6)
- **CTH #53** — `compute/scoring.go` + predictions/ schema (v0.1.x milestone; consumes UpdateAnchor)
- **CTH #60** — Wyrd↔CTH schema bridging (Walk-gate prerequisite for `OpenLiveInventoryWyrd`)
- **CTH #61** — `cth` as Theory Cart tool (Toddle-load-bearing consumer)
- **BMA #107** — M2 WDEvent → CTH ρ_net feedback loop (BMA-side consumer; wires Hooks + NATS)
- **QBP #403** — Federation Tenancy v0.1 §5.3 (QBP-side consumer)
- **Contextus PR #11** — Spec v1.4 theory-as-scope (Contextus-side consumer; OQ #3 TTL-on-mutate)
- **Wyrd PR #35** — ScoutQuery + predictions/ (federation cross-cutting; §4.1 CTHAnchor flow)
- **Wyrd PR #39** — W-Toddle-1 substrate (`AddNodeWithCapability` consumed by Walk path)
- **workspace-phase-architecture §0.11** — η = CTH ρ_net trust-anchor identity
- **workspace-phase-architecture §2.4** — CTH as L3 Beliefs substrate at Toddle
- **workspace-phase-architecture §2.7** — Toddle→Walk exit gate (federation-wide Wyrd v0.2 cutover)

## Changelog

| Revision | Date | Changes |
|---|---|---|
| v0.1 | 2026-05-14 (PR #62 merged 17:52 UTC per beekeeper-override; named-reviewer set incomplete at merge per @contextus-impl §I4 review) | Initial design surface for §51 LiveInventory API. |
| v0.2 | 2026-05-14 (this PR) | Adds §2.1 *Hook semantics — Append fires + per-kind filter*. Resolves T8 (chain/confluence hooks fire on all field changes; no whitelist) and T9 (`Append*` fires hooks with `before == nil`) clarifications flagged in @contextus-impl PR #62 §I4 review + expanded test plan. Updates AppendAnchor + UpdateAnchor + UpdateChain godoc to cross-reference §2.1. §10 OQ #3 marked RESOLVED. Test plan T8/T9 in the impl PR can now be specified without further design work. |
