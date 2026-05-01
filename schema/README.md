# Inventory Schema

`inventory.schema.json` is the JSON Schema (Draft 2020-12) contract that
every CTH inventory document must satisfy. It is embedded by
`internal/validate` and enforced on every load by `store/json.go`.

## Why a schema and not just struct tags

Go struct tags catch type mismatches but not semantic constraints —
e.g. "an Axiom marked `derivable: true` must include `derived_from_axioms`",
"every fork point must have at least 2 branches", "tier ∈ {1,2,3}". The
JSON Schema captures these and is consumable by external producers
(BMA, Contextus) without depending on the Go binding.

## Versioning

The top-level `schema_version` field governs parser dispatch:

| Value           | Confluence shape           |
|-----------------|----------------------------|
| absent / `v0.1` | binary `path_a` / `path_b` (auto-converted on load) |
| `v0.2`          | strict N-ary `paths[]`      |

Anything else is rejected with `unsupported schema version`.

## Field semantics

See:

- `../../Archive/Confluent-Trust-Hypergraph-Theory-v0_2 (1).md` (canonical, 942 lines) — Definitions 1–23, §4.1 data model, §2.8–§2.9 fork extension.
- `../../Archive/cth_engine_v2.py` — current Python reference implementation; field names match.

## Computed (not stored) properties

Some quantities referenced in the theory are **computed**, not stored:

- `Anchor.domain` — derived by `compute.ClassifyDomain(id)`.
- `Anchor.residual_entropy_bits`, `Anchor.confirmatory_info_bits` — `compute/entropy.go`.
- `ConfluencePoint.mutual_info_bits` — `compute/mutual_info.go` (the JSON field is a cache only; ignored on load).

The schema permits these fields in input documents (legacy data may
include them) but they are recomputed on `cth analyse`.
