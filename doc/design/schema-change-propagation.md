# CTH Schema-Change Propagation

**Status:** Crawl-phase process (v0.1). Governing doc — canonical, lives in confluent-trust (schema source-of-truth).
**Companion:** QBP-side operational checklist at `QBP/docs/cth/schema-change-propagation-checklist.md` (cross-links here; authored by qbp-implementor). Front-half local-extension governance is `QBP/docs/cth/qbp-local-extensions.md`.
**Origin:** cth-qbp-live-testing design session 2026-06-01 (seq 27–33); beekeeper-directed; phased Crawl→Toddle→Walk per beekeeper constraint.

---

## 1. Problem

The CTH schema is **multi-homed**. A single field added to an anchor touches all of:

| Surface | Repo | Role |
|---|---|---|
| `model/*.go`, `schema/inventory.schema.json`, `schema.json` | confluent-trust | **source of truth** (cth-implementor) |
| `docs/cth/inventory.schema.v0.3.json` | QBP | **vendored mirror** — what `cth-schema-lint` validates against |
| `docs/cth/inventory.schema.v0.3.meta.json` | QBP | sidecar: pinned upstream SHA + vendoring date + **schema semver** |
| `docs/cth/inventory.schema.current.json` | QBP | symlink → v0.3 |
| `archive/cth-inventory/*.json` | QBP | the actual inventory data |
| `scripts/check_cth_invariants.py` | QBP | cross-field lint (may need to know new fields) |
| `pkg/bookkeeper/testdata/*.json` | qbp-systema | bookkeeper test fixture |

Every divergence between these is a drift incident. The v5.24-fork and the stale-#483 base were both this class — a change applied to one copy, not propagated, validated against a subset rather than canonical. As CTH tooling matures, schema churn **increases**, so this is engineered for the steady state, not the current week.

## 2. Three flow directions

The process must cover all three; the checklist documents each:

1. **Upstream-first** — field originates in confluent-trust → propagates *down* to the QBP mirror + consumers. The default flow. (Pilot: `#92` `peer_review_status`.)
2. **QBP-local-awaiting-upstream** — field originates as a QBP-local extension, governed by `qbp-local-extensions.md` until promoted upstream. (Example: D/I/P provenance values.)
3. **Landed-via-catch-all → formalize-upstream** — a field landed in QBP inventory *data* via `additionalProperties: true` before it existed in the canonical schema, so the mirror data is *ahead* of canonical; cth-implementor formalizes it upstream, then it flows back down. (Pilot: `#93` `killed_by`/`killed_note`/`review_flag`.)

## 3. The phased process

The deliverable is not one sync mechanism — it is a process that **graduates as the federation does**: prevent drift by *discipline* (Crawl) → *detection* (Toddle) → *construction* (Walk).

| Phase | Mechanism | Enforcement | Trigger to advance |
|---|---|---|---|
| **Crawl (now)** | Manual vendor + co-sign + this checklist. cth-implementor opens the upstream PR; qbp-implementor mirrors `inventory.schema.v0.3.json` + bumps the `.meta.json` sidecar in a QBP PR; both co-sign. | Human discipline + existing `cth-schema-lint` (inventory vs vendored mirror). | Volume of schema changes rising past comfortable manual cadence. |
| **Toddle** | + **drift-check CI**: a `schema-sync-check` job pins the canonical schema **semver** and fails the QBP build if the vendored mirror drifts. Keyed on the `.meta.json` semver, not a private-repo fetch (see §5). | Mechanical drift detection — divergence cannot go silent. Updates still human-applied. | NATS + Wyrd substrate coming online (Walk entry). |
| **Walk** | + **single-source, consume-by-reference**: confluent-trust publishes the schema as a versioned release artifact; consumers pin to a semver instead of vendoring a copy. The second copy that can drift stops existing. | Structural — drift becomes *impossible* (one source), not just detected. | — (end state) |

The throughline: each phase is a real implementation at the right cost for its phase. Crawl is not over-engineered; Walk is not hand-waving.

## 4. The schema-document semver — the spine

Three version concepts are currently conflated and **must be kept distinct**:

1. **inventory content version** — `version` (e.g. `"5.3.2"`); changes when *anchors* change. Not what propagation keys on.
2. **schema generation** — `schema_version` (e.g. `"v0.3"`); coarse.
3. **schema-document semver** — *the new primitive*; what Toddle's drift-check and Walk's pin-by-reference key on.

Stamp the schema-document semver **now (Crawl)**, even though it is only human-checked at Crawl, so the later phases have it to pin on. Home: the `$id` URI (`…/inventory.schema.v0.3.1.json`, json-schema-canonical) **plus** a machine field; mirrored in the QBP `.meta.json` sidecar. The `$id` is upstream source-of-truth; the sidecar is the QBP-side pinned value; the drift-check asserts they match.

**minor/major rule:**
- **minor** — additive / optional field, no new `Validate` constraint, `cth migrate` no-op → **mirror may lag safely.** Structurally backed: the v0.3 schema is `additionalProperties: true` at top level and in `$defs/Anchor`, so an additive-optional field does not even break old validators.
- **major** — new required field, new `Validate` constraint, or enum removal → **breaking; mirror must update before QBP writes.**

The lint checks the compatibility class from the semver delta.

## 5. Toddle drift-check without a PAT

confluent-trust is private. A naive drift-check that fetches the canonical schema *from the private repo* reintroduces a CI credential (`WYRD_PAT`) dependency — the exact thing vendoring (PR #459) was chosen to avoid.

Resolution: the drift-check does not need private-repo read — it needs **the canonical schema + its semver at a pinned version.** So confluent-trust **publishes the schema as a versioned release artifact** (Release asset or published URL keyed by semver), and the Toddle check pins to a *published semver*. No PAT. This is the same rail Walk's consume-by-reference uses, so the PAT concern dissolves into the publish-once mechanism we already want. The Toddle trigger condition is therefore "**confluent-trust publishes schema releases**," not "WYRD_PAT provisioned." (PAT-fetch remains a fallback if publishing slips.)

## 6. Ownership

| Surface | Owner |
|---|---|
| Canonical schema + Go model + this governing doc + schema-release publishing | **cth-implementor** |
| QBP mirror + `.meta.json` sidecar + inventory + fixture sync + the operational checklist + Toddle drift-check CI | **qbp-implementor** |
| Theory-axis content of a field (does it gate scoring?) | **qbp-oppenheimer** / qbp-architecture |
| Merge of canonical schema changes | **beekeeper** (push/HVR) |

## 7. Per-change checklist (Crawl)

For every CTH schema change:

1. Classify the **flow direction** (§2) and the **semver class** (§4, minor/major).
2. **Upstream:** cth-implementor opens the confluent-trust PR — `model/*.go` + `schema/inventory.schema.json` + `schema.json` + semver bump in `$id`/machine-field. PR body carries `Closes #N` (close-by-PR rule, FAULT-2026-06-01-002).
3. **Mirror:** qbp-implementor opens the QBP PR — `inventory.schema.v0.3.json` + `.meta.json` sidecar semver/SHA bump; co-signs the upstream PR. For a **major** change, the mirror PR merges before any QBP inventory writes the new field.
4. Update `scripts/check_cth_invariants.py` if the field participates in a cross-field invariant.
5. Sync `qbp-systema/pkg/bookkeeper/testdata/*.json` from canonical (or document the divergence).
6. `cth-schema-lint` green on both sides.

## 8. Live pilots

- **#92** (`peer_review_status`) — upstream-first pilot.
- **#93** (`killed_by`/`killed_note`/`review_flag`) — landed-via-catch-all→formalize-upstream pilot (mirror data currently ahead of canonical).

Both run through this checklist as the first two exercised cases.

## 9. Future phases (documented now, implemented when triggered)

- **Toddle:** §3 row + §5 publish-artifact drift-check. Trigger: schema-change volume / Walk-entry prep.
- **Walk:** §3 row, consume-by-reference off NATS + Wyrd (`store/wyrd.go`, CTH #87). Trigger: NATS+Wyrd substrate online.

This doc names its own evolution so the process is self-documenting (same pattern as the foundations scope doc).
