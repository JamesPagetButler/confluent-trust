# Confluent Trust Hypergraph (CTH)

Go implementation of the Confluent-Trust-Hypergraph theory — a formal
framework for tracking the epistemic state of a research programme (proofs,
measurements, predictions, and the chains between them) as a typed,
versioned hypergraph, plus information-theoretic health measures over it.

> **Status:** live, multi-programme. Schema is at **v0.3.5** (see
> `schema/inventory.schema.json`'s `x-schema-semver` header for the full
> changelog of what each point release added). Two programmes consume it
> today — **QBP** (physics) and **Gleipnir** (P vs NP) — each in their own
> repo, vendoring a byte-identical copy of the canonical schema from here.
> If you're a new programme reading this to decide how to represent your
> own claims: start with [`doc/SCHEMA-GUIDE.md`](doc/SCHEMA-GUIDE.md), not
> this file or the files under `doc/design/` (those are point-in-time
> design-review records, not an onboarding guide).

## What it is

CTH is a Go library plus CLI that:

- Stores a programme's axioms, proof/measurement/prediction anchors, and
  the derivation chains between them as one versioned JSON document (the
  **inventory**), validated against a canonical JSON Schema.
- Computes information-theoretic health measures over that hypergraph:
  information deficit Δ(G), net compression ρ_net, chain fidelity μ,
  per-branch health for hypothesis forks, N-ary mutual information at
  confluence points.
- Cross-references formal proofs (Lean/Coq/Agda) against the ledger's
  `PROOF-*` anchors, and grades multi-assistant proof-checking strength
  (the PA engine, `internal/pa`).
- Scores predictions against observations and reports programme merges
  with theoretical/engineering deficit classification.

It is designed to be embedded by any programme that needs this kind of
epistemic-health tracking — not just physics. **Gleipnir (pure maths, P vs
NP) is the first non-physics consumer**, and found real gaps in the schema
that physics never exercised (mutually exclusive root targets, branches
that refine internal structure without bearing on a root, open problems
blocked by known proof-technique barriers) — see
[`doc/SCHEMA-GUIDE.md`](doc/SCHEMA-GUIDE.md) §6 for how those are modeled.

Storage today is JSON-file persistence (Crawl phase). A Wyrd-backed store
(the workspace's native hypergraph DB) is planned for Walk/Run; no
third-party DB is used at any phase.

## Who owns what here

- **This repo (`confluent-trust`) holds the canonical schema and the CLI.**
  Schema changes land here first, then get vendor-synced (byte-identical
  copy + sidecar sha) into each consuming programme's repo. See
  [`doc/design/schema-change-propagation.md`](doc/design/schema-change-propagation.md)
  for the propagation contract.
- **Programme-specific enforcement tooling lives in the consuming
  programme's own repo, not here.** For example QBP's `root_audit.py`
  (whole-ledger reachability gate) and its anchor-worthiness manifest are
  QBP-side scripts that read a vendored copy of this schema; they are not
  part of this repo and this repo doesn't know about them.
- **Formal-proof cross-referencing (`cth lean-link`) is partially built.**
  It matches a `PROOF-*` anchor's `lean_theorem` against a Lean corpus by
  bare declaration name. It does not yet follow the `<module>::<theorem>`
  join key that `doc/design/proven-theorem-anchoring.md` §3.4 specifies,
  and its identifier regex doesn't accept Lean's Unicode subscripts — both
  tracked as [#124](https://github.com/JamesPagetButler/confluent-trust/issues/124)
  and [#125](https://github.com/JamesPagetButler/confluent-trust/issues/125).

## Layout

```
model/                 Pure data types (stdlib only) — Axiom, Anchor, Chain, etc.
compute/               Pure functions (stdlib only) — the health measures
store/                 JSON-file load/save; Wyrd-backed adapters land at Walk/Run
report/                Dashboard, markdown report, river map rendering
cmd/cth/               CLI — see `doc/TOOLING-GUIDE.md` or `cth help`
internal/validate/     JSON Schema validator (kept out of model/ to honor §1.3 purity)
internal/pa/           Proof-assistant (PA) grading engine
internal/leanlink/     Lean-corpus ↔ PROOF-anchor cross-referencing (#124/#125 open)
schema/                JSON Schema 2020-12 for inventory documents — the canonical copy
testdata/              Fixture inventories; `testdata/minimal.json` is the smallest one
```

## Quick start

```bash
go build ./...
go test ./...

# A 5-anchor fixture inventory is checked in for exactly this purpose:
go run ./cmd/cth health testdata/minimal.json
go run ./cmd/cth analyse testdata/minimal.json
go run ./cmd/cth resolve testdata/minimal.json PROOF-falling-rate
```

For every CLI subcommand (`analyse`, `health`, `merge`, `compare`, `fork`,
`check-branch`, `score`, `migrate`, `lean-link`, `resolve`) with worked
examples and real output, see [`doc/TOOLING-GUIDE.md`](doc/TOOLING-GUIDE.md).
For the schema itself — what each node type means, the enum vocabulary,
and worked examples including the exclusive-root / barrier / no-implication
patterns Gleipnir needed — see [`doc/SCHEMA-GUIDE.md`](doc/SCHEMA-GUIDE.md).

## Design history

The `doc/design/` directory holds point-in-time design-review records
(§I4 review surfaces, named-reviewer threads, acceptance-criteria↔test
mappings). They are the provenance trail for *why* the schema looks the
way it does, not a guide to using it as it stands today — read
`doc/SCHEMA-GUIDE.md` first, and only drop into these if you need the
history behind a specific field.

## Authorship

Theory and design: James Paget Butler, with Claude (schema/PA-engine
stewardship: cth-implementor seat) and the federation's named reviewers
(see individual design docs for per-decision attribution).

## License

Apache 2.0 — see `LICENSE`.
