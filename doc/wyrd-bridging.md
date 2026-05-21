# CTH ↔ Wyrd Schema Bridging

**Status:** Design v0.1 — open for §I4 review per ADR-003 §I4
**Issue:** [CTH #60](https://github.com/JamesPagetButler/confluent-trust/issues/60) — doc: CTH ↔ Wyrd schema bridging — federation-wide Wyrd v0.2 gate prerequisite
**Walk-gate:** This document is the federation-wide prerequisite for the Toddle→Walk exit gate (workspace-phase-architecture §2.7, CTH-migration row). No `store/wyrd.go` adapter PR opens until §I4 sign-off lands here.

---

## §0 §I4 invariant — design-doc-as-S-01-review-surface

This document is the §I4 review surface for the CTH ↔ Wyrd schema bridging design.
Implementation PR for `store/wyrd.go` (the `OpenLiveInventoryWyrd` constructor) is blocked on
explicit sign-off from all named readers.

**Named readers (per D5):**

| Reader | Role |
|---|---|
| `@wyrd-implementor` | Substrate ownership. The forward mapping (§3) adds `cth.*` NodeType discriminators to Wyrd's registry; must confirm no collision with reserved `wyrd.` namespace and that the orientation patterns (§3.3) match Wyrd PR #31/#52 intent. |
| `@bma-implementor` | Consumer-side, consultative. CTH is BMA's L3 Beliefs substrate (workspace-phase-architecture §2.4); BMA's continuous cognitive cycle reads ρ_net on Wyrd-hosted anchors during Toddle endurance. Must confirm the inverse mapping (§5) covers NT_SEED + NT_INSIGHT_SIGNAL cases. |
| `@contextus-impl` | NT_INSIGHT_SIGNAL inverse-mapping angle (§5). Contextus PR #11 Spec v1.4 `cth-derivation` membership predicate traverses proof anchors; the inverse-mapping contract governs when Wyrd-resident NT_INSIGHT_SIGNAL nodes project as Tier 2 vs Tier 3 CTH anchors. |
| `@qbp-implementor` | Federation programme. CTH inventory consumer at Walk; QBP PRED-* anchors map to Wyrd-resident `cth.anchor` nodes at Walk-cutover. Must confirm the `predictions.CTHAnchor.AnchorID` round-trip (§5) holds. |
| `@beekeeper` | Final approval. |

---

## §1 Motivation

### §1.1 Walk-gate context

Workspace-phase-architecture **§2.7 Toddle→Walk exit gate** lists "Federation-wide Wyrd v0.2 — CTH + Contextus + QBP-CU migrated to Wyrd-backed storage" as a hard gate. CTH's row requires a Wyrd-side query view that exposes CTH-shaped data (Anchor / Chain / ConfluencePoint / ForkPoint) to CTH's compute primitives and a substrate adapter that writes CTH model constructs to Wyrd hypergraph nodes and oriented edges.

Workspace-phase-architecture **§0.11**: *"Trust anchors are CTH anchors. η (Systema vocab) = CTH ρ_net (Theory v2.0 vocab). Same concept; two names."* This means BMA's L3 Beliefs cognitive layer must read ρ_net on Wyrd-hosted anchors during the 7-day Toddle endurance, with the actual migration completing at Walk. Without a structural bridging document, the `OpenLiveInventoryWyrd` constructor (see `doc/design/live-inventory-api.md` §7) has no schema contract to implement against.

### §1.2 The Walk-cutover problem

At Crawl/Toddle, CTH operates on JSON files. `store.LoadInventory` / `store.SaveInventory` read and write `model.Inventory` from a single JSON path. At Walk, the inventory lives in a Wyrd hypergraph. The structural translation challenge:

- CTH's model constructs carry derivation-depth semantics (Tier 0–3), provenance, and status.
- Wyrd's `model.Node` carries algebraic-tier semantics (Cayley-Dickson: Complex/Quaternion/Octonion/Sedenion), retention semantics (RetentionTier: Skeleton..Core), and a payload blob.
- CTH's `model.Chain` (source_ids → target_id) maps naturally to Wyrd's oriented `Hyperedge` (Heads = upstream, Tails = downstream), but only because Wyrd PR #31/#52 landed oriented-edge support. Without this PR the mapping was blocked.
- CTH's N-ary constructs (ConfluencePoint with N paths, ForkPoint with N branches) require Wyrd's k-arity hyperedge to avoid lossy decomposition into pairs (per Wyrd `model/hyperedge.go` irreducibility theorems `Wyrd.HolographicHypergraph.theorem2_irreducibility`).

This document maps the structural translation in both directions.

### §1.3 BMA Theory v3.0 §3 — Notary function

BMA Theory v3.0 §3.1 specifies the `NT_NOTARY_VERIFICATION_EVIDENCE` node type: a Wyrd-resident artifact produced when the Notary verifies a Lean proof and posts the verification record into the hypergraph. The pairing with CTH's `Anchor.Verification` (`*model.VerificationRecord`, v0.3 field) is load-bearing:

> `Anchor.Verification` (CTH-resident) ↔ `NT_NOTARY_VERIFICATION_EVIDENCE` (Wyrd-resident)

These are the **same evidence at different federation scopes** — CTH-resident for inventory querying, Wyrd-resident for federation-wide notarization. The bridging document must make this pairing explicit (see §3.7).

---

## §2 Wyrd substrate inventory

The following Wyrd primitives are consumed by this mapping. All paths are relative to `~/Documents/Wyrd/`.

| Wyrd primitive | Source | What it provides |
|---|---|---|
| `model.Node` | `model/node.go` | Base hypergraph node with ID, Type (`NodeType` string), Tier (algebraic), Created, Payload (`[]byte`, opaque to Wyrd), TierImmune, Salience |
| `model.NodeType` | `model/node.go` | Free-form string discriminator; `wyrd.` prefix reserved for Wyrd-internal types; consumers choose their own prefix |
| `model.Node.TierImmune` | Wyrd PR #39 (W-Toddle-1); `model/node.go` | Permanence flag — true = exempt from all eviction paths (cap-per-tier saturation, sleep-cycle compaction) |
| `model.Node.Salience` | Wyrd PR #39 (W-Toddle-1); `model/node.go` | Retention-priority modulator, range 0.0..1.0; 0.0 default; Hebbian co-activation increments, Ebbinghaus decay decrements |
| `model.Hyperedge` | Wyrd PR #31/#52; `model/hyperedge.go` | Edge with `Nodes []NodeID` (all participants), `Heads []int` (upstream indices), `Tails []int` (downstream indices); `IsSymmetric` bool; `Payload []byte` for consumer metadata |
| `model.RetentionTier` | Wyrd PR #39 (W-Toddle-1); `model/retention.go` | Typed enum (Skeleton / Distant / Peripheral / Near / Core) for cap-per-tier eviction policy |
| `Graph.SetRetentionCap(RetentionTier, cap)` | Wyrd PR #39 | Per-tier eviction cap; `cap == 0` disables eviction at that tier |
| `predictions.Prediction` + `predictions.CTHAnchor` | Wyrd PR #35 `predictions/predictions.go` | NT_SIGNAL prediction record (as `model.Node.Payload`) with optional `CTHAnchor.AnchorID = "PRED-*"` federation stamp |

---

## §3 Forward mapping — CTH constructs → Wyrd primitives

One subsection per CTH model construct. For each:
- Wyrd `Node.Type` discriminator string (these widen the NodeType registry; see §6)
- Which payload fields are carried structurally
- Which fields are computed-not-stored
- Tier-axis disambiguation (§4 covers the three axes in full)

### §3.1 `model.Axiom` → `cth.axiom` node

```
Node{
  Type:       "cth.axiom",
  TierImmune: true,
  Salience:   1.0,
  Tier:       TierComplex,          // algebraic tier; see §4 disambiguation
  Payload:    JSON(axiom-fields),
}
```

**Payload fields carried:** `id`, `name`, `statement`, `derivable`, `derived_from_axioms` (if `derivable == true`), `inherited_from` (if present), `layer`, `notes`.

**TierImmune rationale:** Axioms are the underivable bedrock of a programme. Deletion invalidates downstream invariants. Same semantics as BMA NT_SEED nodes (seed protocol Step 9) — permanent, never evicted. `Salience: 1.0` places axioms at maximum retention priority under any pressure.

**Computed-not-stored:** None for Axiom. The `Validate()` invariant (derivable=true ↔ non-empty `derived_from_axioms`) runs at write time; the Wyrd-resident payload always carries the validated form.

**Algebraic-tier choice:** `TierComplex` (user-facing algebra) is the default for CTH constructs unless a consumer requires a higher-privilege tier for capability-gating. See §4.

### §3.2 `model.DerivedPrinciple` → `cth.derived_principle` node + derivation edge

```
Node{
  Type:    "cth.derived_principle",
  Tier:    TierComplex,
  Payload: JSON(derived-principle-fields),
}

Hyperedge{
  Nodes:       [derivedPrincipleNodeID, from1NodeID, from2NodeID, ...],
  Heads:       [0],                // index of derivedPrincipleNodeID in Nodes
  Tails:       [1, 2, ...],        // indices of source axiom/anchor NodeIDs in Nodes
  IsSymmetric: false,
  Payload:     JSON({type: "cth.derivation", derived_principle_id: "DERIV-*"}),
}
```

**Payload fields carried:** `id`, `name`, `statement`, `derived_from` ([]string — source axiom/anchor IDs; also encodes in Hyperedge.Tails for graph traversal), `layer`.

**Hyperedge orientation:** Heads = derived output; Tails = input sources. CTH opcode flow convention per `model/hyperedge.go` §3: "Heads = upstream, Tails = downstream" — the DerivedPrinciple IS downstream from its sources in derivation order.

**Computed-not-stored:** None. `DerivedFrom` encodes redundantly in both the Payload (for self-contained reads) and the Hyperedge.Tails (for graph traversal). Both must be kept consistent on write.

### §3.3 `model.Anchor` → `cth.anchor` node + optional prediction-chain edges

```
Node{
  Type:    "cth.anchor",
  Tier:    TierComplex,
  Payload: JSON(anchor-fields),
}
```

**Payload fields carried (v0.2 + v0.3):**
- Core identity: `id`, `name`, `description`
- CTH tier: `tier` (int 0-3; stored in Payload.cth_tier, NOT in `model.Node.Tier` — see §4)
- Provenance (v0.2 legacy): `provenance` (T/E/H)
- Provenance (v0.3 fine-grained): `provenance_kind` (proof / theory / theory-external / experiment / hypothesis / internal-compute / philosophy)
- Status: `status` (coherent / untested / incoherent / contested / refuted / killed / marginal / converged / falsified)
- Proof state (v0.3): `proof_state` (verified / partial / written / null), `proof_language`, `proof_file`
- Theorems (v0.3): `theorems` ([]TheoremRef — name, status, blockers; stored in Payload; no separate Wyrd node per §3.8)
- Verification (v0.3): `verification` (*VerificationRecord — toolchain, libraries, verified_at, verifier, result; stored in Payload; see §3.7 for federation pairing)
- Additional verifications (v0.3): `additional_verifications`
- Prediction fields: `predicted_value`, `predicted_unit`, `testable_when`
- Measurement fields: `measured_value`, `measured_error`, `measured_source`, `last_tested_at`, `discrepancy_pct`
- Derivation fields: `inherited_from`, `inherited_at`, `bridge_role`, `branch_id`
- Citation fields: `theory_citation`, `theory_doi`, `theory_url`
- Notes: `notes`

**Prediction chain edges:** When `Anchor.PredictionChain` is non-empty, each element is a CTH anchor ID that this anchor depends on for prediction derivation. Encode as a symmetric `Hyperedge` (unordered participation) or as a sequence of binary oriented edges Heads=[this] → Tails=[upstream], per local convention. A **single oriented hyperedge** encoding all prediction_chain sources to this anchor Tail is also valid:

```
Hyperedge{
  Nodes:       [thisAnchorNodeID, pred1NodeID, pred2NodeID, ...],
  Heads:       [1, 2, ...],        // upstream prediction sources
  Tails:       [0],                // this anchor as sink
  IsSymmetric: false,
  Payload:     JSON({type: "cth.prediction_chain", anchor_id: "PRED-*"}),
}
```

**Computed-not-stored:** `Domain`, `ResidualEntropy`, `ConfirmatoryInfo` are produced by `package compute` on demand (per `model/anchor.go` doc comment). These MUST NOT be stored in the Wyrd payload. At Walk, consumers call `compute.NetCompressionDetail(snapshot)` on an `Inventory` constructed from a Wyrd graph snapshot.

**TierImmune:** Not set for anchors by default. Exception: if an anchor is marked `provenance_kind: "proof"` and `proof_state: "verified"` with a substrate-tier theorem (per BMA Spec v9.1 §15 promotion gate), `TierImmune` MAY be set true by the `store/wyrd.go` adapter on write; this is a Walk-α policy decision deferred to the adapter implementation.

### §3.4 `model.Chain` → oriented `cth.chain` hyperedge

CTH's Chain is structurally a hyperedge, not a node. Encode as:

```
Hyperedge{
  Nodes:       [targetNodeID, source1NodeID, source2NodeID, ...],
  Heads:       [1, 2, ...],        // source anchor NodeIDs (upstream)
  Tails:       [0],                // target anchor NodeID (downstream)
  IsSymmetric: false,
  Payload:     JSON(chain-fields),
}
```

**Payload fields carried:** `id`, `name`, `steps`, `step_types`, `status`, `fidelity` (if set), `weakest_link_id` (if set), `domain_boundaries` ([]DomainBoundary — from_domain, to_domain, at_anchor_id, hypothesis, fidelity), `notes`.

**Orientation convention:** Heads = upstream sources (evidence flows FROM); Tails = downstream target (derivation flows TO). This maps directly to CTH's `SourceIDs → TargetID` relationship.

**N-ary source support:** Multiple `SourceIDs` in CTH map to multiple Heads indices. Wyrd's irreducibility theorems (`Wyrd.HolographicHypergraph.theorem2_irreducibility` and `Wyrd.HolographicHypergraphHigherArity.theorem2_irreducibility_n_arity`) guarantee that a k-source chain carries information that CANNOT be recovered by decomposing into binary edges. Consumers MUST use `Graph.AddHyperedge` with all sources at once, NOT split into pairs.

**Computed-not-stored:** `Fidelity` when computed from `StepTypes` (the `WeakestLinkID` heuristic path). Only store `Fidelity` in the payload when it was explicitly set in the source CTH inventory.

### §3.5 `model.ConfluencePoint` → `cth.confluence` node + N incoming chain hyperedges

```
Node{
  Type:    "cth.confluence",
  Tier:    TierComplex,
  Payload: JSON(confluence-fields),
}
```

**Payload fields carried:** `id`, `anchor_id`, `description`, `paths` ([]ChainRef — chain_id, programme, summary, provenance, fidelity), `mutual_info_bits`, `status`.

**Graph structure:** Each `paths[i]` chain hyperedge (from §3.4) has its Tails index pointing at the confluence's associated `anchor_id` node. The ConfluencePoint node itself is a structural marker; the N-ary convergence is expressed by N chain hyperedges sharing the same Tails target (the referenced anchor). The ConfluencePoint node links to the shared anchor via a separate oriented hyperedge:

```
Hyperedge{
  Nodes:       [confluenceNodeID, anchorNodeID],
  Heads:       [0],   // confluence node references the anchor
  Tails:       [1],   // anchor is the convergence target
  IsSymmetric: false,
  Payload:     JSON({type: "cth.confluence_ref", confluence_id: "CONV-*"}),
}
```

**LegacyPathA/LegacyPathB:** Pre-v0.2 `path_a` / `path_b` fields are normalized via `NormalizePaths()` before writing to Wyrd; only the `paths []ChainRef` form is stored.

**Computed-not-stored:** `MutualInfoBits` is a computed quantity (per Theory v0.2 §4 confluence semantics); it SHOULD be stored in the payload as a cached value, clearly tagged as computed. Recomputation from chain fidelities is authoritative; the stored value is for display/query efficiency.

### §3.6 `model.ForkPoint` → `cth.fork` node + per-branch anchor tagging

```
Node{
  Type:    "cth.fork",
  Tier:    TierComplex,
  Payload: JSON(fork-fields),
}
```

**Payload fields carried:** `id`, `branch_node_id`, `question`, `shared_prefix` ([]string — anchor IDs in the shared prefix), `branches` ([]Branch — id, name, hypothesis, anchors, chains, confluences, inputs, predictions, burden), `branch_observations` ([]BranchObservation — anchor_id, interpretations).

**Per-branch anchor tagging:** Branch-specific anchors are identified by their `Anchor.BranchID` field (see `model/anchor.go`). No additional Wyrd structure is needed; `BranchID` encodes in the anchor payload (§3.3), enabling graph queries to filter by branch. A query for "all anchors on branch B" is: find `cth.anchor` nodes where `Payload.branch_id == B`.

**Branch-membership edges (optional):** For graph-traversal efficiency, the adapter MAY emit membership hyperedges:

```
Hyperedge{
  Nodes:       [forkNodeID, branchAnchorNodeID],
  Heads:       [0],   // fork references branch anchors
  Tails:       [1],
  IsSymmetric: false,
  Payload:     JSON({type: "cth.fork_branch", branch_id: "B"}),
}
```

This is an optimization, not a structural requirement. The payload encoding is authoritative.

### §3.7 `model.VerificationRecord` (v0.3) — federation pairing with `NT_NOTARY_VERIFICATION_EVIDENCE`

`model.VerificationRecord` is stored **inside** the `cth.anchor` Payload field — it is not a separate Wyrd node. Payload structure within the anchor:

```json
"verification": {
  "toolchain": "lean4@v4.3.0",
  "libraries": {"Mathlib": {"ref": "v4.3.0", "sha": "abc123..."}},
  "verified_at": "2026-05-21T10:00:00Z",
  "verifier": "Notary-v0.1",
  "result": "ok"
}
```

**Federation pairing (BMA Theory v3.0 §3.1):** When a CTH anchor's verification is produced via `cth lean-link` (CTH #54), the equivalent Wyrd-resident artifact is `NT_NOTARY_VERIFICATION_EVIDENCE`. These are the **same evidence at different federation scopes**:

| Artifact | Location | Scope | Consumer |
|---|---|---|---|
| `Anchor.Verification` (`*VerificationRecord`) | CTH inventory (JSON at Toddle; `cth.anchor` Payload at Walk) | CTH programme inventory | `cth analyse`, `compute.NetCompressionDetail`, L3 Beliefs ρ_net |
| `NT_NOTARY_VERIFICATION_EVIDENCE` node | Wyrd hypergraph | Federation-wide | BMA Notary function, federation audit log |

At Walk, when the `store/wyrd.go` adapter writes a `cth.anchor` node, it MUST also check whether a corresponding `NT_NOTARY_VERIFICATION_EVIDENCE` node exists in the graph (by cross-referencing `Anchor.ID`) and, if so, link them via a cross-reference hyperedge. The link is informational (not a data dependency); the two artifacts are independently authoritative.

**BMA Spec v9.1 §18.6.5 privacy classification:** `NT_NOTARY_VERIFICATION_EVIDENCE` artifacts are **Community tier** (federation-shared evidence; preserves Notary identity). The corresponding `cth.anchor` Payload with `verification` set carries the same evidence in CTH's scope; when CTH adds `Anchor.Privacy` at v0.4 (see §6), the equivalent classification is `PrivacyTierCommunity`.

### §3.8 `model.TheoremRef` (v0.3) — payload-only encoding

`TheoremRef` records (`name`, `status`, `blockers`) are stored **inside** the `cth.anchor` Payload under the `theorems` array field. No separate Wyrd node is created for individual theorem references; they are inventory-internal and do not require independent graph traversal.

**Rationale:** Theorem references are tightly coupled to their parent anchor; their lifecycle is the anchor's lifecycle. The CTH `Inventory.Validate()` enforces the `proof_state == verified ↔ all theorems verified` invariant at write time. Storing theorems as separate nodes would introduce orphan-detection complexity with no graph-traversal benefit.

### §3.9 `model.Input` → `cth.input` node

```
Node{
  Type:    "cth.input",
  Tier:    TierComplex,
  Payload: JSON(input-fields),
}
```

**Payload fields carried:** `id`, `name`, `type` (always `"input"`), `status`, `significant_figures`, `notes`.

**Computed-not-stored:** Entropy contribution `3.32 * SignificantFigures` bits is computed by `package compute` on demand. MUST NOT be stored in Payload.

**Irreducibility note:** Inputs are "eddies in the river" (Theory §4.6) — underived parameters. They are never heads of a derivation chain; only downstream chain hyperedges reference them as Tails sources.

---

## §4 Tier-axis disambiguation (CRITICAL)

Three independent tier axes coexist in the federation. Conflating them is a semantic error. This section documents each axis and their orthogonal relationship.

### §4.1 CTH `model.Tier` (derivation-depth axis)

- **Type:** `int8` (values 0–3)
- **Defined in:** `model/enums.go` (`TierAxiom = 0`, `TierProof = 1`, `TierMeasurement = 2`, `TierPrediction = 3`)
- **Semantics:** Position in the derivation hierarchy. Tier 0 = underivable assumptions (Axiom); Tier 1 = theoretical proofs and derived principles; Tier 2 = experimental measurements; Tier 3 = predictions awaiting evidence.
- **Governs:** ρ_net computation, `compute.NetCompressionDetail`, `Inventory.Health` scoring.
- **Storage in Wyrd:** **Stored in `cth.anchor` / `cth.axiom` Payload as `cth_tier`**. NOT stored in `model.Node.Tier`. CTH derivation-depth is orthogonal to Wyrd algebraic-tier; forcing CTH's int 0-3 into Wyrd's `TierComplex`/`TierQuaternion`/`TierOctonion`/`TierSedenion` would be a semantic error (the value sets are not aligned: CTH Tier 3 ≠ Wyrd `TierSedenion`).

### §4.2 Wyrd `model.Tier` (algebraic-privilege axis)

- **Type:** `int` iota (values 0–3)
- **Defined in:** Wyrd `model/tier.go` (`TierComplex = 0`, `TierQuaternion = 1`, `TierOctonion = 2`, `TierSedenion = 3`)
- **Semantics:** Position in the Cayley-Dickson algebraic privilege tower (ℂ ⊂ ℍ ⊂ 𝕆 ⊂ 𝕊). Governs which processes may author edges referencing the node (capability-gating per `Wyrd.Capability.capability_grants_safe_access`). Operations in a lower tier may READ nodes at higher tiers (downward projection is safe per `Wyrd.Projection.kernel_supervisor_safe`) but may not AUTHOR edges without sufficient tier capability.
- **Governs:** Wyrd capability model; compute tier; algebra-aware operations in Wyrd.
- **For CTH nodes:** All CTH-originated nodes default to `TierComplex` (user-facing algebra). A Walk-phase policy decision (deferred to the `store/wyrd.go` adapter) may choose higher tiers for specific constructs requiring tighter capability gating.

### §4.3 Wyrd `model.RetentionTier` (eviction-policy axis)

- **Type:** `uint8` iota (values 0–4)
- **Defined in:** Wyrd `model/retention.go` (`RetentionSkeleton = 0`, `RetentionDistant = 1`, `RetentionPeripheral = 2`, `RetentionNear = 3`, `RetentionCore = 4`)
- **Semantics:** Contextus Spec v1.3 §9.1 retention-tier axis. Governs per-tier eviction caps set via `Graph.SetRetentionCap(RetentionTier, cap)`. Eviction order under saturation: `TierImmune == false` nodes only; ascending Salience evicted first within each retention tier.
- **Governs:** Eviction policy, sleep-cycle compaction, BMA Ebbinghaus decay.
- **For CTH nodes:** `cth.axiom` nodes are `TierImmune = true`, bypassing retention-tier eviction entirely. All other CTH node types (`cth.anchor`, `cth.chain` payload nodes, etc.) default to no explicit `RetentionTier` assignment. Walk-phase policy (deferred) may assign `RetentionCore` to high-ρ_net anchors or `RetentionSkeleton` to archived prediction anchors.
- **NOT stored on `model.Node`:** `RetentionTier` is a graph-level policy parameter, not a per-node field. `model.Node` carries `Salience` (per-node retention-priority signal); `Graph.SetRetentionCap` sets the tier caps. The retained-tier assignment for a node is implicit in how the eviction policy fires.

### §4.4 Summary table

| Axis | Values | Type | Governs | Stored where |
|---|---|---|---|---|
| CTH `model.Tier` | 0–3 (Axiom/Proof/Measurement/Prediction) | `int8` | Derivation depth, ρ_net computation | `cth_tier` in `model.Node.Payload` |
| Wyrd algebraic `model.Tier` | TierComplex..TierSedenion | `int` iota | Capability gating, algebra-aware ops | `model.Node.Tier` field |
| Wyrd `model.RetentionTier` | Skeleton..Core | `uint8` iota | Per-tier eviction caps | Graph-level policy via `Graph.SetRetentionCap` |

This disambiguation was a finding from the Wyrd PR #39 §I4 consultative read (cth-implementor); it is affirmed and documented here as the authoritative forward reference. The three axes coexist orthogonally.

---

## §5 Inverse mapping — foreign hypergraph state → CTH constructs

At Walk-cutover, when CTH reads a Wyrd-hosted hypergraph (via `OpenLiveInventoryWyrd` or `cth analyse <wyrd-graph>`), foreign Wyrd node types project into CTH model constructs as follows.

### §5.1 Forward-mapped CTH nodes (round-trip)

Nodes originally written by CTH's `store/wyrd.go` adapter (§3) carry `cth.*` type discriminators and reconstruct cleanly:

| Wyrd `Node.Type` | CTH construct | Notes |
|---|---|---|
| `"cth.axiom"` | `model.Axiom` | Deserialize Payload JSON into Axiom struct; run `Axiom.Validate()` |
| `"cth.derived_principle"` | `model.DerivedPrinciple` | Deserialize Payload; reconstruct `DerivedFrom` from Payload field (hyperedge Tails are redundant for round-trip) |
| `"cth.anchor"` | `model.Anchor` | Deserialize Payload; all v0.3 fields included; run `Anchor.Validate()` |
| `"cth.input"` | `model.Input` | Deserialize Payload; run `Input.Validate()` |
| `"cth.confluence"` | `model.ConfluencePoint` | Deserialize Payload; `NormalizePaths()` not needed (v0.2 form already stored) |
| `"cth.fork"` | `model.ForkPoint` | Deserialize Payload; run `ForkPoint.Validate()` |
| `cth.chain` hyperedge | `model.Chain` | Deserialize Hyperedge.Payload; reconstruct `SourceIDs` from `Heads` NodeIDs; `TargetID` from `Tails[0]` NodeID; run `Chain.Validate()` |

### §5.2 BMA-originated nodes → CTH constructs

| Foreign Wyrd shape | CTH model construct | Projection logic |
|---|---|---|
| `Node{Type: "NT_SEED", TierImmune: true, Salience: 1.0}` | `model.Axiom` | BMA seed-protocol Step-9 NT_SEED nodes are constitutional axioms in CTH terms. Map: `Axiom.ID = node.ID`, `Axiom.Statement = Payload["content"]` (or BMA-equivalent field), `Axiom.Derivable = false`. The `TierImmune + Salience=1.0` combination is the structural fingerprint of an NT_SEED. |
| `predictions.Prediction` with `CTHAnchor.AnchorID = "PRED-*"` | `model.Anchor` (already federation-stamped) | Per Wyrd PR #35 §4.1 — `CTHAnchor.AnchorID` IS the CTH anchor.ID. No further mapping needed; the prediction was authored with CTH awareness. Round-trip: anchor already exists in CTH inventory; the Wyrd-resident Prediction carries the `ObservedValue`/`Score` fields that may update the CTH anchor's measurement fields via `UpdateAnchor`. |

### §5.3 Contextus-originated nodes → CTH constructs

| Foreign Wyrd shape | CTH model construct | Projection logic |
|---|---|---|
| `Node{Type: "NT_INSIGHT_SIGNAL", RetentionTier <Spec v1.3 §5.4 mapping>}` | `model.Anchor` | Tier depends on `ProvenanceKind` carried in Payload. Default projection: Tier 2 (measurement) when the signal has an observed value; Tier 3 (prediction) when `ObservedValue == nil`. The Contextus retention-tier (Core vs Peripheral) informs CTH's evictability assumption at Walk but does NOT determine the CTH derivation-depth tier. |

### §5.4 Oriented hyperedge → `model.Chain` (round-trip from §3.4)

```
Hyperedge{IsSymmetric: false, Heads: [...], Tails: [0], Payload: {type: "cth.chain", ...}}
```

Projects to `model.Chain` by:
1. `TargetID = Nodes[Tails[0]]` (cast to string)
2. `SourceIDs = [Nodes[h] for h in Heads]` (cast to strings)
3. Remaining Chain fields from `Hyperedge.Payload` JSON

### §5.5 Projection schema version guard

The `store/wyrd.go` adapter MUST embed a schema version in every `cth.*` node Payload (e.g., `"cth_schema_version": "0.3"`). On inverse-read, if the version is absent or older than the current CTH model, the adapter MUST apply forward-compatibility defaults (omitempty fields → zero values) and log a warning, never fail silently.

---

## §6 Federation-additive-only contract

Per the federation discipline (cth-implementor §5.c §2.8 + Contextus PR #12 precedent + BMA Spec v9.1 §15.5): every cross-repo type change widens, never narrows. The bridging mapping enforces:

### §6.1 Additive on Wyrd side

New `Node.Type` discriminators introduced by CTH widen Wyrd's NodeType registry:

| New `NodeType` | Where used |
|---|---|
| `"cth.axiom"` | §3.1 |
| `"cth.derived_principle"` | §3.2 |
| `"cth.anchor"` | §3.3 |
| `"cth.input"` | §3.9 |
| `"cth.confluence"` | §3.5 |
| `"cth.fork"` | §3.6 |
| `"cth.chain"` | Hyperedge Payload type discriminator (§3.4) |

None of these collide with Wyrd's reserved `wyrd.` prefix. None narrow or modify existing Wyrd primitives. `@wyrd-implementor` must confirm no collision with BMA's `bma.` or Contextus's `contextus.` prefixes.

### §6.2 Additive on CTH side

The Walk-phase `store/wyrd.go` adapter will require a cross-reference field for round-trip provenance. This is a **v0.4 candidate**, not required for this document:

```go
// v0.4 candidate addition to model.Anchor (additive; optional):
WyrdNodeID string `json:"wyrd_node_id,omitempty"`  // cross-ref to Wyrd-resident cth.anchor node
```

This field widens `model.Anchor` without breaking existing JSON inventories (omitempty). It is flagged here for the adapter implementation to plan around.

### §6.3 Privacy field forward-compat

**Status: FORWARD-COMPAT FLAG for v0.4.**

Per BMA Spec v9.1 §18 (Privacy-Tier Schema), `HGNode.Privacy` is a first-class field on BMA hypergraph nodes with four canonical values: `Constitutional` / `Community` / `Operational` / `Private`.

When CTH adds `Anchor.Privacy PrivacyTier` at v0.4 (per inter PR #31 §I4 review flag), the Wyrd-resident counterpart on `cth.anchor` nodes MUST set `Node.Privacy` (once Wyrd adds this field) correspondingly. Anticipated mapping:

| CTH anchor type | Expected `Anchor.Privacy` | `Node.Privacy` on `cth.anchor` |
|---|---|---|
| Substrate-tier verified theorem (proof_state: verified + substrate promotion per §15) | `Constitutional` | `Constitutional` |
| Federation-shared community anchor (shared research artifact) | `Community` | `Community` |
| Operational anchor (in-progress reasoning, runtime metrics) | `Operational` (default) | `Operational` |
| Beekeeper-private anchor | `Private` | `Private` |

**§I4 flag for `@wyrd-implementor` and `@bma-implementor`:** Does Wyrd's `HGNode.Privacy` field (per Spec v9.1 §18.3.1) already exist on `model.Node`, or does the bridging doc need to track a Wyrd-side schema addition as a dependency? At time of authoring, Wyrd `model/node.go` does NOT carry a `Privacy` field — this is BMA-systema-specific (per bma-systema PR #190). The federation-side question is: does the privacy field belong on Wyrd's substrate `model.Node`, or does each tenant implement privacy filtering in their sync layer? Resolution deferred to inter PR #31 §I4 review. **This document does not prescribe the answer; it flags the open question.**

---

## §7 NATS event surface at the federation boundary

The `cth.scoring.{anchor_id}.score_event` topic (CTH #19, with `event_kind` discriminator per CTH #71 §C concern #7) is the federation-wide read-side for anchor-state changes. Per `doc/design/live-inventory-api.md` §6, the NATS publish is wired via `Hooks.OnAnchorChange` by the BMA caller, not by `LiveInventory` itself.

At Walk-cutover, Wyrd-resident mutations to `cth.*` nodes must produce the SAME topic events as CTH-resident mutations:

| Change source | Event path |
|---|---|
| CTH-resident change via `LiveInventory.UpdateAnchor` | `Hooks.OnAnchorChange` → NATS publish (BMA #107 wiring) |
| Wyrd-resident change to a `cth.anchor` node via `Graph.AddNodeWithCapability` (Walk-α `store/wyrd.go`) | `OpenLiveInventoryWyrd` must fire the SAME `Hooks.OnAnchorChange` callback so BMA L3 readers + Contextus `cth-derivation` closure cache do not fragment |

**Implementation note for Walk-α:** The `OpenLiveInventoryWyrd` constructor must internally register a Wyrd graph-change listener (mechanism TBD — Wyrd's Watch API surface if it exists, or a BMA-side polling loop) that fires `Hooks.OnAnchorChange` on mutations to `cth.*` nodes. The hook surface is substrate-agnostic; callers do not observe the difference. This is the substrate-swap promise from `doc/design/live-inventory-api.md` §7.

---

## §8 Implementation phases

| Phase | Substrate | CTH action | Status |
|---|---|---|---|
| **Crawl/Toddle** | JSON files | Mapping is documentation-only (this doc). `OpenLiveInventory(path, hooks)` operates on JSON. No Wyrd dependency. | Current |
| **Walk-α** | Wyrd hypergraph | CTH adds `store/wyrd.go` implementing `OpenLiveInventoryWyrd(graph *wyrd.Graph, programme string, hooks *Hooks)`. Uses forward mapping (§3) for `AppendAnchor` / `AppendChain` writes; inverse mapping (§5) for `Snapshot()` reads. `OpenLiveInventory` (JSON path) remains available for pre-Walk consumers. | Unblocked after this §I4 |
| **Walk+** | Dual (JSON + Wyrd) | Federation tenants migrate at their own pace. CTH's transitional dual-substrate reading keeps pre-Walk consumers functional. JSON substrate is deprecated-not-removed. | Future |

**Sequencing note (Ruling 1 from CTH #71):** The `store/wyrd.go` adapter is Walk-α scope. Crawl/Toddle work does not touch `store/wyrd.go`. This bridging document (doc-only) is the gate for the adapter PR; no Go code changes land with this PR.

---

## §9 v0.3 → v0.4 follow-ups (out of scope for this PR)

These open questions are raised by the mapping work and defer to v0.4:

1. **Privacy field synchronization** — `Anchor.Privacy` addition to CTH model (inter PR #31 §I4 flag); `Node.Privacy` on Wyrd-resident `cth.anchor` nodes; explicit `Anchor.Privacy → Node.Privacy` synchronization rule in `store/wyrd.go`. See §6.3.

2. **`cth resolve <anchor-id>` primitive** — Flagged in inter PR #31 review: enables federation CI to verify `scaffold_anchor` + `claim_record_anchor` resolve per BMA Spec v9.1 §16.5 (Research-Aid Protocol). Deferred; no structural dependency on this bridging doc.

3. **`model.Anchor.WyrdNodeID` cross-reference field** — v0.4 candidate (§6.2). Enables round-trip provenance without graph lookup for every anchor read. Deferred to keep v0.3 schema stable.

4. **Wyrd-side CTH query adapter** — Does `wyrd/query/` need a `cth.View` adapter providing CTH-typed result shapes, or does each consumer do field projection from raw `Node.Payload`? Deferred to the `store/wyrd.go` adapter design phase.

5. **Per-anchor `RetentionTier` assignment policy** — Walk-α: which CTH anchor types map to which Wyrd `RetentionTier`? High-ρ_net verified anchors → `RetentionCore`? Archived prediction anchors → `RetentionSkeleton`? Deferred; profile first.

---

## §10 Cross-references

| Reference | Context |
|---|---|
| [CTH #60](https://github.com/JamesPagetButler/confluent-trust/issues/60) | This issue |
| [CTH #54](https://github.com/JamesPagetButler/confluent-trust/issues/54) | `cth lean-link` — produces `VerificationRecord`s that map to `NT_NOTARY_VERIFICATION_EVIDENCE` (§3.7) |
| [CTH #71](https://github.com/JamesPagetButler/confluent-trust/issues/71) | v0.3 schema (ProvenanceKind / ProofState / VerificationRecord / TheoremRef) |
| `doc/design/inventory-schema-v0_3.md` | v0.3 field specifications and Ruling 1 sequencing |
| `doc/design/live-inventory-api.md` §7 | `OpenLiveInventoryWyrd` constructor contract (Walk-cutover substrate-swap path) |
| Wyrd PR #35 `scoutquery.md` | `predictions.Prediction.CTHAnchor` federation contract (§5.2) |
| Wyrd PR #39 `tier-immunity-salience.md` | W-Toddle-1: `TierImmune` + `Salience` + `SetRetentionCap` (§2, §4) |
| Wyrd PR #31/#52 `oriented-hyperedge.md` | Oriented-hyperedge schema (Heads/Tails indices); §3.4 Chain mapping |
| Wyrd `model/node.go` | `Node`, `NodeType`, `NodeID` |
| Wyrd `model/hyperedge.go` | `Hyperedge`, `Heads`/`Tails` indices, irreducibility theorems |
| Wyrd `model/tier.go` | Cayley-Dickson algebraic `Tier` (§4.2) |
| Wyrd `model/retention.go` | `RetentionTier` eviction-policy axis (§4.3) |
| Wyrd `predictions/predictions.go` | `Prediction`, `CTHAnchor` |
| BMA Spec v9.1 §15 | Federation Lean Promotion Protocol (substrate-tier theorem promotion gate; §3.3 TierImmune note) |
| BMA Spec v9.1 §18 | Privacy-Tier Schema (Constitutional/Community/Operational/Private; §6.3 forward-compat flag) |
| BMA Spec v9.1 §18.6.5 | `NT_NOTARY_VERIFICATION_EVIDENCE` privacy tier = Community (§3.7) |
| BMA Theory v3.0 §3.1 | Notary function; `NT_NOTARY_VERIFICATION_EVIDENCE` ↔ `Anchor.Verification` pairing (§3.7) |
| Contextus PR #11 | Spec v1.4 theory-as-scope — `cth-derivation` membership predicate; §5.3 NT_INSIGHT_SIGNAL mapping |
| workspace-phase-architecture §0.11 | η = CTH ρ_net trust-anchor identity |
| workspace-phase-architecture §2.7 | Toddle→Walk exit gate (CTH-migration row) |
| workspace-roadmap §2.3 | CTH Crawl→Walk criteria |

---

*Status: DRAFT v0.1 — open for §I4 review. Walk-α `store/wyrd.go` implementation PR blocked on explicit sign-off from all named readers in §0.*
