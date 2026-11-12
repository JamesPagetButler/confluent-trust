# CTH Schema Guide — how to represent your programme

This is the guide to read before writing your first inventory. The
canonical schema file (`schema/inventory.schema.json`) is the source of
truth for exact field names and constraints — this doc explains *what the
fields mean* and *when to use which one*, with worked examples. If
something here and the schema file disagree, the schema file is right and
this doc is stale; file an issue.

Every example below is checked against `schema/inventory.schema.json` as
it stands at the time this guide was written (`x-schema-semver: 0.3.5`).

## 1. The core idea

A programme's CTH inventory is one JSON document holding a typed
hypergraph of claims:

- **Axioms** — things the programme takes as given (layer 0, not derived
  from anything else in the ledger).
- **Anchors** — the actual claims: proofs, measurements, predictions.
  Each one has a `tier` (1 = proof, 2 = measurement, 3 = prediction
  awaiting observation) and a `status` (is it currently believed true,
  false, untested, etc.).
- **Chains** — a claimed derivation from one or more source anchors to a
  target anchor, with its own fidelity and status.
- **Confluence points** — where independent chains converge on the same
  anchor, with the mutual information between the converging paths.
- **Fork points** — a question with multiple mutually-exclusive
  hypothesis branches, each with its own anchors/chains, so you can track
  "which branch does the evidence currently favor" without forcing a
  premature merge.

Nothing in the schema asserts that an unresolved claim is true, false, or
even decidable — `status` and the other fields only ever describe what has
actually been established, never what you expect, hope, or need to be
established. If you're tempted to add a field or value whose only purpose
is to make an *unproven* claim look more special than plain "we don't know
yet" (see §6.2 on `barrier` below for a concrete instance of this), that's
usually a sign you want a different anchor, not a different status.

## 2. Anchors — the claim nodes

```json
{
  "id": "PROOF-falling-rate",
  "name": "Mass-independent free-fall rate",
  "tier": 1,
  "provenance": "T",
  "provenance_kind": "theory",
  "status": "coherent",
  "description": "F=ma + uniform gravity yields a = g.",
  "prediction_chain": ["PROOF-newton-2nd"]
}
```

(from `testdata/minimal.json` — run `cth resolve testdata/minimal.json
PROOF-falling-rate` to see this rendered.)

**`id` prefix convention** (not schema-enforced, but followed everywhere):
`PROOF-*` for Tier-1 proofs, `MEAS-*` for Tier-2 measurements, `PRED-*` for
Tier-3 predictions, `FLAG-*`/`KILLED-*` for special states.

**`status`** (the `Status` enum) describes the claim's own current
epistemic state: `coherent` (currently holds), `untested` (nobody has
checked), `incoherent`, `contested`, `refuted`, `killed`, `marginal`
(holds but with a caveat — see `notes`), `converged`, `falsified`. A
status only changes when something in the ledger justifies the change —
never by inference from a sibling claim's status, and never by assertion.

**`provenance_kind`** (the fine-grained classifier, additive on top of the
legacy `provenance` T/E/H field) says *what kind* of claim this is:
`proof` (should be backed by a kernel-checked theorem), `derivation` (the
formal backbone is kernel-verified but the full claim is an
interpretation of it — e.g. "3 generations" from a dimension count),
`theory` (programme-internal argument, not formally checked),
`theory-external` (an external published theorem taken as given),
`experiment`, `hypothesis`, `internal-compute` (e.g. `native_decide` —
checked by a non-kernel-trusted path), `philosophy` (interpretive,
not proof-apt).

**Only `proof` and `derivation` anchors may carry proof-evidence fields**
(`proof_state`, `proof_language`, `theorems[]`, `verification`) — this is
schema-enforced (see `C1` in the Anchor `allOf`). A `theory` anchor that
tries to claim Lean verification will fail validation; that's
deliberate — it stops an unverified argument from dressing itself up as a
checked one.

**`prediction_chain`** is the list of anchor/axiom ids this claim's
derivation rests on. It can be **empty** — an anchor with no
`prediction_chain` entries is a standalone claim, not wired into any
derivation toward anything else. That's a legitimate, common shape (see
§6.3).

## 3. Formal proofs — tying an anchor to Lean/Coq/Agda

```json
{
  "proof_language": "lean4",
  "proof_file": "Foundations/CDAlg/CDDimension.lean",
  "theorems": [{ "name": "CDAlg.finrank_eq_two_pow_n", "status": "verified" }],
  "verification": {
    "toolchain": "leanprover/lean4:v4.30.0",
    "libraries": { "mathlib": { "ref": "...", "sha": "..." } },
    "verified_at": "2026-09-01T00:00:00Z",
    "verifier": "lake build",
    "result": "pass"
  }
}
```

`lean_theorem` names the single primary witness by its **fully-qualified
name as it appears after the `namespace` declaration — never inferred
from the filename**. (This has bitten reviewers before: a file named
`OctonionLaws.lean` can still declare `namespace QBP.Foundations.CDAlg`,
so its theorems resolve as `CDAlg.*`, not `OctonionLaws.*`.)
`lean_companion_theorems` lists any supporting witnesses.

`cth lean-link` cross-references these against an actual Lean corpus —
see `doc/TOOLING-GUIDE.md`. As of this writing it matches by bare
declaration name only; it does not yet parse the `<module>::<theorem>`
join key some consumers use as a workaround, and its identifier scanner
drops Lean's Unicode subscripts (tracked as #124/#125).

**`proof_assistants[]`** is a separate, additive mechanism: it records
*which* assistant(s) independently checked the claim, each entry pointing
at pinned, re-executable evidence (`evidence_ref`) rather than declaring a
grade. The actual PA (proof-assistant count) grade is **derived by the PA
engine from that evidence**, never read off this field directly — see
`internal/pa` and `docs/verification/108-pa-engine.md` for the mechanics.
`trust_check` on this record is veto-only: `fail` zeroes that assistant's
contribution, `pass` never substitutes for the engine's own re-derivation.

## 4. Chains, confluence, and forks

A **Chain** claims a derivation exists from `source_ids` to `target_id`,
with `steps` (count), `fidelity` (0–1, multiplicative reliability), and
its own `status`. A **ConfluencePoint** names an anchor reached by more
than one independent chain, with `mutual_info_bits` quantifying how much
the paths actually agree rather than merely coincide.

A **ForkPoint** models a genuine, still-open empirical or theoretical
fork: one `question`, 2+ `branches` (each its own sub-hypothesis with its
own `hypothesis` statement, `burden`, and attached anchors/chains/
predictions), and optionally `branch_observations` recording how a shared
anchor is interpreted differently under each branch. This is for "we
don't yet know which of these is right and both are live" — not for a
Clay-style exclusive pair where proving either one settles the question
for good (see §6.1 for that case, which uses a different mechanism).

## 5. Open questions with a stated falsifier — the four-bucket pattern

A root-level claim (an `Axiom`, `MetaPrinciple`, or `Interpretation`) can
carry a **decision lifecycle**:

```json
{
  "id": "AXIOM-1",
  "decision_state": "open",
  "kill_condition": [
    {
      "kill": "A measurement at >5σ finds a clock-rate anomaly",
      "closure": "measurement",
      "discharge": "MEAS-clock-anomaly-check"
    }
  ]
}
```

`decision_state` is `open` (a live question with a stated way to close
it) or `settled` (proved or forced, with the forcing anchor/ruling cited
in `discharge`). **The schema enforces that `open` always carries a
non-empty `kill_condition`** — nothing gets parked open without someone
having written down what would actually resolve it. Each
`kill_condition` entry is one *Impasse Record*: `kill` is the
ledger-observable event that would fire it (no placeholders — minimum 8
characters, and real content, checked at root-audit time by consuming
programmes); `closure` says *how* it closes — `derivation` or
`measurement` (both require `discharge`, naming the resolving anchor/
issue) or `ruling-rescope` (a constitutional change; no `discharge`
needed because firing the kill *is* the resolution).

**A root can carry more than one `kill_condition` entry** — this is how
you represent "this closes if A happens, or separately if B happens"
without ever asserting A∨B anywhere in the schema. See §6.1.

## 6. Patterns Gleipnir needed that QBP never exercised

QBP is one physical theory converging on predictions, so until Gleipnir
(the P vs NP programme) the schema had never been stressed by a
programme whose claims are pure mathematical dichotomies. These three
patterns are the answers that came out of that — recorded here so the
next non-physics programme doesn't have to re-derive them.

### 6.1 Mutually exclusive root targets, without ever asserting their disjunction

Problem: a root like "P = NP?" has exactly two possible resolutions, and
proving either one settles it and makes the other one permanently
irrelevant. You must never encode the root as "target-A ∨ target-B",
because excluded middle would make that trivially provable regardless of
whether anyone has actually proved either side.

**Pattern:** model the root as a `MetaPrinciple`, `decision_state: open`,
with **one `kill_condition` entry per candidate resolution** — each
`closure: "derivation"` discharging to its own independent `PROOF-*`
anchor:

```json
{
  "id": "META-clay-p-vs-np",
  "name": "P vs NP",
  "statement": "Is P = NP?",
  "decision_state": "open",
  "kill_condition": [
    { "kill": "Gleipnir.Clay.PEqNP is proven in Lean",
      "closure": "derivation", "discharge": "PROOF-clay-p-eq-np" },
    { "kill": "Gleipnir.Clay.PNeNP is proven in Lean",
      "closure": "derivation", "discharge": "PROOF-clay-p-ne-np" }
  ]
}
```

`PROOF-clay-p-eq-np` and `PROOF-clay-p-ne-np` are two ordinary, independent
`status: "untested"` anchors. Nothing anywhere asserts a relationship
*between* them — the schema never encodes "one of these must be true," it
just records two candidate routes to settling one shared root. When
(if) one of them is ever proven, that root's `decision_state` flips to
`settled` citing that anchor's id in `discharge`.

The sibling anchor that never got proven **stays `status: "untested"`
forever** — that remains the honest state, since nobody determined its
truth value; the question just stopped mattering once the root settled
on the other route. Don't flip it to `refuted` (nobody refuted it) or
invent a new status for "mooted" — a `notes` entry pointing at the root
that settled is enough, because `status` describes *this claim's own*
truth state, never the state of some other claim or root.

### 6.2 "No known implication" — a true result that doesn't move the root

Problem: a result like AKS primality-testing (PRIMES is in P, 2002) is
true, proven, and worth recording — it sharpens the internal map of what
is known to be in P — but it has **no known bearing** on P vs NP itself.
Marking it `refuted` would be false (nobody showed it fails to imply
anything); marking it `untested` would be false too (it *is* proven).

**Pattern:** this needs no special status at all — it's just an ordinary
`status: "coherent"`, `provenance_kind: "proof"` anchor with an **empty
`prediction_chain`**: a standalone proof, not wired into any chain toward
the root. The schema already permits this (`prediction_chain` has no
minimum length). If a program expects *many* such internal-structure
results and wants them queryable rather than merely absent-from-any-chain,
an additive field (tracked as a proposal on
[#126](https://github.com/JamesPagetButler/confluent-trust/issues/126),
not yet landed) would make that explicit rather than implicit — but
nothing here blocks you from starting without it.

### 6.3 Known proof-technique barriers — never a status, always their own anchor

Problem: `P ≠ NP` is known to be unreachable by whole classes of proof
technique — relativizing arguments, natural proofs, algebrizing
arguments — and those barrier results are themselves real, citable
theorems (Baker–Gill–Solovay; Razborov–Rudich; Aaronson–Wigderson). It's
tempting to add a `status` value like `"open-hard"` to mark a target
anchor as "known-hard" rather than merely `untested`.

**Don't do this.** A barrier result is evidence about a *different*
claim — which techniques can't work — not about the target's own truth
value. Citing ten barriers against a technique changes nothing about
whether the target itself is true, false, or still simply unproven; a
`status` that implied otherwise would be the same category error as
letting an unverified `native_decide` run or a napkin hypothesis read as
settled. (This is the same lesson behind the existing `derivation`
provenance_kind's G2 guardrail: a result that *looks* close to proving
something never gets to flip a status it hasn't actually earned.)

**Pattern:** the barrier is a normal, independent anchor in its own
right — `status: "coherent"`, `provenance_kind: "theory-external"` (or
`proof` if formalized), citing the actual theorem:

```json
{
  "id": "PROOF-natural-proofs-barrier",
  "name": "Natural-proofs barrier (Razborov–Rudich 1994)",
  "tier": 1,
  "provenance": "T",
  "provenance_kind": "theory-external",
  "status": "coherent",
  "theory_citation": "Razborov, Rudich. Natural Proofs. JCSS 1994.",
  "description": "No natural combinatorial property both large and useful can separate P from NP under standard pseudorandomness assumptions.",
  "prediction_chain": []
}
```

The target anchor (`PROOF-clay-p-ne-np` above) **stays `status:
"untested"`** no matter how many of these point at it. A purely
informational link from the target to the barrier anchors it's known to
face (so "why hasn't anyone just tried X" is answerable) is a reasonable
small addition — tracked as a proposal on #126, not a `Status` change.

### 6.4 Formal nodes tied to Lean CI

This needs nothing new — it's exactly §3 above (`lean_theorem` +
`theorems[]` + `verification`), the same mechanism every programme uses.
A programme whose own CI already pins required theorems (e.g. a
`<programme>-gate.json` with a `required_theorems` list) is the *right*
way to guarantee a node marked formal always points at something the
kernel actually checked — that's enforcement living in the consumer repo,
same as QBP's `root_audit.py` (see "Who owns what here" in the top-level
README).

## 7. Canonical vs. vendored — if you're adding a new programme

This repo holds the **one canonical copy** of `schema/inventory.schema.json`.
Every consuming programme (QBP, Gleipnir, ...) vendors a **byte-identical
copy** into its own repo plus a sidecar sha recording which canonical
version it matches. Never fork or locally modify the vendored copy — if
your programme needs a field this schema doesn't have, that's a design
surface against *this* repo (open an issue here, like #126), not a
local patch.

**New-programme checklist:**

1. Write your inventory against the current canonical schema (check
   `x-schema-semver` in `schema/inventory.schema.json`'s top-level
   `$comment`/description for the latest fields).
2. Vendor a byte-identical copy into your own repo with a sidecar sha.
3. If a claim doesn't fit the existing vocabulary, don't invent a local
   extension — open a design-surface issue here first (see #126 for the
   template this guide grew out of).
4. If you want CI enforcement (a reachability gate, an orphan check, a
   required-theorems pin), that script lives in *your* repo, reading your
   vendored schema copy — not in this one.
