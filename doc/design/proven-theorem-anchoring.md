# Proven-Theorem → CTH Anchoring Protocol

**Status:** Crawl-phase protocol (v0.1). Canonical — lives in confluent-trust (schema source-of-truth).
**Origin:** confluent-trust #96 (§I4 design surface, filed by qbp-implementor per the two-stream model, `pr407-conflict-resolution` seq=82–92). AC1 ruled by cth-implementor (issue comment 4627150444); AC2 co-signed by qbp-architecture (issue comment 4627162512). This document is AC3.
**Companion surfaces:** inter#57 (foundation↔physics dependency map — consumes this protocol's `PROVEN` definition); QBP `docs/foundations/` (cross-links here); `doc/design/schema-change-propagation.md` (the rail the schema additions ride).

---

## 1. Purpose

The two-stream model (beekeeper-directed, 2026-06-02) firewalls foundation *proving*
(qbp-oppenheimer + lean-prover; zero-sorry by construction per beekeeper ruling 480-B)
from *integration* (qbp-implementor) and physics consumption. The seam contract: the
federation consumes **proven** foundation results, never assumed ones. This protocol
defines how a proven Lean theorem becomes a CTH anchor that the dependency map
(inter#57) may mark `PROVEN` — the one mechanical bridge between proof tempo and
federation consumption.

## 2. Anchor form (Q1)

- Proven foundation theorems are **existing `PROOF-*` anchors** with
  `provenance_kind = proof`. **No new anchor prefix, no new NT_ node type.**
  Foundation-vs-physics provenance is metadata, not identity; anchor-ID prefixes carry
  domain semantics (`ClassifyDomain`, the pinned `cth://` URI grammar) and must not be
  overloaded with stream provenance.
- **`foundation_batch`** (optional string field, additive): the QBP #474
  property-matrix row id the theorem belongs to. One field, two duties:
  anchors with `foundation_batch` set *are* the foundation set (queryable), and the
  value names the batch sign-off that covers the anchor (audit trail).
- Wyrd-side NT_ representation is **deferred to Walk** (CTH #87 cutover).

## 3. Federation-consumable minimal set (Q2)

The dependency map flips a row to `PROVEN` **iff the anchor carries all five**:

1. `provenance_kind = proof` and `status = coherent` (INV1/Inv4 proof-field gates apply).
2. The #71 v0.3 provenance fields: `proof_system` + version (Lean 4 toolchain pin),
   library pins (Mathlib rev).
3. A verification record (INV2/INV3): date + toolchain.
4. The **Lean-identity join key**: `<module path>::<theorem name>` — the **same string,
   verbatim**, in the anchor's verification field and the inter#57 proof-status column.
   One key binds both artifacts; parallel naming schemes are forbidden (they drift).
5. The **`#print axioms` attestation**: the recorded axiom closure, which MUST be
   `⊆ {propext, Classical.choice, Quot.sound}` (sorryAx-clean).

An anchor missing any of 1–5 is **not federation-consumable**; its dependency-map row
stays `PENDING_PROOF` (soft blocker per the tempo rule — physics may carry forward
empirically, must not ship the result to federation as confirmed).

## 4. Cadence (Q3)

Sign-off is **per-batch**, one batch per QBP #474 property-matrix row (~14 Phase-1
batches). Per-theorem sign-off is rejected: it couples CTH review cadence to proof
tempo — the exact mismatch the two-stream firewall exists to prevent.

## 5. Roles and flow (Q4)

```
qbp-oppenheimer            qbp-implementor                cth-implementor      qbp-architecture
(proving stream)           (integration seam)             (schema authority)   (coherence)
      │                          │                              │                   │
      │ emit theorem list        │                              │                   │
      │ per batch (#474 row) ───▶│ file anchor records          │                   │
      │                          │ against CANONICAL ──────────▶│ batch-sign        │
      │                          │                              │ (verify §3 set) ─▶│ per-batch
      │                          │                              │                   │ co-sign
      │                          │◀──── dep-map rows flip to PROVEN (inter#57) ─────│
```

**Canonical target:** `QBP archive/cth-inventory/confluent-trust-inventory-v5_3.v0.3.json`
— the single theory-state authority (`cth-qbp-live-testing` seq=19–23). NOT the
`archive/cth-inventory/README.md` pointer, which is stale until QBP #509 lands.

## 6. Batch sign-off checklist (cth-implementor)

For each batch, before signing:

1. Read back the filed anchor records from canonical (never sign from the channel
   description).
2. Verify every anchor carries the §3 five-item set; spot-verify at least one
   Lean-identity key against the actual Lean source (module + theorem exist; the
   attestation's axiom set is plausible for the proof style).
3. Verify the `foundation_batch` value matches the emitted #474 row id.
4. Sign as a GitHub comment on the filing PR + sessionbridge verdict
   (comment-form per the shared-identity constraint); qbp-architecture per-batch
   co-sign follows.

## 7. Schema mechanics

`foundation_batch` is additive-optional ⟹ **minor** schema-document semver per
`schema-change-propagation.md` §4; it rides the staged canonical minor bump together
with the #93 fields (`killed_by`/`killed_note`/`review_flag`) and `class_floors`
(contextus PR #32 Q1) — one PR, one semver stamp, one QBP mirror sync; flow-direction 1
(upstream-first) of the propagation checklist.

## 8. Sequencing notes

- First batch (AC4 validation): `SedenionOctonionCount` / `Octonion32Count` /
  `FanoOrientationF3` — zero-sorry, axiom-closure compliant per `pr407` seq=91.
  Batch boundaries per qbp-oppenheimer's emit.
- v0.4 (#95 sheaf scoring) is expected to land **after** QBP #509's reconciliation under
  v0.3; if #95 timing changes, re-sequence per `pr407` seq=90 and flag on channel.
