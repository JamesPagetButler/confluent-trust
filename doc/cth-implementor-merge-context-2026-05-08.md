# CTH-Implementor Context — QBP Merge Implementation

**To:** cth-implementor (the CTH session)
**From:** qbp-architecture (Claude Opus 4.7) + James Paget Butler
**Date:** 2026-05-08
**Status:** Context briefing — read on next bridge poll. Three new issues filed (#54 + #55 + #56) for work this prompt sets up.

---

## TL;DR

You shipped CTH v0.1.0 on 2026-05-05. Since then, **two architectural conversations have landed work on your plate** that you need context on:

1. **Federation tenancy pattern** (Contextus PR #8 + QBP PR #403) — QBP becomes the first live federation tenant; CTH is the scoring infrastructure
2. **QBP integration / four-fork reconciliation** (this document) — QBP's work has fragmented across four parallel streams; the reconciliation primitive is CTH's `merge` (already shipped) + two new CTH primitives (`lean-link` + `manifest`) + one schema extension (INST-*)

You have **five live issues**:
- **#51** live inventory update API (live append/mutate)
- **#52** qbp_v3_2 → v0.2 schema migration (qbp-implementor instantiation prerequisite)
- **#54** `cth lean-link` (new — Branch L reconciliation)
- **#55** `cth manifest <layer>` (new — view on inventory, not stored state)
- **#56** INST-* anchor type + boot_requirements field (new — schema extension)

Plus a bridge ping (seq=33 on addendum-18-walk) for the **ScorePrediction issue** still pending your filing.

This doc explains the architectural why behind #54 + #55 + #56 so you can move on them with full context.

---

## 1. What happened since you shipped v0.1.0

### 1.1 BMA Theory Addendum 18 landed (BMA PR #132, merged 2026-05-08)

The federation now has an explicit **hypergraph access pattern** — Stance × Locale × Scout × Scoring. Three docs in `~/Documents/BMA/theory/hypergraph-inference/`:

- `BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md` — v0.1 (10 sections)
- `A18-v0.2-design-surface.md` — v0.2 changes (3 blockers P8/P10/P12 + 9 wording fixes + new content additions)
- `Hypergraph-Inference-BMA.md` — Gemini's seed paper

**Why this matters for you:** A18 §2.4 names CTH as the **scoring infrastructure** (compute.NetCompressionDetail, ChainFidelity, PairwiseMI/NaryMI, InformationDeficit) for the federation's predictions/observations loop. The 3-layer prediction infra (per addendum-18-walk D3):

- BMA `params.ProposalStore` — operational parameter predictions (existing in BMA)
- Wyrd `predictions/` — NT_SIGNAL referent storage with optional `cth_id` (PRED-* prefix)
- **CTH compute primitives** — your stuff, plus a `ScorePrediction(kind, pred, obs) Score` wrapper

The ScorePrediction wrapper + `cth score` CLI subcommand + `predictions/` JSON sub-schema is the issue you owe per bridge ping seq=33 (addendum-18-walk channel). Title shape: `[v0.1.x] compute/scoring.go + predictions/ schema + cth score CLI subcommand (A18 §2.4 glue)`.

### 1.2 Contextus Tenancy Pattern landed (Contextus PR #8 — open)

`~/Documents/Contextus/doc/contextus-tenancy-pattern.md` defines the **reusable pattern** for running CTH + Contextus + Wyrd as a live system for a research programme, with BMA observing.

**Why this matters for you:** the pattern frames Wyrd as substrate; Contextus and CTH as **named projections / views** on the Wyrd graph, NOT separate stores. This composes with the architectural choice for the merge work below.

### 1.3 QBP Federation Tenancy landed (QBP PR #403 — open)

`~/Documents/QBP/docs/qbp-federation-tenancy.md` declares QBP as the **first live tenant**. It calls out:

- §5 baseline = qbp_v3_2 ported to v0.2 schema (your CTH #52)
- §5.3 live updates = append API (your CTH #51)
- §6.4 ρ_net regression alarm (downstream of BMA #107)
- §8 cross-project issues to file

### 1.4 QBP integration discovery from web Claude session

James worked with a web Claude session on the four-fork reconciliation problem. The four forks:

- **Branch P (Physics)** — the git QBP repo as it stands (theory papers, derivations, predictions, some Lean infrastructure)
- **Branch L (Lean)** — the substantial Lean 4 proof corpus (Sedenion, Quaternion, Graphene, Kitaev, Bi₂Se₃, Crystallisation, Elements, plus the Wyrd-* series)
- **Branch C (Confluent Trust Hypergraph)** — your domain. Web session has been advancing inventory state to v5.13 (150 anchors) while your v0.1.0 release was based on the v0.2 schema. **Schema-version drift between web-session CTH state and Go-implementation CTH state is real.**
- **Branch W (Wisdom Registry)** — a NEW layer (cognitive lenses; no anchors; no error bars). Source theory at `~/Documents/QBP/archive/QBP-First-Wisdom-Theory.md` argues for a separate registry.

The web Claude session proposed a **manifest layer** stored alongside CTH (manifest/inventory.json + manifest/runtime-stack.json + manifest/bindings/). qbp-architecture's counter-proposal (which James accepted): **don't store a separate manifest layer; generate manifests as views on CTH**. Single source of truth.

This produces three new CTH primitives + one schema extension (the three CTH issues just filed).

---

## 2. The three new issues you have

### Issue #54 — `cth lean-link`

**Goal:** Cross-reference Lean theorems with PROOF-* anchors in an inventory.

**Why it matters:** The QBP `archive/lean-project/` has **208 theorems with 2 sorries** (per the QBP State Report just produced — see §3 of this doc). PROOF-* anchors already carry `proof_file` + `sorry_count` + `last_tested_at` schema fields per v0.2. The link from those fields to actual Lean files **doesn't exist as a primitive yet** — anchors hand-curated.

**v0.1 design:** new CLI subcommand `cth lean-link <inventory.json> <lean-corpus-root>` that walks the corpus, extracts theorems via regex, reconciles against PROOF anchors, and reports proven / orphan / stale / drift. Optional `--update-inventory` for atomic write-back.

**Stdlib-only doable.** No Lean toolchain dependency at v0.1.

### Issue #55 — `cth manifest <layer>`

**Goal:** Generate per-runtime-layer boot manifests as **deterministic views on CTH inventory**, not stored state.

**Why it matters:** The runtime stack (qbpcu, Wyrd, CTH, Contextus, BMA) needs each layer's startup requirements pulled from a single source. Web Claude wanted to store these in `manifest/`. qbp-architecture's read: **the manifest IS the query** — generated, not stored. Eliminates two-source-of-truth.

**v0.1 design:** CLI subcommand `cth manifest <inventory.json> <layer-name>` (e.g., `cth manifest inventory.json INST-qbpcu`). Emits JSON with the layer's `boot_requirements` + referenced anchors. View-only operation — never mutates inventory.

**Depends on #56 landing first** (INST-* schema extension).

### Issue #56 — INST-* anchor type + `boot_requirements` field

**Goal:** Extend the schema so runtime layers themselves live as inventory entries.

**Why it matters:** Each runtime layer (qbpcu, Wyrd, CTH, Contextus, BMA) becomes an INST-* anchor. Its `boot_requirements` field references other anchors (PROOF-* for proofs, MEAS-* for known measurements, etc.). The manifest views in #55 are then just queries on these.

**v0.1 design (schema-only):**

```go
type Anchor struct {
    // ... existing fields ...
    BootRequirements []AnchorRef `json:"boot_requirements,omitempty"`
}

type AnchorRef struct {
    AnchorID string            `json:"anchor_id"`
    Role     string            `json:"role,omitempty"`
    Required bool              `json:"required,omitempty"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

Plus AnchorType enum extension to include `"INST"`. Plus JSON Schema update. Backward-compat preserved (existing inventories without INST-* anchors still load/analyse cleanly).

**This issue should land BEFORE #55** since #55 queries on INST-* anchors.

---

## 3. State of QBP relative to your work

A QBP State Report just landed at `~/Documents/QBP/archive/QBP-Repo-State-Report-2026-05-08.md`. Key findings relevant to you:

- **40+ CTH inventory versions exist in `archive/`** (v2 through v5.13). The web session has been advancing rapidly.
- **The web doc claimed v5.1 = 138 anchors; actual v5.1 file has 126 anchors.** v5.13 has 150 anchors. v5.13 PROOF anchors don't appear to populate `proof_file` per a quick jq probe — schema may have evolved divergently from Go-implementation v0.2.
- **The `archive/lean-project/` corpus has 208 theorems / 2 sorries**, substantially larger than the web doc's "~82 theorems" claim.
- The web doc's expectation that Wyrd-Supervisor-Architecture + Skuld-Spec + QBP-CU SiFive specs would be in the QBP repo is **wrong** — those live in sibling repos (`~/Documents/Wyrd/` + `~/Documents/QBP-Compute-Unit/`).
- **WISDOM-* anchors exist in v5.x inventories** despite the W theory explicitly arguing for separate registry. CTH #52 (qbp_v3_2 → v0.2 schema migration) is a natural place to migrate them OUT into a separate `wisdom-registry.yaml` at QBP repo root.

**Open architectural question for you to surface in your design surfaces:** which version of the web-session inventory (v5.1 or v5.13?) should drive CTH #52 migration? Or does CTH #52 stay scoped to qbp_v3_2 only, and v5.x → v0.2 schema migration is a separate v0.1.x or v0.2 issue?

My recommendation: **scope CTH #52 to qbp_v3_2 only** (that's what its existing scope says); **file a follow-on issue for v5.13 → v0.2** schema migration once you have a clearer read on whether v5.x is a different schema entirely or just data evolution within the same schema. If different schema → separate migration; if same schema → bigger CTH #52.

---

## 4. The Wisdom Registry decision

Per `~/Documents/QBP/archive/QBP-First-Wisdom-Theory.md` (dated 2026-04-27), wisdoms are explicitly NOT anchors — they have no error bars, no truth values, no provenance class. They're cognitive lenses. The W theory says they belong in a **separate registry**.

v5.x inventories added them as WISDOM-* anchors anyway. **This is a leak.**

**Your call** (in coordination with qbp-implementor when instantiated):

- (a) Migrate WISDOM-* anchors OUT of CTH; emit a separate `wisdom-registry.yaml` at QBP repo root. Reference from BMA's manifest view but not from the inventory.
- (b) Keep WISDOM-* as a non-standard tier in CTH; the manifest view filters them appropriately.
- (c) Refactor CTH schema to support multiple "registries" (anchor registry + wisdom registry + other future registries) in one file.

**Recommendation: (a)** — matches the W theory's own argument; keeps schema semantics clean.

---

## 5. Issue priorities & ordering

Suggested order:

1. **#52** (qbp_v3_2 migration) — qbp-implementor instantiation prerequisite. **Highest priority** for federation forward progress.
2. **ScorePrediction issue** (bridge seq=33) — file this if not already filed; A18 §2.4 substrate.
3. **#56** (INST-* schema extension) — small surgical change; unlocks #55.
4. **#51** (live inventory update API) — enables BMA #107 feedback loop downstream.
5. **#54** (cth lean-link) — independent; can ship in parallel with #51/#56.
6. **#55** (cth manifest) — depends on #56.

§I4 design-surface PRs welcome for each; named reviewers per individual issue.

---

## 6. Process notes

- **Federation default merge type: Squash** (rebase only for multi-commit branches with meaningful commit boundaries). See `~/Documents/github-best-practices.md` §1.4.
- **Branch protection** active on confluent-trust main (PR-required, no force push, no deletion, linear history, enforce-admins).
- **Co-author trailers** on every commit. Pattern: `Co-Authored-By: James Paget Butler <jamespagetbutler@gmail.com>` + AI co-authors as relevant.
- **Branch retention:** do NOT auto-delete on merge; release-time cleanup per `github-best-practices.md` §5.5.

---

## 7. Where to find context

| Resource | Path |
|---|---|
| This context doc | `~/Documents/CTH/cth/doc/cth-implementor-merge-context-2026-05-08.md` |
| QBP State Report (just landed) | `~/Documents/QBP/archive/QBP-Repo-State-Report-2026-05-08.md` |
| QBP Integration Discovery Prompt (web Claude session) | `~/Downloads/QBP-Integration-Discovery-Prompt.md` |
| QBP Federation Tenancy v0.1 (open PR) | `JamesPagetButler/QBP#403` (`~/Documents/QBP/docs/qbp-federation-tenancy.md`) |
| Contextus Tenancy Pattern v0.1 (open PR) | `JamesPagetButler/contextus#8` (`~/Documents/Contextus/doc/contextus-tenancy-pattern.md`) |
| BMA Theory Addendum 18 | `~/Documents/BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md` |
| Wisdom Theory (source) | `~/Documents/QBP/archive/QBP-First-Wisdom-Theory.md` |
| Latest CTH inventory (web-session state) | `~/Documents/QBP/archive/confluent-trust-inventory-v5.13.json` |

---

## 8. Bridge

You're subscribed to `addendum-18-walk` and `live-test`. The seq=33 ping for ScorePrediction issue is in `addendum-18-walk` history. A new bridge update lands today summarising the 3 new issues + state-report + this context doc.

When you pick up #56 (INST-* schema), file a §I4 design-surface PR and ping qbp-architecture + bma-implementor + Gemini + wyrd-implementor + qbp-implementor (when instantiated) as named reviewers.

---

*CTH-Implementor Context Brief | 2026-05-08*
*Co-Authored-By: James Paget Butler (Beekeeper)*
*Co-Authored-By: Claude Opus 4.7 (qbp-architecture)*
