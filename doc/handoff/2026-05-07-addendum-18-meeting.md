# Addendum-18-Walk Meeting Handoff — CTH-side record

**Channel:** `#addendum-18-walk` (sessionbridge MCP)
**Kickoff:** 2026-05-07
**Round-2 closeout:** 2026-05-13
**Channel retired:** 2026-05-14 (post Wyrd PR #35 merge)
**Author:** cth-implementor (Claude Opus 4.7)
**Audience:** Future CTH worker sessions inheriting the federation context

---

## TL;DR

The addendum-18-walk meeting absorbed **BMA Theory Addendum 18 v0.1** ("Hypergraph Access Pattern: Stance × Locale × Scout × Scoring") into the federation. CTH was a participating instance (4th in Round 1 floor order). The meeting produced 15 D-decisions and 12 P-pushbacks; three of each fell on CTH's axis (D13/D14/D15, P7/P8/P9). All committed CTH-side work has shipped or is in flight as of 2026-05-14.

Effective state at handoff:

- **`compute.ScorePrediction`** primitive shipped (PR #63, merged) — the A18 §2.4 scoring glue
- **`store.LiveInventory`** shipped (PR #65, merged) — the live append/mutate surface for BMA's L3 Beliefs continuous-loop
- **`predictions_lifecycle.json` fixture** shipped (PR #67, awaiting merge) — the canonical end-to-end test target
- **`cth score` CLI** shipped (PR #68, stacked on #67) — closes the milestone v0.1.x — Scoring Complete
- **Wyrd PR #35** (federation predictions/ schema) merged 2026-05-14 15:03 UTC — the federation contract CTH's PRED-* anchors handshake against

CTH is now load-bearing for BMA's L3 Beliefs cognitive layer at Toddle per workspace-phase-architecture §2.4. The Wyrd-cutover gate for Walk is tracked under milestone **v0.2 — Walk Gates** (#54, #55, #56, #60).

---

## 1. What the meeting was

**BMA Theory Addendum 18 v0.1** (`~/Documents/BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md`) introduced the four-axis hypergraph access pattern: **Stance × Locale × Scout × Scoring**. The federation absorption meeting ran on `#addendum-18-walk` over multiple work cycles:

- **Round 1** (2026-05-07 → 2026-05-08): each implementor 5-minute floor for substantive read. CTH-implementor's floor was 4th.
- **Round 2** (2026-05-09 → 2026-05-11): async Q1–Q7 letter-shaped questions, with named-implementor decisions per letter.
- **Closeout** (2026-05-13): qbp-architecture posted Round-2 closeout (seq=49); channel-of-record handoff to per-PR threads.
- **Sync-meeting offer** (2026-05-12 → 2026-05-14): Marcy (BMA Gen 61) held the offer pending Wyrd PR #35 §I4 reads (sync-meeting tally: 4/5 at cth-implementor's read; 5/5 at contextus-impl's).
- **Toddle-design cascade** (2026-05-14 04:14 UTC): the architectural cluster cascaded into a separate `#toddle-design` meeting. Closed 05:05 UTC.
- **Both channels retired**: post Wyrd PR #35 merge.

---

## 2. Decisions on CTH's axis

Three D-decisions named CTH directly:

| ID | Decision | CTH-side artifact |
|---|---|---|
| **D13** | `compute/scoring.go` ships in CTH (not BMA) — single ScorePrediction primitive over CTH's existing `InformationDeficit` + `PairwiseMI` machinery. KindScalar + KindCategorical at v0.1; KindProcess deferred per P7. | PR #63 (merged) |
| **D14** | NATS topic for federation-wide score publication: `cth.scoring.{anchor_id}.score_event`. Owned by CTH; consumed by BMA's L3 Beliefs + Contextus's `cth-derivation` membership predicate. | Comment on #19 (carved scope); wired via `store.LiveInventory.Hooks.OnAnchorChange` per CTH #51 |
| **D15** | When a BMA `NT_SIGNAL` mints a prediction that's also a CTH `PRED-*` anchor, the Wyrd Prediction stamps `CTHAnchor.AnchorID = "PRED-*"` for federation scoring. CTH's own `Anchor.ID` IS the cth_id from CTH's vantage — no new field needed. | Wyrd PR #35 §4.1 carries the wyrd-side stamp; CTH schema docs the PRED-* convention via PR #67 |

Three P-pushbacks were CTH-relevant:

| ID | Pushback | Resolution |
|---|---|---|
| **P7** | Staging invariant: ScorePrediction must NOT admit `KindProcess` (MI on probability distributions) at v0.1. Deferred to v0.2 of A18. | `compute.ScorePrediction` v0.1 errors on KindProcess; only Scalar + Categorical admitted. |
| **P8** | Ordering: `predictions/` JSON schema should ship concurrent with Wyrd ScoutQuery v0.1 (not after) so NT_SIGNAL referents have a landing place. | Met via Wyrd PR #35 (merged 2026-05-14) defining `predictions/` simultaneously with CTH PR #67 predictions-lifecycle fixture. |
| **P9** | `InformationDeficit` is the canonical delta primitive — ScorePrediction should compose it, NOT introduce a new entropy framework. | `compute.ScorePrediction` is a thin wrapper over existing primitives (`-log2(delta)` for ConfirmInfo; matches `ConfirmatoryInfo` semantics). |

---

## 3. CTH-side artifacts landed

| PR | Title | Status |
|---|---|---|
| [#58](https://github.com/JamesPagetButler/confluent-trust/pull/58) | `testdata: qbp_v3_2 inventory migrated to v0.2 schema (#52)` | merged 2026-05-11 (qbp-implementor instantiation blocker cleared) |
| [#62](https://github.com/JamesPagetButler/confluent-trust/pull/62) | `doc(design): live inventory update API §I4 surface (#51)` | merged 2026-05-14 17:52 (beekeeper-override pre-contextus-impl §I4) |
| [#63](https://github.com/JamesPagetButler/confluent-trust/pull/63) | `feat(compute): ScorePrediction primitive for A18 §2.4 scoring loop (#53)` | merged 2026-05-14 18:15 |
| [#64](https://github.com/JamesPagetButler/confluent-trust/pull/64) | `doc(design): live-inventory-api v0.2 — resolve T8 + T9 clarifications` | merged 2026-05-14 18:15 (resolves contextus-impl §I4 hook-semantics gaps) |
| [#65](https://github.com/JamesPagetButler/confluent-trust/pull/65) | `feat(store): LiveInventory — live append/mutate API (#51)` | merged 2026-05-14 18:36 (closes #51) |
| [#66](https://github.com/JamesPagetButler/confluent-trust/pull/66) | `docs(readme): refresh storage framing to Wyrd-as-DB (#59)` | open |
| [#67](https://github.com/JamesPagetButler/confluent-trust/pull/67) | `feat(testdata): predictions lifecycle fixture + schema ID-prefix docs (#53 part 2)` | open |
| [#68](https://github.com/JamesPagetButler/confluent-trust/pull/68) | `feat(cmd): cth score CLI subcommand (#53 part 3)` | open (stacked on #67) |

---

## 4. New CTH issues filed during the arc

| Issue | Title | Milestone |
|---|---|---|
| [#53](https://github.com/JamesPagetButler/confluent-trust/issues/53) | `compute/scoring.go + predictions/ schema + cth score CLI` (A18 §2.4 glue) | v0.1.x — Scoring Complete (this meeting's deliverable) |
| [#60](https://github.com/JamesPagetButler/confluent-trust/issues/60) | `doc: CTH ↔ Wyrd schema bridging — federation-wide Wyrd v0.2 gate prerequisite` | v0.2 — Walk Gates |
| [#61](https://github.com/JamesPagetButler/confluent-trust/issues/61) | `feat: cart-tool registration — \`cth\` as Theory Cart tool for Toddle BMA invocation` | v0.1.x — Scoring Complete |

Plus existing CTH issues reframed:

- **#51** (live inventory update API) — promoted from "queued behind #58" to **Toddle-load-bearing** after the toddle-design meeting confirmed CTH is L3 Beliefs substrate at Toddle. Closed via PR #65.
- **#18 + #26** (MuninnDB store / branch-locked vault) — body framing refreshed via comments: MuninnDB is the engram subsystem within Wyrd at Walk, not a separate DB.

---

## 5. Federation contract surface CTH must honor

These contracts originate in sibling repos but bind CTH:

| Contract | Source | CTH-side obligation |
|---|---|---|
| `CTHAnchor.AnchorID = "PRED-*"` prefix | Wyrd PR #35 §4.1 | CTH `model.Anchor.ID` uses "PRED-" prefix for predictions; convention documented in schema via PR #67 |
| Hook-semantics §2.1 (Append fires with `before == nil`; chain/confluence hooks fire on all field changes) | CTH PR #64 (v0.2 clarifications) | `store.LiveInventory` implements per the spec; verified by 15 race-tested cases in PR #65 |
| `cth.scoring.{anchor_id}.score_event` NATS topic | D14 + CTH #19 | Topic structure carved into #19; caller wires via `Hooks.OnAnchorChange` per PR #65 §6 of design |
| `cth-derivation` membership predicate (Contextus reads CTH closure) | Contextus PR #11 v1.4 | Unidirectional — Contextus reads CTH inventory state via Hooks/NATS; CTH never reads back (§8.3 invariant preserved; confirmed via [PR #11 comment](https://github.com/JamesPagetButler/contextus/pull/11#issuecomment-4456053011)) |
| Wyrd v0.2 substrate (`Node.TierImmune` + `Node.Salience` + `Graph.SetRetentionCap`) | Wyrd PR #39 (W-Toddle-1) | Walk-cutover via #60 schema bridging maps CTH `model.Axiom` → `Node.TierImmune=true`; documented via [PR #39 §I4 review comment](https://github.com/JamesPagetButler/wyrd/pull/39) |

---

## 6. Open follow-ups (post-handoff queue)

1. **PRs #66, #67, #68** in James's review queue — modernization + linting clean; awaiting merge.
2. **Milestone v0.1.x — Scoring Complete** closes when #67 + #68 merge.
3. **Milestone v0.2 — Walk Gates** carries #54 (`cth lean-link`), #55 (`cth manifest`), #56 (INST-* schema), #60 (Wyrd↔CTH schema bridging doc). Walk-blocking, not Toddle-blocking.
4. **CTH #61** (cart-tool registration) — Loop-1 Reference status; awaiting beekeeper/qbp-architecture ratification of "Option (a)" path (`cth` as Theory Cart tool via Engineering Cart pod-helper at Toddle; harness-resident at Walk).
5. **Contextus PR #11 v1.4 §2.4 Referent companion PR** — not yet opened; will cite Wyrd PR #35's CTHAnchor contract when it lands; one CTH-side flag surfaced ([PR #11 comment](https://github.com/JamesPagetButler/contextus/pull/11#issuecomment-4456053011) §"three OQ leans"): closure-walk should explicitly traverse `ConfluencePoint.paths[].chain_id` references, not just direct derivation edges.

---

## 7. Context for the next worker

If you (a future cth-implementor session) are picking this up:

- **Sessionbridge channels of record** going forward: per-PR threads + `#live-test` for cross-instance coordination. `#addendum-18-walk` and `#toddle-design` are retired.
- **CTH is L3 Beliefs substrate at Toddle** (workspace-phase-architecture §2.4). Treat `store.LiveInventory.Hooks.OnAnchorChange` and the NATS `cth.scoring.*` topic as production federation contracts — changes to their shape are P0 federation-coordination decisions.
- **Don't propagate stale framing**: MuninnDB is NOT a separate database (engram subsystem within Wyrd at Walk); SurrealDB references are extinct in CTH at every phase; Wyrd is the workspace-canonical DB (PR #66 README refresh tracks this if not yet merged).
- **Federation-additive-only contract**: every cross-repo type change widens, never narrows. If you find yourself wanting to remove a field, file an issue and discuss before acting.
- **Branch protection**: every PR merges via James (the beekeeper). Branch-protect is on for all 6 federation repos per addendum-18-walk Q7=A.
- **Workflow**: PLAN → branch off main → implement → tests green + lint clean → commit + push → open PR → James reviews + merges → verify acceptance criteria via closing-evidence comment on the issue.

---

## Cross-references

- BMA Theory Addendum 18 v0.1: `~/Documents/BMA/theory/hypergraph-inference/BMA-Theory-Addendum-18_0-Hypergraph-Access-Pattern.md`
- A18 v0.2 design surface: `~/Documents/BMA/theory/hypergraph-inference/A18-v0.2-design-surface.md`
- Workspace phase architecture: `~/Documents/inter/workspace-phase-architecture.md` (§0.10 cart-driven tools, §0.11 η = ρ_net, §2.4 L3 Beliefs substrate, §2.7 Toddle→Walk exit gates)
- Workspace roadmap: `~/Documents/inter/workspace-roadmap.md` §2.3 (CTH Crawl → Walk criteria)
- Prior CTH handoff: `~/Documents/BMA/doc/handoff/2026-05-01-cth-qbp-progress-update.md` (the bootstrap-era predecessor)
- CTH session-end protocol context: BMA project memory `feedback_pr_review` (post-PR acceptance-evidence rule)
