# Inventory Schema v0.3 — design surface (CTH #71)

**Status:** §I4 design surface (Loop 1 Reference). Implementation deferred to follow-up PRs after §I4 sign-off.

**Issue:** [#71](https://github.com/JamesPagetButler/confluent-trust/issues/71) — schema: decomposed proof-formalisation provenance with version-aware verification records (federation-wide)

**Predecessor thread:** Issue #71 comments 1–3 — qbp-implementor's proposal → cth-implementor's review with 7 concerns + 4 direct answers → qbp-implementor's Corrections 1–4 fold-in → qbp-architecture's 5 Rulings → cth-implementor's synthesis acceptance.

## 0. §I4 invariant — design-doc-as-S-01-review-surface

This document is the §I4 review surface per the federation's design-doc-then-impl pattern. The four implementation PRs ship only after §I4 sign-off lands here.

**Named reviewers (per D5 reviewer-list pattern):**

- `@qbp-implementor` — **primary federation driver**. Issue #71 author; 28 PROOF-* anchors targeting v0.3 + foundations rebuild `DEFN-*` family.
- `@qbp-architecture` — **federation architecture**. CTH #71 Ruling 2 (`verification.toolchain` required) needs reflection in invariants below.
- `@wyrd-implementor` — **substrate-cutover consultative**. Verification-record fields intersect Wyrd's `cycle_counter_monotonic_per_phase` style substrate-tier theorems.
- `@bma-implementor` — **L3 Beliefs consumer angle**. CTH-as-L3-Beliefs reads anchor verification state during continuous-loop.
- `@contextus-impl` — **closure-caching consumer**. The `cth-derivation` membership predicate (Contextus PR #11 v1.4) traverses proof anchors; verification-state changes may invalidate closures.
- `@notary-implementor` — **Notary integration**. The verification record IS the Notary's CTH-resident output artifact (per BMA Theory v3.0-DRAFT §3.1 `NT_NOTARY_VERIFICATION_EVIDENCE` cross-federation pairing).
- `@beekeeper` — review & approve.

## 1. Motivation

The QBP programme is preparing a foundations rebuild (Cayley-Dickson tower ℝ → ℂ → ℍ → 𝕆 → 𝕊). A discovery audit on the QBP repository (2026-05-20) verified ~254 zero-`sorry` Lean theorems across `proofs/QBP/` and `proofs/Sprint12-Inherited/`, plus 28 `PROOF-*` anchors marked `provenance: T` per historical convention. Re-grading these correctly under a finer-grained scheme is foundation-critical.

The current v0.2 vocabulary cannot represent:

1. Whether a proof has been written in a formal proof assistant.
2. Whether such a proof has been verified.
3. Which proof assistant + version was used.
4. Which library versions (Mathlib, MathComp, etc.) were in scope at verification time.
5. External theorems we invoke as proof (e.g., Hurwitz 1898) without independent formalisation.
6. Partial progress: some theorems in a file verified, others still being authored.

Plus the Notary function (BMA Theory v3.0 §3) requires `verification.libraries.<lib>.sha` + `verification.toolchain` as machine-readable drift-detection fields. The v0.3 schema provides exactly these.

## 2. New `provenance_kind` enum

v0.3 introduces `provenance_kind: string` as the top-level kind of evidence. The legacy `provenance: <letter>` field continues to be read on v0.3 (transitional period); writes emit only `provenance_kind`.

| `provenance_kind` value | Meaning | Migrates from |
|---|---|---|
| `proof` | Formal proof in a proof assistant (Lean, Coq, PVS, Agda, Isabelle, etc.) | _(new)_ |
| `theory` | Mathematical argument in prose, internal to the programme | `T` (when internal) |
| `theory-external` | Claim relies on an external published theorem invoked as our proof (e.g., Hurwitz 1898). Distinct from `theory` because external authority backs it. | `T` (when external) |
| `experiment` | Empirical measurement | `E` |
| `hypothesis` | Tentative claim awaiting evidence | `H` |
| `internal-compute` | Numerical or symbolic computation (incl. Python verifications) | _(new — promoted from QBP-local convention)_ |
| `philosophy` | Conceptual framing or programmatic principle | _(new — promoted from QBP-local convention)_ |

**Provenance migration from v0.2:** mechanical for `E` → `experiment` and `H` → `hypothesis`. Per-anchor decision for `T` → `theory` vs `theory-external` (CLI tooling flags candidates; James decides). `I` and `P` were QBP-local extensions that bypassed strict v0.2 validation; v0.3 promotes both upstream into the canonical federation enum.

## 3. Status enum extension

Per qbp-implementor Correction 4 (CTH #71 comment 2): the same audit-defect class that produced `I` / `P` provenance also produced `killed` / `marginal` / `converged` / `falsified` Status values in QBP's local inventory. These bypassed v0.2 strict Status enum validation.

**v0.3 Status enum: `{coherent, untested, incoherent, contested, refuted, killed, marginal, converged, falsified}` — 4 additions per option (a) propagate-upstream.**

Rationale per cth-implementor synthesis (CTH #71 comment 4):

- `killed` carries connotation `refuted` doesn't — KILLED-* anchors are *definitively disproved with the proof recorded in CTH*; `refuted` admits "still-contested counter-evidence"
- `converged` is load-bearing for `CONV-*` family anchors; doesn't fit `coherent` (which is a single-anchor judgment, not a convergence-of-multiple-paths judgment)
- `marginal` sits between `coherent` and `contested`
- `falsified` is `refuted`'s stronger sibling (proof rather than counter-evidence)

QBP migration is loss-less (no Status remapping required). Other federation consumers gain richer vocabulary they may opt into.

## 4. Proof-bearing anchor fields

When `provenance_kind == "proof"`, additional fields are required.

```go
// model/anchor.go (v0.3 — additive over v0.2)

type Anchor struct {
    // ... existing v0.2 fields preserved ...

    // v0.3 additions
    ProvenanceKind  string             `json:"provenance_kind,omitempty"`
    ProofState      string             `json:"proof_state,omitempty"`      // verified | partial | written | null
    ProofLanguage   string             `json:"proof_language,omitempty"`   // lean4 | lean3 | coq | pvs | agda | isabelle | ...
    Theorems        []TheoremRef       `json:"theorems,omitempty"`
    Verification    *VerificationRecord `json:"verification,omitempty"`

    // v0.3 theory-external fields
    TheoryCitation  string             `json:"theory_citation,omitempty"`
    TheoryDOI       string             `json:"theory_doi,omitempty"`
    TheoryURL       string             `json:"theory_url,omitempty"`

    // v0.3 mixed-language fallback
    AdditionalVerifications []AdditionalVerification `json:"additional_verifications,omitempty"`
}

type TheoremRef struct {
    Name     string `json:"name"`                // theorem identifier in the proof assistant
    Status   string `json:"status"`              // verified | written | not_started
    Blockers string `json:"blockers,omitempty"`  // optional human-readable note
}

type VerificationRecord struct {
    Toolchain   string                       `json:"toolchain"`   // REQUIRED per CTH #71 Ruling 2
    Libraries   map[string]LibraryRef        `json:"libraries"`
    VerifiedAt  string                       `json:"verified_at"` // ISO 8601
    Verifier    string                       `json:"verifier"`    // authority that signed off
    Result      string                       `json:"result"`      // zero-sorry | partial | tcc-discharged | safe-mode-clean | ...
}

type LibraryRef struct {
    Ref string `json:"ref"`           // human-readable tag (e.g., "v4.30.0")
    SHA string `json:"sha"`           // immutable commit hash (ground truth)
    URL string `json:"url,omitempty"` // source URL
}

type AdditionalVerification struct {
    ProofLanguage string              `json:"proof_language"`
    ProofFile     string              `json:"proof_file"`
    Theorems      []TheoremRef        `json:"theorems"`
    Verification  *VerificationRecord `json:"verification"`
}
```

### 4.1 ProofState rollup semantics

- `verified` — all listed theorems have `status: verified` in the latest verification run
- `partial` — mix of `verified`, `written`, and/or `not_started` theorems
- `written` — proof file exists, no theorems verified yet
- `null` (absent) — no proof file (anchor in planning stage)

### 4.2 Proof language enum

v0.1 admits: `lean4`, `lean3`, `coq`, `pvs`, `agda`, `isabelle`. Additive at v0.x.x — adding a new language is a minor bump, never breaking.

## 5. Verification record (CTH #71 §C + Ruling 2)

The verification record captures the complete reproducibility context. Required when `proof_state ∈ {verified, partial}`:

```json
{
  "toolchain": "leanprover/lean4:v4.30.0-rc2",
  "libraries": {
    "mathlib": {
      "ref": "v4.30.0",
      "sha": "abc123def456...",
      "url": "https://github.com/leanprover-community/mathlib4"
    }
  },
  "verified_at": "2026-05-15T14:23:00Z",
  "verifier": "lake-build-ci",
  "result": "zero-sorry"
}
```

**Required fields per Invariant 2 (below):** `toolchain` (non-null), `libraries` (each entry has non-null `sha`), `verified_at`, `verifier`, `result`.

**`result` per-language enum** (extensible; admits unknown values for future languages):
- Lean: `zero-sorry`, `partial`
- Coq: `zero-admits`, `partial`
- PVS: `tcc-discharged`
- Agda: `safe-mode-clean`

## 6. Schema invariants (CI-enforced + runtime-validated)

Per CTH #71 §"Proposed schema invariants" + Invariant 5 added by my review §5.

**Invariant 1:** `status: untested` ⟺ `provenance_kind == "theory"` (or `"hypothesis"`) AND no formal proof attempted (`proof_state` absent or null). If a claim has been formalised in a proof assistant (any `proof_state` value), the act of formalisation has tested it; `status: untested` becomes incoherent. CI lint.

**Invariant 2:** `proof_state ∈ {verified, partial}` ⟹ `verification` non-null AND `verification.libraries.<lib>.sha` non-null AND **`verification.toolchain` non-null** (per CTH #71 Ruling 2). No "verified but we can't tell against what" anchors. Schema-level via JSON Schema Draft 2020-12 `if/then`.

**Invariant 3:** `proof_state == "verified"` ⟹ all theorems in `theorems[]` have `status: "verified"`. The rollup must match the per-theorem state. Runtime in `model.Anchor.Validate()`.

**Invariant 4:** `provenance_kind != "proof"` ⟹ `proof_state`, `proof_language`, `proof_file`, `theorems`, `verification` are all null or absent. Non-proof anchors don't carry proof-specific fields.

**Invariant 5 (NEW — phantom-artifact rule):** `theorems[].status == "not_started"` ⟹ theorem name does **not** appear in `proof_file` on disk. Transitioning to `written` requires CI lint (via `cth lean-link` per #54) to find the name in the file. Closes §2.g phantom-artifact gap for forward-declared theorems.

## 7. Transitional dual-field reading (v0.2 → v0.3)

The rename `provenance` → `provenance_kind` is **narrowing** at the Go type level; per federation-additive-only contract a transitional period is required:

- **v0.3 schema_version** dispatch in `store.LoadInventory` (precedent: v0.1↔v0.2 binary→N-ary confluence migration)
- **Both `provenance` (legacy) AND `provenance_kind` (new) accepted on v0.3 read.** If both present, `provenance_kind` wins. If only `provenance` present, mechanical migration applied (T → theory; E → experiment; H → hypothesis; I → internal-compute; P → philosophy).
- **`SaveInventory` emits `provenance_kind` only on v0.3 write.** The legacy `provenance` field is not written.
- **One phase later (v0.4?):** drop legacy `provenance` read support; CTH stops accepting v0.2 inventories.

## 8. NATS topic event-kind discriminator (CTH #71 concern #7)

The `cth.scoring.{anchor_id}.score_event` topic (CTH #19) currently carries anchor-status changes. v0.3 verification events (e.g., `proof_state` transitions, `verification.libraries.<lib>.sha` changes detected by `cth lean-link`) fold into the **same topic** with an `event_kind` field discriminator:

```json
{
  "event_kind": "verification",   // or "score" (existing) | "anchor_status" (existing)
  "anchor_id": "PROOF-foo",
  "before": { "proof_state": "written", ... },
  "after":  { "proof_state": "verified", "verification": { ... } },
  "fired_at": "2026-05-20T..."
}
```

No new topic; existing subscribers (BMA L3 reader, Contextus `cth-derivation` closure-cache) filter on `event_kind`.

## 9. Mapping from v0.2 to v0.3

Per CTH #71 concern #3 — existing Lean-specific fields supersede + need migration.

| v0.2 field | v0.3 destination |
|---|---|
| `provenance: "T"` | `provenance_kind: "theory"` OR `"theory-external"` (per-anchor decision; tooling flags candidates) |
| `provenance: "E"` | `provenance_kind: "experiment"` |
| `provenance: "H"` | `provenance_kind: "hypothesis"` |
| `provenance: "I"` (QBP-local) | `provenance_kind: "internal-compute"` |
| `provenance: "P"` (QBP-local) | `provenance_kind: "philosophy"` |
| `proof_system: "lean4"` | `proof_language: "lean4"` |
| `lean_theorem: "foo"` | `theorems: [{name: "foo", status: "verified"}]` |
| `lean_companion_theorems: [...]` | append to `theorems[]` |
| `sorry_count: 0` | `proof_state: "verified"` |
| `sorry_count > 0` | `proof_state: "partial"` |
| `proof_file: "..."` (Lean only) | `proof_file: "..."` (language-agnostic; pairs with `proof_language`) |
| `status: "killed"` (QBP-local) | `status: "killed"` (now canonical) |
| `status: "marginal"` (QBP-local) | `status: "marginal"` (now canonical) |
| `status: "converged"` (QBP-local) | `status: "converged"` (now canonical) |
| `status: "falsified"` (QBP-local) | `status: "falsified"` (now canonical) |

Migration tool fills `verification` field with **partial information** for existing Lean anchors (toolchain + library sha not present in v0.2; tooling prompts for manual entry OR marks `verifier: "migration-stub-2026-05-XX"` + `verified_at` null).

## 10. Federation rollout sequencing (CTH #71 Ruling 1)

1. **CTH design surface PR** (this PR) — §I4 cycle with named readers per §0
2. **CTH impl PRs** (4 in sequence):
   - `model.Anchor` v0.3 struct + JSON Schema dual-file (`schema/inventory.schema.json` + `internal/validate/schema.json` via `TestSchemaInSync`)
   - `cth migrate v0.2 -> v0.3 <inventory.json>` CLI subcommand
   - CTH #54 (`cth lean-link`) co-evolution with v0.3 verification record
   - Integration tests + fixture round-trip (extending `testdata/predictions_lifecycle.json` to v0.3)
3. **QBP first consumer** — 28 PROOF-* re-graded via `cth migrate`; foundations rebuild `DEFN-*` family targets v0.3 from day 1
4. **QBP-CU second** — Notary drift detection consumes `verification.libraries.<lib>.sha` + `verification.toolchain`
5. **Wyrd, BMA, Contextus on own schedules** — no immediate PROOF-* backlog; transitional dual-field reading keeps pre-v0.3 consumers functional

## 11. Sprint planning (CTH #71 Ruling 3 + Ruling 5)

- **Dedicated `v0.3-schema` milestone** (NOT folded into Option F per Ruling 5)
- Design surface PR (this PR) opens Sprint 2 second-half post-Notary-Phase-1-first-dispatch (signal landed: `live-test` seq=225 2026-05-21 00:30:37Z)
- CTH impl PRs follow §I4 close; sequencing per §10
- QBP-side: human-judgment pre-pass classifying 28 PROOF-* anchors (Ruling 3) is QBP-side work, no CTH dependency

## 12. What v0.1 of this design ships

This PR (the design surface) ships only the design doc. Implementation PRs that follow ship:

```
model/anchor.go              — TheoremRef + VerificationRecord + LibraryRef + AdditionalVerification structs + Anchor v0.3 fields
model/enums.go               — Status enum extension (killed/marginal/converged/falsified) + new ProvenanceKind type
schema/inventory.schema.json — v0.3 schema with $defs for new types + Invariants 1–5 via if/then conditionals
internal/validate/schema.json — synced via TestSchemaInSync
store/json.go                — v0.3 schema_version dispatch; transitional dual-field reading per §7
cmd/cth/migrate.go           — cth migrate v0.2 -> v0.3 CLI
testdata/predictions_lifecycle_v0_3.json — v0.3 fixture extending the v0.2 lifecycle fixture
doc/integration/v0_3-migration.md — caller-side wiring sketch
```

## 13. Migration path

1. Land this design doc — §I4 sign-off from named reviewers per §0
2. Open impl PR #1: `model/` + `schema/` dual-file + tests
3. Open impl PR #2: `cmd/cth/migrate.go` CLI + e2e tests against v0.2 fixtures
4. Open impl PR #3: CTH #54 (`cth lean-link`) co-evolution; reads v0.3 verification record
5. QBP-side: `cth migrate testdata/qbp_v3_2.json -o testdata/qbp_v3_2_v0_3.json` + per-anchor classification
6. (Walk-α) Federation-wide consumers migrate at their own pace; v0.4 drops legacy `provenance` field reading

Steps 1–3 in scope for the `v0.3-schema` milestone.

## 14. Open questions for §I4 reviewers

1. **`cth migrate` interaction model** — interactive prompts for per-anchor `theory` vs `theory-external` decision, OR a config file with explicit per-anchor decisions, OR generates a markdown report + caller edits + re-runs? My lean: **generate markdown report + caller edits + re-runs** (matches PR-shaped diff workflow per qbp-implementor's first-2-bookkeeper-ops pattern at sprint-1-closeout seq=14).

2. **`verification.toolchain` format** — single string (e.g., "leanprover/lean4:v4.30.0-rc2") OR structured object (e.g., `{compiler: "lean4", version: "v4.30.0-rc2", channel: "release"}`)? My lean: **single string** at v0.3 (matches `lean-toolchain` file convention); revisit if/when Coq + PVS bring different formats.

3. **`additional_verifications[]` priority ordering** — explicit `priority: int` field per entry, OR caller-defined order? My lean: **caller-defined order** at v0.3 (defer `verification.authority` field to v0.4 per qbp-implementor concern #6 acceptance).

4. **`event_kind` discriminator schema** — strict enum `{score, verification, anchor_status}` OR open-ended string? My lean: **strict enum** at v0.3 (matches JSON Schema discipline); add values via minor bump as new federation event types surface.

5. **Backward-compat horizon for legacy `provenance` field** — drop at v0.4 (one phase) OR v0.5 (two phases)? My lean: **v0.4** — gives federation consumers one full sprint to migrate after v0.3 ships.

## Cross-references

- **CTH #71** — proposal issue with full thread (qbp-implementor proposal + cth-implementor review + qbp-architecture rulings + synthesis)
- **CTH #19** — NATS event integration; carries `cth.scoring.{anchor_id}.score_event` (read-side for §8)
- **CTH #54** — `cth lean-link` (Notary-adjacent primitive; co-evolves with v0.3 per Ruling 1)
- **CTH #60** — Wyrd↔CTH schema bridging doc (Walk-gate dependency; v0.3 schema feeds the bridging table)
- **CTH PR #62 / #65** — live-inventory v0.1/v0.2 design surface + impl (precedent for §I4 design-surface-first pattern)
- **BMA Theory v3.0 §3** — Notary function specification; `NT_NOTARY_VERIFICATION_EVIDENCE` is the Wyrd-resident shape; CTH `Anchor.verification` is the CTH-resident shape; same evidence at different federation scopes
- **Wyrd PR #35 §4.1** — `Prediction.CTHAnchor.AnchorID` PRED-* handshake (federation contract)
- **Contextus PR #11 v1.4** — `cth-derivation` membership predicate (consumer of v0.3 verification events)
- **`sprint-1-closeout-2026-05-17`** §2.g (phantom-artifact rule), §2.h (Notary role authorisation), §2.i (federation rule #7)
- **`live-test` seq=225** — Notary Phase 1 first dispatch fire signal (the trigger for opening this design surface)
