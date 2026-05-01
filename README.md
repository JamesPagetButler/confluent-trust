# Confluent Trust Hypergraph (CTH)

Go implementation of the [Confluent-Trust-Hypergraph theory v0.2][theory] —
a formal framework for quantifying the epistemic health of scientific
research programmes using information-theoretic measures on directed
hypergraphs.

> **Status:** Bootstrap (v0.1.0-alpha). Crawl phase in progress.

## What it is

CTH is a Go library plus CLI that computes:

- Information deficit Δ(G) — what a programme doesn't explain
- Net compression ρ_net — bits explained per bit assumed
- Chain fidelity μ — multiplicative reliability along derivations
- N-ary mutual information at confluence points
- Per-branch health for hypothesis forks (Theory v0.2 §2.8)
- Programme merge with theoretical / engineering deficit classification

It is designed to be embedded by [BMA][bma] (the cognitive substrate),
[Contextus][contextus], and any programme that needs epistemic-health
monitoring. Storage is layered: JSON files (Crawl), MuninnDB engrams
(Walk), SurrealDB structural ground truth (Run).

## Phases

| Phase | Storage | Capabilities |
|---|---|---|
| Crawl (v0.1.x) | JSON files | Core types, all compute, CLI |
| Walk  (v0.2.x) | MuninnDB + NATS | Hebbian co-activation, Ebbinghaus decay, branch-locked vaults |
| Run   (v0.3.x) | + SurrealDB | BMA integration, agent flow field |

See `~/Documents/CTH/Archive/CTH-Go-Implementation-Plan (1).md` for the
27-issue plan. All issues are filed against this repo with phase /
type / priority labels.

## Layout

```
model/                 Pure data types (stdlib only)
compute/               Pure functions (stdlib only)
store/                 JSON, MuninnDB, SurrealDB backends
report/                Dashboard, markdown report, river map
cmd/cth/               CLI: cth analyse | merge | health | compare | fork
schema/                JSON Schema 2020-12 for inventory documents
internal/validate/     Schema validator (kept out of model/ to honor §1.3)
testdata/              minimal, qbp_v3_2, qbp_quantum, qbp_dm_fork
```

## Quick start

```bash
go build ./...
go test ./...
go run ./cmd/cth analyse testdata/qbp_v3_2.json
```

## Theory references

- Theory v0.2 (canonical: 942-line variant): `Confluent-Trust-Hypergraph-Theory-v0_2 (1).md` in the parent CTH/Archive/
- Companion analysis: `QBP-CTH-Analysis-Report-v3_2.md` (planned, Issue #21)

## Authorship

Theory and design: James Paget Butler, with Claude (Opus, Red Team).
This repo implements the v0.2 theory. The Python reference engine
(`cth_engine_v2.py`) lives in the parent CTH/Archive/ directory and is
the source of truth for all known-value regression tests until this
implementation reaches v0.1.0.

## License

Apache 2.0 — see `LICENSE`.

[theory]: ../Archive/Confluent-Trust-Hypergraph-Theory-v0_2%20%281%29.md
[bma]: https://github.com/JamesPagetButler/bma-systema
[contextus]: ../../Contextus/
