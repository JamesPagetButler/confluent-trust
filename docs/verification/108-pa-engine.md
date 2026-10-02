# confluent-trust#108 — PA grading engine verification record

Closes #108.

This is the acceptance-criteria → test → mutant record for the proof-assistant (PA)
grading engine (inter#150 criticality grading), landed via #109 (Strike 0,
attestation `verify()` + shared contract) and #110 (Strike 1, the engine + schema +
fixtures). The issue's original matrix named `TestPA_*` tests; the build used
`TestFixtures/<file>` sub-tests in `internal/pa`, `TestSchema_*` in
`internal/validate`, and the attestation contract tests. This file is the mapping
of record.

**Core principle — derived, never declared.** The engine grades a claim only from
engine-**derived** evidence, never a producer's **declared** fields. Every guard
below *requires* a good value and fails closed on a missing one. Each guard, removed
in turn, is caught by exactly the named test; the suite is otherwise green and the
source is restored → green (source-mutation harness, run per guard).

## AC1 — additive `proof_assistants` schema (`internal/validate`)

`proof_assistants[]` with `assistant ∈ {lean4,coq,agda}`, `evidence_ref`,
`trust_check` on proof/derivation anchors; additive, old records still validate.

| Check | Test | Mutant → result |
|---|---|---|
| old records still validate (additive) | `TestSchema_AdditiveOldRecordsValidate` | make `proof_assistants` required → **KILLED** |
| a valid proof_assistants entry is accepted | `TestSchema_AcceptsValidProofAssistant` | — (acceptance test; no single-guard mutant) |
| assistant enum is closed | `TestSchema_RejectsUnknownAssistant` | widen the enum → **KILLED** |
| required sub-fields enforced | `TestSchema_RejectsProofAssistantMissingRequired` | drop `proof_assistants` from both schema copies → **KILLED** |
| trust_check enum enforced | `TestSchema_RejectsBadTrustCheck` | widen the trust_check enum → **KILLED** |

Schema mutants re-run independently on `main` by qbp-architecture: widening the
assistant enum → KILLED; dropping `proof_assistants` from either the canonical or
the embedded copy → KILLED (byte-identity).

## inter#149 — attestation `verify()` contract (`attestation`)

v0 stub (every record unsigned; truth guarantee is re-execution). Consumers depend
only on the contract, so v0→v1→v2 never changes caller code.

| Check | Test |
|---|---|
| the shared contract holds for the stub + all fakes | `TestVerifierContract` |
| v0 verdict pinned (unsigned/unknown/none/false) | `TestStubV0_AlwaysUnsigned` |
| the fakes model the distinct AC6 cases | `TestFakes_DistinctVerdicts` |
| the contract itself catches bad verifiers | `TestContract_RejectsBadVerifiers` |

`Contract` meta-tests: each of the 5 contract checks (role-in-set, method-in-set,
non-empty signer, unsigned⇒!verified, determinism), removed in turn, reddens exactly
its sub-test of `TestContract_RejectsBadVerifiers` — **all KILLED** (confluent-trust#109 §I4).

## AC2–AC6 + fail-closed guards + staleness (`internal/pa`)

Grade from `GradeClaim` (PA0/PA1/PA2); effective PA from `EffectivePA` over
derivation edges only. Fixture oracle: exact grade + exact flag-set per fixture;
`expected_pa` is stripped structurally (`pa.Claim` has no such field).

| AC / req | Guard (mutant = remove it) | Caught by | Result |
|---|---|---|---|
| AC2 | assistant counts clean | `TestFixtures/01_lean_clean`, `04_agda_safe` | — |
| AC2 | trust_check must equal `pass` | `TestFixtures/P5_trust_check_missing`, `P5b_trust_check_pending`; `TestTrustCheckVetoOnly` | **KILLED** |
| AC2/C2 | grade from trust_check alone (M-tc) | `TestFixtures/03_coq_undeclared_vm_compute` | **KILLED** |
| AC2/R3 | output-hash reproduce (missing/mismatch) | `TestFixtures/07b_single_hashmismatch`, `P4_missing_declared_hash` | **KILLED** |
| AC3 | kernel-cleanliness derived in-engine | `TestFixtures/02_lean_native_decide` | **KILLED** |
| AC3/R1 | lean4 axiom whitelist | `TestFixtures/P1_axiom_ofReduceBool`, `P1b_axiom_sorryAx` | **KILLED** |
| AC3/R1 | compiled-path tactic scan | `TestFixtures/P7_coq_declared_vm_compute` | **KILLED** |
| AC3/R1 | coq/agda closed-context | `TestFixtures/P8_coq_open_context`, `P8b_agda_postulate` | **KILLED** |
| AC3/R1 | unknown-assistant fail-closed | `TestFixtures/P9_unknown_assistant` | **KILLED** |
| AC3/R1 | kernel_clean keep-reject | `TestFixtures/P10_kernel_clean_false` | **KILLED** |
| AC3 (derive) | declared≠derived guard | `TestFixtures/03_coq_undeclared_vm_compute` | **KILLED** |
| AC4 | derivation-only edges in min | `TestEffectivePA_Chain` | **KILLED** |
| AC5 | correspondence requirement | `TestFixtures/05_uncorresponded_pair`; `06_valid_pair_fano` | **KILLED** |
| AC5/C1 | non-self correspondence | `TestFixtures/13_self_attested_correspondence` | **KILLED** |
| AC6 | role rejection | `TestFixtures/11a_verified_wrong_role`; `11b`/`11c` (require_signature) | **KILLED** |
| AC6 | self-attestation (signer≠producer) | `TestFixtures/14_self_attested_signer` | **KILLED** |
| R2 | re-execution success (exit 0) | `TestFixtures/P2_reexec_failed` | **KILLED** |
| R3 | no-evidence (empty derived hash) | `TestFixtures/P6_both_hashes_empty` | **KILLED** |
| notary#3 | staleness (fail-closed) | `TestFixtures/15_stale_source`, `P3_missing_source_sha`; `TestStaleWhilePassing` | **KILLED** |

`expected_pa` is never read: `TestEngineIgnoresExpectedPA` plants a wrong value and
asserts the computed grade == the true grade ≠ the planted value (the mutant
"engine reads expected_pa" is structurally inexpressible — `pa.Claim` has no field).

## notary-request v1.3 emit record (`internal/pa`, `testdata/pa/notary-request.schema.json`)

The engine returns the locked notary#3 v1.3 record; the caller emits. Canonical
schema lives here (architecture R4-Q4); others vendor byte-identically.

| Check | Test |
|---|---|
| the shared sample validates against the canonical schema | `TestNotaryRequestSeam_SampleValidates` |
| the reason-conditional bites (nullability, enums, sha pattern) | `TestNotaryRequestSeam_ConditionalBites` |
| the optional `detail` field validates (locked v1.3) | `TestNotaryRequestSeam_DetailAccepted` |
| the engine's own returned record conforms | `TestNotaryRequestSeam_EngineEmitsConforming` |
| dedupe key = (claim_id, source_sha, reason, artifact_digest) | `TestDedupeKey` |
| shortfall/staleness shaping | `TestRequestFor` |

## Quality gate

`gofmt` + `go vet ./...` + `golangci-lint run ./...` (0 issues) + `go test -race ./...`
all green on `main`. The mutation tables above were produced by a source-mutation
harness: each guard removed → its named test red, suite otherwise green, source
restored → green.

## Follow-ups (tracked, not blocking the close)

- confluent-trust#112 — schema 0.3.6: restrict `proof_assistants` to proof/derivation
  anchors (close the C1 description/enforcement gap).
- confluent-trust#111 — upstream adoption of `pa_local`/`pa_effective` (CI-recomputed,
  drift-fails-CI).
- Engine doc-comments for the composite-manifest staleness key and the
  grade-attaches-to-own-statement principle (bundle with 0.3.6).
