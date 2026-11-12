# CTH Tooling Guide — the CLI and library, with real output

Every example below was run against the checked-in fixture
`testdata/minimal.json` and the output is pasted verbatim (not
reconstructed from memory) — run it yourself to confirm:

```bash
go build -o cth ./cmd/cth
```

`cth help` (or `-h`, `--help`, or any unrecognized command) prints the
full usage block, which is also the authoritative command list — if this
guide and `cth help` ever disagree, trust `cth help`.

## Two ways to consume CTH

1. **The CLI** (`cmd/cth`) — for operators, scripts, and CI/CD. Every
   subcommand below.
2. **The Go library directly** — `model` (types) + `compute` (the pure
   health-measure functions) + `store` (load/save) + `report` (rendering).
   This is the recommended path for a Go program embedding CTH (e.g. a
   dashboard or a CI gate) rather than shelling out to the CLI. Load an
   inventory with `store.LoadInventory` — it transparently migrates
   legacy v0.1 binary-confluence documents, so you don't need to think
   about schema versions on read.

## `cth health` — compact dashboard

```
$ cth health testdata/minimal.json
╔════════════════════════════════════════════════════════════╗
║ CTH Health: MINIMAL v0.1                                   ║
╠════════════════════════════════════════════════════════════╣
║ Anchors:        5    (axiom 1, proof 2, meas 2, pred 0) ║
║ Coherence:      1.0000                                      ║
║ ρ_net:          1.1729  [1.2550, 1.1729, 1.0372]            ║
║ ρ_gross:        1.1729                                      ║
║ Top bridge:     PROOF-newton-2nd                            ║
║ Sediment:       L=0  LS=2  M=0  H=0  sharp=false            ║
║ Highest eddy:   INST-air-resistance  (π_w=0.0000, g_w=+Inf) ║
╚════════════════════════════════════════════════════════════╝
```

The one-screen version of `analyse` — use this for a quick "is this
programme's ledger in a sane state" check, e.g. in a CI job comment.

## `cth analyse` — full markdown report

```
$ cth analyse testdata/minimal.json
# CTH analysis — MINIMAL v0.1

*Inventory timestamp: 2026-04-30T00:00:00Z*

## Compression

- **ρ_gross**: 1.1729
- **ρ_net**:   1.1729
- **Sensitivity bracket** (½H / H / 2H): 1.2550 / 1.1729 / 1.0372
- **Sensitivity ratio**: 0.8265 (>0.5 = robust)

## Tier breakdown

- Tier 0 (axioms):       1
- Tier 1 (proofs):       2
- Tier 2 (measurements): 2
- Tier 3 (predictions):  0

## Coherence

- **R_c**: 1.0000

## Top bridges
...
```

(truncated — also includes sediment/eddy sections and the full bridge
table). Use `-o out.md` to write it to a file instead of stdout.

## `cth resolve` — look up one anchor

```
$ cth resolve testdata/minimal.json PROOF-falling-rate
{
  "description": "F=ma + uniform gravity yields a = g.",
  "name": "Mass-independent free-fall rate",
  "id": "PROOF-falling-rate",
  "prediction_chain": [ "PROOF-newton-2nd" ],
  "tier": 1,
  "provenance": "T",
  "status": "coherent"
}
```

Useful in scripts (e.g. `cth resolve inv.json $ID | jq .status`) or for a
quick manual check during review — this is what reviewers use to verify
an anchor's fields first-hand rather than trusting a PR description.

## `cth score` — predictions vs. observations

```
$ cth score testdata/minimal.json
# Prediction scores — MINIMAL v0.1

**1 anchors with predictions** (1 scored, 0 untested, no observation)

| Anchor ID | Status | Predicted | Measured | Discrepancy | Regime | Confirm bits |
|---|---|---|---|---|---|---|
| MEAS-galileo-tower | coherent | 9.81 | 9.79 | 0.2039% | laminar | 8.9381 |
```

`--prediction <ID>` narrows to one anchor's detail; `--regime` groups the
table by regime instead of listing every anchor flat.

## `cth fork` / `cth check-branch` — hypothesis-fork health

```
$ cth fork testdata/minimal.json
# Fork health — MINIMAL v0.1

_No fork_points in this inventory._
```

`testdata/minimal.json` has no `fork_points`, so both commands report
empty — that's the correct behavior, not an error. Try them against
`testdata/qbp_dm_fork.json` to see live fork-health output (per-branch
ρ_net comparison and a consistency report that flags an anchor claimed by
more than one branch).

## `cth compare` — compression velocity between two snapshots

```
$ cth compare testdata/minimal.json testdata/minimal.json
# Compression velocity

- old MINIMAL v0.1: ρ_net = 1.1729 (n=5)
- new MINIMAL v0.1: ρ_net = 1.1729 (n=5)
- Δρ/Δn = 0
```

Feed it two inventory snapshots (e.g. before/after a PR) to see whether
the ledger got more or less explanatory per anchor added.

## `cth merge` — combine two programmes

```
$ cth merge testdata/minimal.json testdata/minimal.json
{
  "programme": "MINIMAL+MINIMAL",
  "version": "merged",
  "parent_programmes": [ "MINIMAL", "MINIMAL" ],
  "axioms": [ ... ],
  ...
}
```

Writes the merged inventory as JSON (to stdout, or `-o out.json`). Use
this when two programmes' ledgers need to become one — it carries over
both parents' nodes and records `parent_programmes` for provenance.

## `cth migrate` — v0.2 → v0.3 upgrade

```
$ cth migrate testdata/minimal.json --check
# Migration report: testdata/minimal.json → 

## Summary

- Total anchors: 4
- Mechanically translated: 4
- Decisions applied (from --decisions): 0
- Decisions still needed: 3
- Warnings: 0

## Decisions still needed

The following anchors carry `provenance: "T"` (theoretical) and need
human classification as either `theory` (programme-internal argument) or
`theory-external` (external published theorem invoked as proof).
...
### PROOF-newton-2nd
- **Suggestion:** `theory`
- **Rationale:** no citation pattern detected → safe default
```

`--check` runs the dry-run report above without writing anything — use
this first. The legacy `provenance: T/E/H` field doesn't carry enough
information to auto-assign `provenance_kind` safely (an anchor marked
`T` could be `theory` or `theory-external`), so migrate flags every
ambiguous one for a human decision rather than guessing. Write a
decisions file in the shape the report prints, then:

```bash
cth migrate inv.json --decisions decisions.json -o inv-v0.3.json
```

## `cth lean-link` — cross-reference Lean theorems against PROOF anchors

```
$ cth lean-link testdata/minimal.json .
# lean-link reconciliation report

## Summary

- Proof-bearing anchors examined: 0
- Theorems found in corpus: 4
- Proven & wired: 0
- Orphan theorems: 4
- Stale references: 0
```

(`testdata/minimal.json` has no `lean_theorem` fields set, so this run
is just illustrative — zero proof-bearing anchors to check.) Points
`<corpus-root>` at a directory of `.lean` files and reports which
`theorem`/`lemma` declarations are (a) wired to a `PROOF-*` anchor and
verified, (b) orphaned (exist in the corpus, not referenced by any
anchor), or (c) stale (an anchor's `proof_file`/`lean_theorem` no longer
resolves in the corpus). `--update-inventory` writes resolved metadata
back; `--strict` exits non-zero on any finding, for use as a CI gate.

**Known limitations** (as of this writing — check
[#124](https://github.com/JamesPagetButler/confluent-trust/issues/124)
and [#125](https://github.com/JamesPagetButler/confluent-trust/issues/125)
for current status before relying on either):

- Matches by **bare declaration name only**. If your `lean_theorem` uses
  the `<module>::<theorem>` join key from
  `doc/design/proven-theorem-anchoring.md` §3.4 (some consumers already
  do, as a workaround), lean-link doesn't resolve the module part — #124.
- The identifier scanner's regex doesn't accept Lean's Unicode
  subscripts (`B₁`, `B₂`, ...), so declarations differing only by a
  subscript collapse to the same matched name — #125.

## What's *not* in this repo

- **Programme-side CI gates** (e.g. QBP's `root_audit.py`, its
  anchor-worthiness manifest, or Gleipnir's `lean/.gleipnir-gate.json`
  required-theorems pin) live in the *consuming* programme's own repo.
  They read a vendored copy of `schema/inventory.schema.json`, but this
  repo has no code that knows about them.
- **Rendering a programme's own narrative view** (e.g. a generated
  `ladder.md` for Gleipnir) is also the consuming programme's own
  tooling — `cth analyse`/`cth health` give you the generic report; a
  programme-specific narrative is built on top, typically by a small
  script that reads the inventory JSON directly via the Go library or
  shells out to `cth resolve` per anchor.
