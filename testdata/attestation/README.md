# `testdata/attestation/` — Evidence Attestation Verifier fixtures (inter#149)

The verifier interface + v0 stub live in package `attestation`; the four fake
verifiers and the shared `Contract(t, name, Verifier)` runner live in package
`attestation/attestationtest`.

**Why the fakes are in `attestationtest/`, not here.** The Go tool excludes
`testdata/` from builds, so a package placed here cannot be imported. The fakes
and the contract runner must be importable by consumers in *other* repos (the
QBP ledger CI, the qbp-compute-unit gate) so they can run the identical contract
against whatever `Verifier` they wire — proving the v0 → v1 → v2 swap never
changes caller code. So the executable test support is importable; this
directory holds the record-level **fixture data** the fakes operate on.

*(Placement flagged for §I4: inter#149 named `testdata/attestation/` for "the
shared contract tests"; the contract runner is importable for cross-repo reuse,
and the fixtures that pin each fake's verdict live here. Repoint if preferred.)*

## Files
- `sample_record.json` — one evidence record plus the `verify()` Result each
  fake returns for it. The contract asserts only interface-shape invariants
  (role/method enums, non-empty signer, unsigned⇒¬verified, determinism); these
  per-fake verdicts document the concrete behaviour consumers depend on.

## The contract (every Verifier must satisfy)
- `role ∈ {notary, beekeeper, none}`
- `method ∈ {unsigned, gitsign, ssh, yubikey}`
- `signer` non-empty (a seat-id, or `unknown`)
- an `unsigned` record is never `verified`
- `Verify` is deterministic for a given `(record, claimed_signer)`

Under the v0 stub every record is `{verified:false, signer:unknown, role:none,
method:unsigned}` — so a consumer's truth guarantee is **re-execution of the
evidence, never the signature** (confluent-trust#108 AC6).
