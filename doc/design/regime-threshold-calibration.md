# Regime Threshold Empirical Calibration Protocol

**Status:** Design surface — §I4 invariant applies (see §0)
**Sprint:** sprint-1-closeout-2026-05-17 seq=12 (Notary-bootstrap target #1)
**Issue:** #49
**Last updated:** 2026-05-21

---

## §0 §I4 Invariant — Named Readers

This document is a **design surface**, not a calibration run. No thresholds are changed here.
The calibration run is gated on production data availability (see §7).

**Named readers** whose review is requested before any threshold-update PR is opened:

| Reader | Role |
|--------|------|
| `@bma-implementor` | Continuous-loop owner; produces live scoring telemetry post-Toddle |
| `@qbp-implementor` | Data source candidate; owns `archive/cth-inventory/` inventory |
| `@notary-implementor` | Verification-discipline reviewer; owns Notary competency #2 |
| beekeeper | Final sign-off for any threshold change that affects existing inventory status classifications |

---

## §1 Motivation

`compute.ScorePrediction` in `compute/scoring.go` classifies a prediction-vs-observation delta
into one of four regimes:

| Regime | Threshold | Status mapping |
|--------|-----------|----------------|
| `ScoreRegimeLaminar` | delta < 1% | `coherent` |
| `ScoreRegimeLowSediment` | 1% ≤ delta < 10% | `coherent` (annotated) |
| `ScoreRegimeModerate` | 10% ≤ delta < 50% | `contested` |
| `ScoreRegimeHeavy` | delta ≥ 50% | `refuted` |

The current thresholds (1% / 10% / 50%) originate from **BMA Theory v0.2 §4.4 vocabulary** —
they were chosen to map cleanly onto the prose description of cognitive states (laminar flow,
sedimentation, turbulence), not calibrated against any empirical prediction distribution.

This is an acknowledged design debt. CTH issue #71 (cth-implementor review §1.b) notes:

> "The thresholds are not empirically calibrated. They are borrowed from BMA Theory v0.2 §4.4
> vocabulary and chosen to feel right. Production observation distributions may have different
> shapes."

The `testdata/predictions_lifecycle.json` fixture exercises all four regimes with deltas
hand-chosen to land cleanly in each band (0.07%, 2.38%, 16.67%, 66.67%). These are illustrative,
not representative of the QBP prediction record.

**Why this matters:** If the production distribution is heavily skewed — e.g., most physics
predictions land at sub-1% (measurement-precision-limited) or most predictions are untested
— then the current thresholds may produce uninformative regime classifications. A regime that
contains 90% of all observations signals no structural information about prediction quality.

---

## §2 Data Sources

Three candidate datasets exist for calibration work, at varying levels of readiness.

### §2.1 QBP Historical Prediction Record

**Location:** `~/Documents/QBP/archive/cth-inventory/` (two tracked snapshots)
- `confluent-trust-inventory-v5.13.json` — 150 anchors (federation-tenancy stream, 2026-04-30)
- `confluent-trust-inventory-v5_3.json` — 141 anchors (Session-13 closeout, 2026-05-11)

**Calibratable pairs:** As of 2026-05-21, the two inventories together yield **44 unique
anchor pairs** that carry both `predicted_value` and `measured_value` with a non-zero
predicted value.

**Observed distribution (preliminary, pre-calibration-run):**

| Regime (current thresholds) | Count | Fraction |
|-----------------------------|-------|----------|
| Laminar (< 1%) | 23 | 52% |
| LowSediment (1%–10%) | 11 | 25% |
| Moderate (10%–50%) | 2 | 5% |
| Heavy (≥ 50%) | 8 | 18% |

**Percentiles:**
- P25: 0.00% (exact match)
- P50: 0.72% (within Laminar)
- P75: 3.42% (LowSediment)
- P90: 71.05% (Heavy)
- P95: 84.21% (Heavy)

**Key observation:** Over half of QBP predictions currently land in Laminar at the current
thresholds. The P50 is 0.72% — within the Laminar band. The Heavy tail is bimodal:
8 pairs (18%) exceed 50%, with at least one extreme outlier (delta ~142,857%) that likely
represents a unit-mismatch or a prediction that was never seriously expected to be close.

This preliminary distribution is informative but **not yet sufficient** to trigger a
threshold change: the dataset (n=44) is small, the extreme outlier needs audit, and it
does not yet include BMA continuous-loop data.

**Minimum threshold for a calibration run:** n ≥ 30 calibratable pairs with audited
unit consistency. The QBP inventory is at the margin; a unit-consistency audit pass is
required before using it for calibration decisions.

### §2.2 BMA Continuous-Loop Predictions

**Location:** BMA continuous-loop telemetry (post-Toddle, when L3 Beliefs produces live scores)
**Readiness:** Not yet available. Gated on Toddle phase activation (BMA Step 9+).

This is the primary target dataset. BMA's continuous-loop will produce prediction-vs-observation
pairs at a much higher volume and with more consistent provenance than the QBP inventory.
Each `WDEvent → CTH ρ_net feedback loop` (BMA #107) and each `NT_SIGNAL → CTH PRED-*` flow
(Wyrd PR #35) contributes a calibratable pair when the corresponding observation arrives.

**Expected data shape:** Unknown until Toddle. May skew more heavily Laminar (BMA predictions
are self-correcting) or may show wider spread depending on the domains BMA is predicting.

### §2.3 Synthetic Test Fixtures

**Location:** `testdata/predictions_lifecycle.json`, `testdata/predictions_lifecycle_v0_3.json`

These fixtures are **not calibration data** — they are specifically engineered to exercise
each regime cleanly. Their deltas (0.07%, 2.38%, 16.67%, 66.67%) are chosen to demonstrate
the API, not to represent the empirical distribution.

They are included here as a reference for the regime structure, not as input to the
calibration run.

---

## §3 Calibration Approaches — Three Candidates

Three approaches exist for setting regime thresholds. Each has distinct tradeoffs.

### §3.1 Fixed-Percentile Approach

**Mechanism:** Collect all observed deltas. Sort. Set thresholds at fixed percentiles —
for example:
- Laminar upper bound = P25 (bottom quartile: "well-predicted")
- LowSediment upper bound = P50 (median: "typical agreement")
- Moderate upper bound = P75 (third quartile: "notable discrepancy")
- Heavy: everything above P75

**Pros:**
- Distribution-shape-adaptive. If predictions get more precise over time, the thresholds
  automatically tighten.
- Ensures roughly equal population across regimes (by construction at steady state).
- No domain knowledge required — purely data-driven.

**Cons:**
- **Retroactive reclassification on every recalibration.** Because thresholds shift as the
  data distribution shifts, a prediction classified as "contested" in 2026 may be
  reclassified as "coherent" in 2027 without any new observation. This breaks audit trails.
- **Gaming risk.** If BMA generates many easy-to-confirm predictions, the Laminar band
  inflates, making genuinely hard predictions look worse by comparison.
- **Semantic instability.** Regime names (Laminar, Heavy) carry cognitive meaning. Percentile
  anchoring decouples the name from its meaning.
- **n-sensitivity.** At n=44 (current QBP inventory), percentile estimates are noisy.
  P25 and P50 are inside the Laminar band under current thresholds — a fixed-percentile
  approach would produce a drastically different threshold set from a small, possibly
  unrepresentative sample.

**Verdict:** Not recommended as primary method. The retroactive-reclassification failure
mode is incompatible with CTH's verification-discipline goals.

### §3.2 K-Means Clustering

**Mechanism:** Apply K-means clustering (K=4) to the scalar log-delta distribution.
Use cluster boundaries as thresholds.

Rationale for log-delta: the delta distribution is likely log-normal (multiplicative errors).
Working in log space gives better-behaved cluster separation.

**Pros:**
- Data-driven cluster structure — if the distribution is genuinely bimodal or multimodal,
  K-means will find it.
- Does not presuppose percentile shape.
- Can be run as a diagnostic without committing to its output.

**Cons:**
- **K=4 is an a priori choice** that biases the analysis. If the natural cluster count is
  K=2 or K=3, forcing K=4 introduces spurious boundaries (see §8, open question #1).
- **Sensitive to outliers.** The extreme outlier in the QBP inventory (delta ~142,857%)
  will pull a cluster centroid hard unless excluded — but excluding it requires a
  judgment call about what constitutes a "legitimate" calibration pair.
- **Same retroactive-reclassification risk** as fixed-percentile if thresholds are updated
  frequently.
- **Uninformative on small n.** At n=44, K-means is fitting 4 clusters to 44 points —
  the centroids will be noisy.

**Verdict:** Valuable as a **drift diagnostic** (see §3.3 hybrid recommendation), but
not suitable as the primary threshold-setting mechanism.

### §3.3 Domain-Anchor Approach

**Mechanism:** Set thresholds at semantically meaningful values with explicit domain
justification. The current thresholds are an instance of this approach, but without
documented rationale.

Example rationale for the current thresholds:

| Threshold | Value | Domain justification |
|-----------|-------|---------------------|
| Laminar upper bound | 1% | Typical instrumental measurement error in precision physics. A prediction within 1% is "inside the noise floor" for most QBP measurements. |
| LowSediment upper bound | 10% | Typical calibration tolerance. A 10% discrepancy is "outside the noise but within order of magnitude." |
| Moderate upper bound | 50% | Factor-of-2 discrepancy. At >50%, the prediction and observation are "qualitatively different" — the theory has a structural problem, not just a calibration problem. |

**Pros:**
- **Stable across recalibrations.** Thresholds do not shift as data accumulates, so
  historical classifications are preserved.
- **Semantically grounded.** The regime names carry consistent meaning across time.
- **Resistant to gaming.** Easy predictions cannot inflate the Laminar band.
- **Audit-friendly.** Each threshold is a documented claim about domain physics, not
  a statistical artifact.

**Cons:**
- Returns to hand-picked values — the "empirical calibration" is just documented judgment.
- May be wrong for a specific domain. QBP physics predictions at the precision frontier
  may need tighter thresholds than synthetic/programming predictions.
- **Cross-domain calibration problem** (see §8, open question #2): a 1% error in a
  fine-structure constant measurement is fundamentally different from a 1% error in a
  software performance prediction.

**Verdict:** Recommended as the primary approach, with the K-means analysis as a
**drift signal** (see §3.4).

### §3.4 Recommended Hybrid Approach

**Recommendation: domain-anchor thresholds as primary, K-means as drift signal.**

The operational protocol:

1. **Maintain current thresholds** (1% / 10% / 50%) as the production values. These are
   the domain-anchor thresholds. They are not empirically wrong — the QBP inventory
   preliminary data (§2.1) shows the current thresholds do produce a non-degenerate
   distribution (no regime is empty).

2. **Run the K-means analysis** (§4) on each calibration dataset to produce cluster
   centroids. This is a diagnostic, not a decision.

3. **Trigger condition for threshold review:** If K-means cluster centroids (in log-delta
   space) differ from the domain-anchor thresholds by more than ±20% of the threshold
   value (i.e., the K=4 cluster centroid nearest each boundary is more than 20% away from
   the boundary), open a threshold-review PR.

4. **Threshold-review PR** (§5) requires the full governance cycle (§I4 readers + beekeeper).
   New thresholds must be domain-justified, not purely percentile-derived.

**Rationale for ±20% trigger:** At the current thresholds (1%, 10%, 50%), a ±20% window means:
- Laminar boundary: trigger if K-means suggests a boundary outside [0.8%, 1.2%]
- LowSediment boundary: trigger if K-means suggests a boundary outside [8%, 12%]
- Moderate boundary: trigger if K-means suggests a boundary outside [40%, 60%]

These windows are wide enough to absorb small-sample noise but narrow enough to catch
genuine structural misalignment between the thresholds and the actual prediction distribution.

---

## §4 Calibration Data Pipeline

The following protocol governs a calibration run. Steps 1–3 are data preparation;
step 4 is analysis; step 5 is the decision.

### Step 1 — Collect calibratable pairs

Sources (in priority order):

**Primary:** BMA continuous-loop telemetry (post-Toddle)
- Extract `predicted_value` + `measured_value` from all scored anchors with status
  in {`coherent`, `contested`, `refuted`} (exclude `untested`).
- Minimum n: 30 pairs per domain category (physics, programming, synthetic).
- Provenance tag: `source=bma_continuous_loop` + timestamp of the scoring run.

**Secondary:** QBP `archive/cth-inventory/*.json`
- Collect `predicted_value` + `measured_value` from all anchors where both fields
  are populated and `predicted_value != 0`.
- Current count: 44 unique pairs across both tracked snapshots (as of 2026-05-21).
- **Required pre-step:** unit-consistency audit. Confirm that each pair uses the same
  unit for both values. Flag outliers (delta > 100×) for manual inspection before
  including in the calibration dataset.
- Provenance tag: `source=qbp_inventory` + snapshot filename + sha.

**Tertiary:** Additional programme inventories as they are added to CTH.

### Step 2 — Compute deltas

For each calibratable pair (predicted_value `p`, measured_value `m`):

```
delta = |p - m| / |p|
```

This is the same formula as `compute.ScorePrediction` for `KindScalar`.

Exclude pairs where:
- `p == 0` (ill-defined; matches `ErrZeroPrediction`)
- `delta > 100` (likely unit mismatch; flag for manual audit)
- Provenance is `synthetic` (test fixtures; do not contaminate calibration data)

### Step 3 — Run analysis

Three outputs are required:

**Output A — Histogram:**
Plot `log10(delta * 100)` (log-scale discrepancy percentage) with bin width 0.25.
Annotate current threshold positions (log10(1), log10(10), log10(50)).

When the Theory Cart Python tool is available, use:
```python
import matplotlib.pyplot as plt
import numpy as np

log_deltas = np.log10([d * 100 for d in deltas if d > 0])
plt.hist(log_deltas, bins=40, edgecolor='black')
for t in [1, 10, 50]:
    plt.axvline(np.log10(t), color='red', linestyle='--', label=f'{t}%')
plt.xlabel('log10(discrepancy %)')
plt.ylabel('count')
plt.title('Delta distribution vs current regime thresholds')
plt.savefig('regime_calibration_histogram.png', dpi=150)
```

For manual analysis (before Python tool is available), compute and record:
- Total n
- Fraction of pairs in each regime under current thresholds
- P10, P25, P50, P75, P90, P95, P99 of the delta distribution

**Output B — K-means analysis (K=4, log-delta space):**
Apply K-means to `log10(delta * 100)` for delta > 0 (zero-delta pairs excluded — they
represent exact matches, which should always be Laminar and do not inform threshold
boundaries).

Record:
- Cluster centroids (back-transformed to percentage)
- Nearest threshold to each centroid
- Distance from nearest threshold as a fraction of threshold value

**Output C — Summary table:**

| Metric | Value |
|--------|-------|
| n (total pairs) | |
| n (delta > 0) | |
| Current Laminar fraction | |
| Current LowSediment fraction | |
| Current Moderate fraction | |
| Current Heavy fraction | |
| P50 delta | |
| K-means centroids (K=4) | |
| Max centroid-to-threshold distance | |
| Trigger condition met? | Yes / No |

### Step 4 — Update decision

Apply the hybrid trigger condition (§3.4):

- **No action** if all K-means centroids are within ±20% of their nearest domain-anchor
  threshold. Record the analysis results in a commit comment on the relevant issue.
  Tag the commit with `calibration-check-YYYY-MM-DD`.

- **Open threshold-review PR** if any centroid exceeds the ±20% window. The PR body
  must include:
  - The full Output C summary table
  - A histogram image (or raw percentile data if no plotting tool)
  - Proposed new threshold values with domain justification
  - Dataset signature: date, source filenames, sha/provenance tags, n
  - Cross-reference to this design doc

---

## §5 Threshold-Update Governance

Threshold changes to `compute.ScorePrediction` are **breaking changes** that affect:
- Regime classification of all existing and future scored predictions
- Status mapping (`coherent` / `contested` / `refuted`) for anchors in inventory files
- Federation consumers who depend on stable regime semantics

The following governance protocol applies:

### §5.1 PR requirements

A threshold-update PR to `compute/scoring.go` must:

1. **Bear the `repo-confluent-trust` label** (marks it as a semantics-change PR)
2. **Carry the §I4 reader list** in the PR body (§0): `@bma-implementor`,
   `@qbp-implementor`, `@notary-implementor`, beekeeper
3. **Include the calibration analysis artifact** (Output C table + histogram or percentile
   data) as an attached file or inline in the PR body
4. **Record the dataset signature:** date of calibration run, source files, sha/provenance
   tags, n, unit-consistency audit result
5. **State old and new thresholds explicitly:**

   ```
   | Boundary | Old value | New value | Justification |
   |----------|-----------|-----------|---------------|
   | Laminar → LowSediment | 1% | X% | ... |
   | LowSediment → Moderate | 10% | Y% | ... |
   | Moderate → Heavy | 50% | Z% | ... |
   ```

6. **Flag fixture migration requirement** if `testdata/predictions_lifecycle.json` or
   `testdata/predictions_lifecycle_v0_3.json` delta values fall outside the new regime
   boundaries (they were chosen to land cleanly in each band). These fixtures must be
   updated in the same PR or in an immediately following PR.

### §5.2 `cth migrate` integration

If a threshold change causes existing inventory files to reclassify anchors across status
boundaries (e.g., an anchor previously `coherent` becomes `contested`), this is a
**migration event** for `cth migrate` (v0.4 candidate).

The migration command should:
- Accept `--old-thresholds` and `--new-thresholds` flags
- Re-score all anchors with both `predicted_value` and `measured_value`
- Report which anchors change status
- Update `discrepancy_pct` and `status` fields in-place if instructed

Federation consumers must be notified before a threshold migration is deployed. The
migration is a **breaking change** under the federation versioning protocol.

### §5.3 Approval requirements

| Approver | Required for |
|----------|-------------|
| `@bma-implementor` | Any threshold that affects BMA continuous-loop classification |
| `@qbp-implementor` | Any threshold that reclassifies existing QBP inventory anchors |
| `@notary-implementor` | All threshold changes (Notary owns verification discipline) |
| beekeeper | All threshold changes (constitutional gate per BMA Governance §3) |

**Veto right:** Any named reader may raise a `MAJOR_CONCERN` or `VETO` on the threshold
change. Beekeeper has final escalation authority per BMA Governance succession protocol.

---

## §6 Notary-Discipline Angle

The Notary function (BMA Theory v3.0 §3) verifies that published claims match their
stated evidence. Regime threshold governance is a natural Notary competency surface.

**Notary competency #2 applied to calibration:**

When a threshold-update PR is opened, Notary:

1. **Receives the dataset signature** from the PR body (date, source files, sha, n).
2. **Independently re-runs the analysis pipeline** (§4 Steps 1–3) against the cited
   dataset.
3. **Compares outputs:** Notary's Output C table vs. the PR body's Output C table.
   If they match within floating-point tolerance, Notary produces a `VERIFIED` record.
   If they differ, Notary raises a `DISCREPANCY` flag with the diff.
4. **Records the verification event** in the Notary ledger with the PR number, dataset
   signature, and verification result.

**Cross-check invariant:** Notary verifies that the *proposed thresholds are consistent
with the cited calibration dataset* — not that the thresholds are optimal. Threshold
optimality is a domain judgment; dataset-to-claim consistency is a Notary claim.

**Subscription:** Notary subscribes to the `repo-confluent-trust` label in the CTH
repository. Any PR bearing that label and modifying `compute/scoring.go` triggers the
verification workflow.

**This is Notary competency #2-adjacent:** cross-check actual data (calibration dataset)
against published claim (proposed threshold values). It does not require Notary to have
access to the continuous-loop at verification time — the PR body must include the full
dataset or a deterministic pointer to a content-addressed snapshot.

---

## §7 Sprint Planning — Calibration Timeline

### Now (sprint-1-closeout to sprint-2)

This design surface PR opens. No thresholds change. No calibration run.

### Calibration run trigger conditions (whichever is earlier)

**Trigger A — BMA continuous-loop active:**
BMA Toddle phase (post-Step 9) is running and has accumulated ≥ 30 scored
prediction-vs-observation pairs from the continuous-loop telemetry.

**Trigger B — QBP inventory threshold:**
`archive/cth-inventory/` contains ≥ 30 unique calibratable pairs that have passed a
unit-consistency audit. (As of 2026-05-21, we are at n=44 total but audit is not yet
complete. An audit pass could trigger this condition in sprint-2.)

### When triggered

A follow-up PR runs the protocol defined in §4. The PR either:
- Confirms no-change (K-means centroids within ±20% of current thresholds) and records
  the calibration artifact, or
- Proposes new thresholds with the full governance cycle (§5).

### Cadence thereafter

Not fixed. The calibration run is **data-volume-triggered**, not calendar-triggered.
When the running dataset doubles in size from the last calibration run, a new run is
warranted. This is tracked via a `calibration-last-run` tag in the relevant issue.

---

## §8 Open Questions

The following questions are deferred to the calibration run or to beekeeper decision.
They are recorded here for the named readers (§0).

### OQ-1: K=4 a priori

The K-means analysis (§3.2) uses K=4 matching the four named regimes. However, the
empirical distribution may have K=2 or K=3 natural clusters (e.g., "clearly right /
borderline / clearly wrong"). Should the analysis also run K=3 and K=5 as sensitivity
tests?

**Tentative answer:** Yes, run K=3 and K=5 as diagnostics. If K=3 produces better
silhouette scores than K=4, it is evidence that the four-regime structure is over-fitted
and the domain-anchor justification needs revisiting.

### OQ-2: Cross-domain calibration

Physics predictions at the precision frontier (QBP fine-structure constant: delta < 0.1%)
may be structurally different from programming/software predictions (e.g., performance
regressions where 10% is "within noise"). Should domain-specific thresholds be supported?

**Tentative answer:** Deferred. The current `ScoreRegime` type has no domain tag.
Adding per-domain thresholds requires a schema change (`PredictionKind` extension or a
new `PredictionDomain` type). This is a Walk-phase consideration. For Crawl/Toddle,
a single threshold set is adequate.

**Risk if deferred:** The QBP inventory skews heavily Laminar because physics predictions
are precision-constrained. If BMA continuous-loop predictions span a wider domain range,
mixing the distributions may produce misleading calibration results. Flag this during
the calibration run.

### OQ-3: Threshold-update cadence

Annual? Per-sprint? Data-volume-triggered (§7)?

**Tentative answer:** Data-volume-triggered, with a minimum floor of one calibration
check per major BMA phase transition (Crawl → Toddle, Toddle → Walk).

### OQ-4: Calibration dataset retention and staleness

At what point does old calibration data become stale enough to exclude? Physics constants
change very slowly; BMA predictions from an early-Toddle period may not represent the
distribution at mid-Walk.

**Tentative answer:** Use a sliding window of the most recent 200 calibratable pairs
(or all available pairs if fewer than 200). Tag each calibration run with a `window_start`
and `window_end` date.

### OQ-5: Federation rollout

Does each CTH tenant calibrate independently, or does CTH ship federation-canonical
thresholds?

**Tentative answer:** CTH ships canonical thresholds. Tenant overrides are possible via
a configuration flag (deferred to Walk-phase federation work). Until then, all tenants
use the same `compute/scoring.go` thresholds. This is consistent with the current design
where thresholds are compile-time constants.

---

## §9 Cross-References

| Reference | Notes |
|-----------|-------|
| CTH #71 thread — cth-implementor review §1.b | Original note: "thresholds are not empirically calibrated" |
| sprint-1-closeout-2026-05-17 seq=12 | Parent sprint; this is Notary-bootstrap target #1 |
| `compute/scoring.go` | Current threshold implementation (`scoreClassifyRegime`) |
| BMA Theory v0.2 §4.4 | Source of hand-picked vocabulary (Laminar / LowSediment / Moderate / Heavy) |
| BMA Theory Addendum 18 §2.4 | Scoring loop: "predictions are recorded, observed values arrive over time, the delta is computed and stored" |
| BMA Theory v3.0 §3 | Notary function — competency #2 (cross-check actual data against published claim) |
| `testdata/predictions_lifecycle.json` | Synthetic fixture; illustrative, not calibration data |
| `testdata/predictions_lifecycle_v0_3.json` | v0.3 schema fixture; same caution applies |
| `~/Documents/QBP/archive/cth-inventory/README.md` | QBP inventory provenance + usage notes |
| CTH #49 | This design's parent issue |
