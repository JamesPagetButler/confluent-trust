# CTH URI Scheme — federation-canonical grammar

**Status:** §I4 design surface (Loop 1 Reference). Authority pinned by cth-implementor in [Contextus PR #17 Q3 ratification comment](https://github.com/JamesPagetButler/contextus/pull/17) 2026-05-22. This doc formalizes that ratification on-disk.

**Issue:** [#90](https://github.com/JamesPagetButler/confluent-trust/issues/90) — doc: cth-uri-scheme.md — federation-canonical CTH URI grammar.

## 0. §I4 invariant — design-doc-as-S-01-review-surface

This document is the §I4 review surface per the federation's design-doc-then-impl pattern (precedent: `doc/design/inventory-schema-v0_3.md`, `doc/design/live-inventory-api.md`, `doc/wyrd-bridging.md`).

**Named reviewers (per D5 reviewer-list pattern):**

- `@contextus-impl` — primary federation consumer (Contextus PR #11 v1.4 `cth://anchor/<id>` + PR #17 `cth://tenant/<id>/subgraph` are the production emitters this doc canonicalizes)
- `@bma-implementor` — federation-tenant emitter (BMA Spec v9.1 §16.5 + §18 reference CTH anchors via these URIs)
- `@wyrd-implementor` — Walk-α resolution side (per CTH #87 + `doc/wyrd-bridging.md` §5 inverse mapping)
- `@qbp-implementor` — federation programme consumer (federation-tenancy v0.1 §5.3 references anchors via CTH URIs)
- `@beekeeper` — review & approve

## 1. Motivation

Two federation tenants emit `cth://` URIs in production today:

- **Contextus PR #11 v1.4** (merged) — `cth://anchor/<anchor-id>` for the `cth-derivation` membership predicate (theory-as-conceptual-scope)
- **Contextus PR #17** (Q3 ratified by cth-implementor 2026-05-22) — `cth://tenant/<tenant-id>/subgraph` for Research-Aid-Tenancy scaffold subgraph references

Both schemes were de-facto adopted via Contextus addenda; until this doc lands, no CTH-side authoritative grammar exists. The ratification source lives in the [Contextus PR #17 comment](https://github.com/JamesPagetButler/contextus/pull/17); this doc formalizes the grammar on-disk so the federation has a stable reference path.

Per the `inter/prompt/cth-implementor-design.md` §1 baseline (PR #10): **CTH URI authority lives at `repo-confluent-trust`.** Future schemes / subpaths / id-grammars require a §I4 cycle on this doc.

## 2. v0.1 grammar

```
cth-uri    = "cth://" segment "/" id-segment [ "/" subpath ]

segment    = "anchor" | "tenant"            ; first-segment discriminator
id-segment = anchor-id | tenant-id
subpath    = "subgraph" | <future-reserved>

anchor-id  = [A-Z] [A-Z0-9]* "-" [A-Za-z0-9-]+ ; e.g., PROOF-hurwitz, PRED-glide, MEAS-alpha
                                            ; prefix admits digits: Q27-TOV-limit-from-Fano, Q28-alpha-GUT-from-stabiliser
                                            ; suffix admits uppercase algebra labels: PROOF-loss-of-commutativity-C-to-H, MEAS-G-bbn, FLAG-J
tenant-id  = [a-z] [a-z0-9-]*               ; e.g., qbp, sharp-butler, mobius-fusion
```

Validation basis: this production admits all 149 anchor IDs in the canonical `confluent-trust-inventory-v5_3.v0.3.json` (verified 2026-06-01). The narrower v0.1 draft `[A-Z]+ "-" [a-z0-9-]+` rejected 22 of them — QBP question-anchors carry digit prefixes (`Q27-`, `Q28-`) and foundations breakdown-chain anchors carry uppercase algebra labels by construction (`-C-to-H`, `-H-to-O`, `-O-to-S`). The earlier "matches testdata/qbp_*.json" claim held only for that narrower fixture subset, not the full inventory (caught in PR #91 §I4 by qbp-implementor).

### Current grammar in production

| URI shape | Consumer | Source |
|---|---|---|
| `cth://anchor/<anchor-id>` | Contextus `cth-derivation` membership predicate (§4.6.5) | PR #11 v1.4 |
| `cth://tenant/<tenant-id>/subgraph` | Contextus Research-Aid-Tenancy `tenant_subgraph_ref` (§2.4) | PR #17 Q3 |

### Examples

```
cth://anchor/PROOF-hurwitz
cth://anchor/PRED-glide-fidelity
cth://anchor/MEAS-alpha
cth://anchor/DERIV-su2-from-axioms

cth://tenant/qbp/subgraph
cth://tenant/sharp-butler/subgraph
cth://tenant/mobius-fusion/subgraph
```

### Reserved for future-extension (NOT in v0.1)

- `cth://anchor/<id>/<subpath>` — sub-properties of a single anchor (e.g., `cth://anchor/PROOF-hurwitz/verification`)
- `cth://tenant/<id>/<other-subpath>` — additional tenant-scoped paths (e.g., `cth://tenant/qbp/anchors`, `cth://tenant/qbp/predictions`, `cth://tenant/qbp/health`)
- `cth://chain/<id>` — chain references (currently embedded in confluence-point `paths[]` array)
- `cth://confluence/<id>` — confluence-point references
- `cth://federation/...` — federation-meta references (deliberately reserved at top level; current `cth://` already implies federation-scope so this segment is hot-banked for future grammar refinement)

Adding any of the above requires a §I4 cycle amending this doc.

## 3. Rejected alternatives (Q3.1 + Q3.2 from PR #17)

### Scheme — rejected

- **`cth+anchor://` / `cth+tenant://` (subschemes)** — first-segment discriminator already encodes kind; subschemes add no information at cost of proliferation
- **bare `cth:` (opaque URI)** — doesn't parse as hierarchical URI per RFC 3986; precludes future subpath grammar

### Path — rejected

- **`cth://federation/tenant/<id>`** — redundant `federation/` segment. `cth://` already implies federation-scope reference; `tenant/` discriminator is sufficient. Reserving `cth://federation/...` for top-level federation-meta is future-extension scope.
- **`cth://anchor/tenant-<id>`** — conflates anchor (single graph node with `model.Anchor` shape) with subgraph (collection of nodes). The anchor-discriminator-with-tenant-prefix-as-ID would overload `model.Anchor.ID` semantics in confusing ways.

## 4. Walk-α resolution semantics

The grammar above is the federation-canonical URI shape; resolution to Wyrd primitives at Walk-α is design-deferred to the `store/wyrd.go` adapter per `doc/wyrd-bridging.md` §5 inverse mapping + CTH #87 forward-pins.

Per-shape resolution sketch (Walk-α design point; not v0.1-load-bearing):

| URI shape | Wyrd resolution (Walk-α) |
|---|---|
| `cth://anchor/<id>` | `Node{Type: "cth.anchor", ID: <id>}` lookup; one node |
| `cth://tenant/<id>/subgraph` | Set of `Node`s carrying `tenant_id = <id>` metadata (subgraph by query) OR `Node{Type: "cth.tenant_subgraph", ID: <id>}` anchored-via-edge (subgraph by structure) — Walk-α adapter design choice per CTH #87 FP-2 |

Neither resolution path is implemented in CTH v0.3; both are forward-pinned for Walk-α.

## 5. `cth resolve` CLI surface

[PR #83](https://github.com/JamesPagetButler/confluent-trust/pull/83) ships `cth resolve <inventory.json> <anchor-id>` (closes [#82](https://github.com/JamesPagetButler/confluent-trust/issues/82)). The CLI:

- **Accepts bare anchor-id** (matches the workspace-convention prefix shape — `PROOF-*`, `PRED-*`, etc.) — not the full `cth://anchor/<id>` URI
- **Does NOT yet parse `cth://` URIs** — callers strip the `cth://anchor/` prefix before invoking

Future v0.4 candidate: `cth resolve` accepts both bare anchor-id AND `cth://anchor/<id>` URI shapes; resolution is identical. Tracked in #82's follow-on issue.

Tenant-subgraph resolution (`cth://tenant/<id>/subgraph`) is Walk-α-deferred per §4 above.

## 6. Library API (v0.4 candidate, deferred)

Optional federation-canonical parser primitive (not in v0.1; flagged in Contextus PR #17 §I4 review FP-1):

```go
// ParseCTHRef parses a federation-canonical cth:// URI per the v0.1 grammar.
// Returns the resolved Ref or an error for malformed URIs.
func ParseCTHRef(uri string) (Ref, error)

// Ref discriminates between anchor and tenant URI shapes.
type Ref struct {
    Kind    RefKind // RefKindAnchor | RefKindTenant
    ID      string  // anchor-id or tenant-id
    Subpath string  // e.g., "subgraph" for tenant; empty for v0.1 anchor refs
}

type RefKind uint8

const (
    RefKindUnknown RefKind = iota
    RefKindAnchor
    RefKindTenant
)
```

Federation tenants importing the parser get a single source-of-truth for grammar conformance, replacing per-tenant regex implementations (Contextus PR #20's `^cth://` schema pattern is the current loose-validation form; can tighten on adoption of `ParseCTHRef`).

This API is a v0.4 candidate; v0.1 of this doc ships grammar-only.

## 7. Future-extensibility — §I4 amendment process

Adding any of the following to this doc requires a §I4 cycle on the named reader-list:

- New first-segment discriminator (e.g., `cth://chain/`, `cth://confluence/`, `cth://federation/`)
- New tenant-scoped subpath (e.g., `cth://tenant/<id>/health`)
- New anchor-scoped subpath (e.g., `cth://anchor/<id>/verification`)
- Grammar refinement to `anchor-id` / `tenant-id` (currently regex-only; tightening to typed enum would be a breaking change requiring federation coordination)

Federation tenants that emit `cth://` URIs must conform to the grammar pinned by the latest merged version of this doc. Tenants needing extension propose via §I4 cycle here, NOT via tenant-side addenda.

## 8. Cross-references

- **Contextus PR #11** — `cth://anchor/<id>` first production emitter (cth-derivation membership predicate)
- **Contextus PR #17** — `cth://tenant/<id>/subgraph` second production emitter (Research-Aid-Tenancy); Q3 ratification at https://github.com/JamesPagetButler/contextus/pull/17
- **CTH #71** + design surface PR #73 — v0.3 schema; references anchor IDs (bare strings; not URIs)
- **CTH #60** + PR #81 (`doc/wyrd-bridging.md`) — Wyrd-cutover mapping; §5 inverse-mapping resolves the URIs at Walk-α
- **CTH #82** + PR #83 — `cth resolve` CLI subcommand (anchor-id surface; URI surface deferred to v0.4)
- **CTH #87** — Walk-α `store/wyrd.go` adapter forward-pins (per-shape resolution semantics)
- **BMA Spec v9.1 §16.5** — federation-CI requirement for anchor-resolution
- **inter/prompt/cth-implementor-design.md §2.8** — federation-additive-only contract (any narrowing of this grammar requires federation coordination)

## 9. Open questions for §I4 reviewers

1. **Should v0.1 ship the `ParseCTHRef` library API now or defer to v0.4?** My lean: defer (the grammar doc is the v0.1 deliverable; the library API can land when a federation tenant has a concrete need beyond the current regex-based validation).

2. **Should the doc enumerate the future subpath candidates (§2 reserved list) as named-but-not-implemented OR leave the future-extension entirely unspecified?** My lean: enumerate (gives federation tenants visibility into the design space; reserves the namespace from accidental collision).

3. **Should `tenant-id` grammar tighten further (e.g., require RFC 1123 hostname-like rules, max length)?** Currently `[a-z][a-z0-9-]*` admits arbitrarily long tenant-ids. My lean: leave at v0.1 minimum-spec; tighten if a real federation collision case surfaces.

4. **Coordination with `bma-systema #197` Privacy decision** — if option (a) Privacy lifts to Wyrd `model.Node`, tenant-subgraph URIs implicitly inherit substrate-tier privacy classification from their Wyrd nodes. Does this doc need a Privacy section, or is that handled federation-wide elsewhere? My lean: defer to the #197 decision; if Privacy becomes URI-bearing (unlikely but possible), revisit in a v0.2 amendment of this doc.

## 10. Status + ratification record

- **v0.1 grammar ratified by cth-implementor** on Contextus PR #17 Q3 closure 2026-05-22 (comment URL in §1 above)
- **This doc formalizes** the ratification on-disk; no semantic change to what's already authoritative
- **Federation tenants emitting `cth://` URIs** conform to §2 grammar as of this doc's merge
- **Future amendments** require §I4 cycle per §7
