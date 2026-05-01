#!/usr/bin/env bash
# Seed all 27 issues for JamesPagetButler/confluent-trust from the
# CTH Go Implementation Plan §3.3.
# Idempotent guard: skips creation if an issue with the same title already exists.
set -euo pipefail

REPO="${REPO:-JamesPagetButler/confluent-trust}"
M_CRAWL=1   # v0.1.0 — Crawl Complete
M_WALK=2    # v0.2.0 — Walk Complete
M_RUN=3     # v0.3.0 — Run Complete

create() {
  local title="$1" milestone="$2" labels="$3" body="$4"
  if gh issue list -R "$REPO" --search "in:title \"$title\"" --state all --json title -q '.[].title' \
       | grep -Fxq "$title"; then
    echo "skip: $title"
    return
  fi
  gh issue create -R "$REPO" \
    --title "$title" \
    --milestone "$milestone" \
    --label "$labels" \
    --body "$body" \
    >/dev/null
  echo "created: $title"
  sleep 1
}

# ---------- 1 ----------
create "Repository scaffolding" "v0.1.0 — Crawl Complete" "type:infra,phase:crawl,priority:p0" "$(cat <<'EOF'
Set up the repository structure:

- `go.mod` with module path `github.com/JamesPagetButler/confluent-trust`
- Package directories: `model/`, `compute/`, `store/`, `report/`, `cmd/cth/`, `testdata/`
- `.gitignore` for Go
- `LICENSE` (Apache 2.0)
- `README.md` with project overview referencing the theory doc
- CI: GitHub Actions workflow running `go test ./...` on push
- Linting: `golangci-lint` config

**Acceptance:** `go build ./...` and `go test ./...` pass on empty packages.
EOF
)"

# ---------- 2 ----------
create "Core data model — model/ package" "v0.1.0 — Crawl Complete" "type:model,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement the data types from Theory v0.2 §4.1:

- `Anchor` struct with fields: ID, Name, Tier, Derivable, Status, Provenance, Domain, ResidualEntropyBits, ConfirmatoryInfoBits, Description, PredictionChain, LastTestedAt
- `Chain` struct with: ID, SourceIDs, TargetID, Steps, WeakestLinkID, Fidelity, Status, DomainBoundaries
- `DomainBoundary` struct with: FromDomain, ToDomain, AtAnchorID, Fidelity, Hypothesis
- `ChainRef` struct with: ChainID, Programme, Provenance (enum: Internal/External/CrossProgramme), Fidelity, Summary
- `ConfluencePoint` struct with: ID, AnchorID, Paths ([]ChainRef), MutualInfoBits, Status
- `Inventory` struct as the top-level container: Programme, Version, Axioms, DerivedPrinciples, Anchors, Inputs, Chains, ConfluencePoints, Health
- Enums: `Tier`, `Status`, `Provenance`, `ChainProvenance`
- Tier 0 derivability invariant: validation function that rejects Tier 0 anchors with `Derivable: true`

JSON tags for serialisation/deserialisation matching the existing inventory format.

**Acceptance:** Round-trip: load `testdata/qbp_v3_2.json` → serialise → deserialise → deep equal.
EOF
)"

# ---------- 3 ----------
create "Test fixtures — synthetic and regression inventories" "v0.1.0 — Crawl Complete" "type:test,phase:crawl,priority:p0" "$(cat <<'EOF'
Create test fixture inventories in `testdata/`:

1. `minimal.json` — 5 anchors (1 axiom, 2 proofs, 1 measurement, 1 prediction), 2 chains, 1 confluence. All values hand-computed. Purpose: unit test baseline.
2. `qbp_v3_2.json` — Full QBP v3.2 inventory. Purpose: regression test against known Python engine outputs.
3. `qbp_quantum_v0_1.json` — QBP-Quantum v0.1 inventory (updated for v0.2 schema: N-ary confluences, ChainRefs with provenance). Purpose: merge test.
4. `known_values.go` — Expected outputs for all compute functions against each fixture (hand-verified or Python-verified).

**Blocked by:** #2

**Acceptance:** All fixtures parse without error. Known values documented for every compute function.
EOF
)"

# ---------- 4 ----------
create "Residual entropy and confirmatory information — compute/entropy.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definitions 7 and 7a:

- `ResidualEntropy(a Anchor) float64` — η(v) per tier, handling δ=0 case correctly
- `ConfirmatoryInfo(a Anchor) float64` — ι(v), returning 1.0 for structural matches, log₂(1/|δ|) for precision matches, 0 for untested
- `InputEntropy(significantFigures int) float64` — 3.32n bits
- `AxiomEntropy(a Anchor) float64` — returns assigned entropy for Tier 0

Tests against `minimal.json` known values and QBP v3.2 hand-calculated values from the theory doc §5.2.

**Blocked by:** #2

**Acceptance:** All entropy values match theory doc to 3 decimal places.
EOF
)"

# ---------- 5 ----------
create "Chain fidelity — compute/fidelity.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definition 9:

- `StepFidelity(stepType string) float64` — lookup from §4.4 fidelity table, including domain boundary row
- `ChainFidelity(c Chain, anchors map[string]Anchor) float64` — multiplicative product
- `ClassifyFidelityRegime(mu float64) string` — "laminar" (≥0.999), "low_sediment" (≥0.90), "moderate" (≥0.70), "heavy" (<0.70)

**Note:** The Python engine used string pattern-matching on weakest_link_id to assign fidelity. The Go implementation should use the Chain's explicit `Fidelity` field when present, falling back to computation from step types when not. The goal is to eliminate the heuristic lookup table.

**Blocked by:** #2

**Acceptance:** Sediment partition on QBP v3.2 matches Python engine output.
EOF
)"

# ---------- 6 ----------
create "N-ary mutual information with cap — compute/mutual_info.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definitions 10 and 10a:

- `PairwiseMI(predA, predB, sigmaA, sigmaB float64) float64` — Gaussian pairwise formula with ε regularisation
- `NaryMI(predictions []Prediction, sigmas []float64) float64` — multivariate formula
- `CappedMI(mi float64, chainCapacities []float64) float64` — min(MI, min(capacities))
- `StructuralMI(arity int, minCapacity float64) float64` — for structural confluences: min(arity, minCapacity)

Test cases:
- Perfect agreement (capped, not infinite)
- Large disagreement (small MI)
- Structural confluence (integer bits)
- 3-way confluence (verify > sum of pairwise)

**Blocked by:** #2

**Acceptance:** No infinite values. Cap correctly applied. 3-way > pairwise sum for test case.
EOF
)"

# ---------- 7 ----------
create "Compression ratio and velocity — compute/compression.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definitions 13 and 14:

- `GrossCompression(inv Inventory) float64` — ρ_gross
- `NetCompression(inv Inventory, inputEntropy map[string]float64) (rho float64, detail NetCompressionDetail)` — ρ_net with fractional input cost allocation
- `CompressionVelocity(prev, curr VersionSnapshot) float64` — Δρ/Δn

`NetCompressionDetail` includes per-anchor breakdown: gross, input cost, net, inputs used. This replaces the Python `compute_net_bits` function.

**Blocked by:** #4

**Acceptance:** ρ_net for QBP v3.2 = 0.765 ± 0.001. Velocity matches Python engine.
EOF
)"

# ---------- 8 ----------
create "Axiom entropy sensitivity — compute/sensitivity.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p1" "$(cat <<'EOF'
Implement Definition 15 and §4.6:

- `SensitivityBracket(inv Inventory, inputEntropy map[string]float64) (halfH, baseH, doubleH float64)` — three-point ρ_net report
- `SensitivityRatio(halfH, doubleH float64) float64` — doubleH/halfH, >0.5 = robust

**Blocked by:** #7

**Acceptance:** Bracket reported for QBP v3.2. Sensitivity ratio computed.
EOF
)"

# ---------- 9 ----------
create "Weighted gap and eddy proximity — compute/gap.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definitions 16 and 17:

- `StepDifficulty(stepCategory string) float64` — lookup from difficulty table
- `WeightedGap(inputID string, inv Inventory) (gap float64, nearestProven string)` — BFS with difficulty-weighted edges
- `EddyProximity(inputID string, inv Inventory) float64` — η/g_w
- `RankEddies(inv Inventory) []EddyRanking` — sorted by weighted proximity

Replaces the Python `compute_eddy_gaps` with the improved weighted metric.

**Blocked by:** #2, #4

**Acceptance:** f(0) remains top eddy. INST-Jc has proximity ≈ 0 (irreducible). Weighted ranking differs from unweighted for at least one input.
EOF
)"

# ---------- 10 ----------
create "Bridge centrality — compute/bridge.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p1" "$(cat <<'EOF'
Implement Method 5:

- `ClassifyDomain(id string) string` — domain classification (port from Python, but make configurable)
- `BridgeCentrality(inv Inventory, excludeAxioms bool) []BridgeNode` — sorted by domain count
- `BridgeNode` struct: ID, Domains []string, DomainCount int

**Blocked by:** #2

**Acceptance:** Matches Python engine output for QBP v3.2 (two co-equal hubs at 4 domains).
EOF
)"

# ---------- 11 ----------
create "Sediment partition detection — compute/sediment.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p1" "$(cat <<'EOF'
Implement Method 3:

- `DetectSedimentPartitions(inv Inventory) SedimentReport` — partition chains by fidelity regime, detect domain correlation
- `SedimentReport` struct: partitions (laminar/low/moderate/heavy), domain composition per partition, clean-only domains, dirty-only domains, sharp partition flag

**Blocked by:** #5

**Acceptance:** REBCO identified as dirty-only domain. Sharp partition detected.
EOF
)"

# ---------- 12 ----------
create "Confluence depth (arity-weighted) — compute/confluence_depth.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p1" "$(cat <<'EOF'
Implement §4.7 confluence depth:

- `AnchorConfluenceDepth(inv Inventory) map[string]int` — per-anchor, weighted by arity (len(paths)-1 per confluence)
- `ChainConfluenceDepth(inv Inventory) map[string]int` — per-chain

Replaces binary confluence count with arity-weighted measure.

**Blocked by:** #2

**Acceptance:** Anchors downstream of 3-way confluences have higher depth than those downstream of 2-way.
EOF
)"

# ---------- 13 ----------
create "Incoherence localisation — compute/localise.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p1" "$(cat <<'EOF'
Implement Method 6:

- `LocaliseIncoherence(anchorID string, inv Inventory) LocalisationResult` — walk backwards, use confluences as checkpoints
- `LocalisationResult` struct: error segment (start, end anchor IDs), last coherent confluence, weakest link in segment

**Blocked by:** #2

**Acceptance:** For FLAG-J in QBP v3.2, localises error to [U_dd → J] segment with t_pd as weakest link.
EOF
)"

# ---------- 14 ----------
create "Programme merge — compute/merge.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement §5:

- `MergeProgrammes(a, b Inventory) (merged Inventory, report MergeReport)` — Theorems 2 and 3
- `MergeReport` struct: shared anchors, bridge edges created, deficit classification (theoretical vs engineering), lossless flag
- Merge rules: tier = min, status = consensus, entropy = min, bridge fidelity from domain boundaries

**Blocked by:** #2, #4, #5

**Acceptance:** Merging QBP v3.2 + QBP-Q v0.1 produces correct zero-theoretical-deficit classification. Shared Tier 1 anchors have bridge fidelity 1.0.
EOF
)"

# ---------- 15 ----------
create "JSON store — store/json.go" "v0.1.0 — Crawl Complete" "type:store,phase:crawl,priority:p0" "$(cat <<'EOF'
- `LoadInventory(path string) (Inventory, error)` — deserialise from JSON file
- `SaveInventory(inv Inventory, path string) error` — serialise to JSON file
- `LoadMultiple(paths []string) ([]Inventory, error)` — for merge workflows

Backward-compatible with existing v3.2 JSON format. Handles v0.1 format (path_a/path_b) by converting to N-ary paths on load.

**Blocked by:** #2

**Acceptance:** Round-trip all three test fixtures without data loss.
EOF
)"

# ---------- 16 ----------
create "CLI tool — cmd/cth/main.go" "v0.1.0 — Crawl Complete" "type:infra,phase:crawl,priority:p0" "$(cat <<'EOF'
CLI entrypoint:

```
cth analyse <inventory.json>           # Full analysis report
cth merge <inv_a.json> <inv_b.json>    # Merge two programmes
cth health <inventory.json>            # Dashboard only
cth compare <inv_old.json> <inv_new.json>  # Compression velocity
cth fork <inventory.json>              # Per-branch health comparison
cth check-branch <inventory.json>      # Branch consistency check
```

Output: markdown report to stdout, or file with `-o` flag.

**Blocked by:** #4, #5, #6, #7, #8, #9, #10, #11, #12, #13, #14, #15, #24, #25

**Acceptance:** `cth analyse testdata/qbp_v3_2.json` produces output matching Python engine dashboard values.
EOF
)"

# ---------- 17 ----------
create "Health dashboard and report — report/ package" "v0.1.0 — Crawl Complete" "type:doc,phase:crawl,priority:p1" "$(cat <<'EOF'
- `Dashboard(inv Inventory, analysis FullAnalysis) string` — compact text dashboard
- `MarkdownReport(inv Inventory, analysis FullAnalysis) string` — full analysis report
- `RiverMap(inv Inventory) string` — narrative river description

Dashboard must include: anchor count, tier breakdown, coherence ratio, ρ_net with sensitivity bracket, velocity, top bridge, sediment partition, highest-value eddy.

**Blocked by:** #7, #8

**Acceptance:** Dashboard output matches refined dashboard from Python engine.
EOF
)"

# ---------- 18 ----------
create "MuninnDB store — store/muninn.go" "v0.2.0 — Walk Complete" "type:store,phase:walk,priority:p0" "$(cat <<'EOF'
Implement MuninnDB backend using Go SDK:

- `NewMuninnStore(baseURL, token, programme string) *MuninnStore`
- `SyncInventory(inv Inventory) error` — write all anchors/chains/confluences as engrams
- `ActivateContext(context []string, maxResults int) ([]Anchor, error)` — context-aware anchor retrieval
- `RegisterTrigger(pattern []string, callback func(Engram)) error` — semantic triggers
- Vault naming: `cth:{programme}`
- Engram concept patterns per §2.1 mapping table

Hebbian co-activation: when `ActivateContext` returns anchors together, MuninnDB strengthens their association automatically.

Ebbinghaus decay: anchors not re-tested decay in activation priority over time.

**Blocked by:** #2, #15

**Acceptance:** Write QBP v3.2 inventory to MuninnDB. Activate with context "REBCO exchange coupling" and verify FLAG-J surfaces. Verify decay: anchor not activated for 30 days has lower priority than recently activated anchor.
EOF
)"

# ---------- 19 ----------
create "NATS event integration" "v0.2.0 — Walk Complete" "type:store,phase:walk,priority:p1" "$(cat <<'EOF'
- Publish anchor status changes on `cth.{programme}.anchor.{id}.status`
- Publish health snapshots on `cth.{programme}.health`
- Subscribe pattern for BMA/Contextus consumers

**Blocked by:** #18

**Acceptance:** Anchor status change triggers NATS message received by test subscriber.
EOF
)"

# ---------- 20 ----------
create "Ab initio preference scoring — compute/ab_initio.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p2" "$(cat <<'EOF'
Implement R7 from Python engine:

- `AbInitioScore(inv Inventory) []AbInitioResult` — for multi-path targets, score = fidelity / (1 + input_count)
- Prefer lower-deficit path only when fidelities are comparable

**Blocked by:** #5

**Acceptance:** Matches Python engine output.
EOF
)"

# ---------- 21 ----------
create "Theory document — companion analysis report" "v0.1.0 — Crawl Complete" "type:doc,phase:crawl,priority:p1" "$(cat <<'EOF'
Extract §5–§11 from v0.1 theory doc into a standalone companion document: `QBP-CTH-Analysis-Report-v3_2.md`. This is the worked example referenced as [6] in the v0.2 theory.

**Acceptance:** Theory v0.2 references [6] correctly. Companion doc contains all QBP-specific analysis.
EOF
)"

# ---------- 22 ----------
create "CI and test infrastructure" "v0.1.0 — Crawl Complete" "type:infra,phase:crawl,priority:p0" "$(cat <<'EOF'
- GitHub Actions: `go test ./...` on push and PR
- GitHub Actions: `golangci-lint` on push and PR
- Test coverage reporting
- `go vet` in CI

**Blocked by:** #1

**Acceptance:** Green CI on an empty repo with placeholder tests.
EOF
)"

# ---------- 23 ----------
create "Fork model — model/fork.go" "v0.1.0 — Crawl Complete" "type:model,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement the fork data types from Theory v0.2 §2.8–§2.9:

- `ForkPoint` struct with: ID, BranchNodeID, Question, SharedPrefix ([]string), Branches ([]Branch)
- `Branch` struct with: ID, Name, Hypothesis, Burden (enum: Minimal/Extended), Anchors, Chains, Confluences, Inputs, Predictions
- `BranchObservation` struct with: AnchorID, Interpretations ([]BranchInterpretation)
- `BranchInterpretation` struct with: BranchID, Interpretation (string), Status, PredictionChain
- `BurdenType` enum: `Minimal`, `Extended`
- Validation: at least one branch must be `Minimal`; at least 2 branches required

Extend `Inventory` to include optional `ForkPoints []ForkPoint` field.

**Blocked by:** #2

**Acceptance:** Round-trip a forked inventory JSON (dark matter fork fixture) through serialise/deserialise with deep equality.
EOF
)"

# ---------- 24 ----------
create "Per-branch health metrics — compute/fork_health.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definition 20:

- `BranchHealth(inv Inventory, fork ForkPoint, branchID string) BranchHealthResult` — compute ρ_net, deficit, coherence ratio for a single branch using shared prefix + branch-specific anchors
- `CompareBranches(inv Inventory, fork ForkPoint) ForkComparison` — side-by-side comparison of all branches
- `BranchHealthResult` struct: ρ_net, ρ_net_sensitivity_bracket, information_deficit, theoretical_deficit, engineering_deficit, coherence_ratio, anchor_count, confluence_count
- `ForkComparison` struct: per-branch results, which branch has lower deficit, which has higher ρ_net

The shared prefix contributes identically to all branches. Branch-specific anchors contribute only to their own branch.

**Blocked by:** #2, #4, #7, #23

**Acceptance:** For the dark matter fork fixture: Branch A (no DM) has lower information deficit than Branch B (DM exists). Both share the same algebraic core metrics. ρ_net differs between branches.
EOF
)"

# ---------- 25 ----------
create "Branch consistency checker — compute/branch_check.go" "v0.1.0 — Crawl Complete" "type:compute,phase:crawl,priority:p0" "$(cat <<'EOF'
Implement Definition 21:

- `CheckBranchConsistency(anchor Anchor, branch Branch, sharedPrefix []string) []ConsistencyViolation` — verify all inputs in prediction chain are from shared prefix or this branch
- `CheckAllAnchors(inv Inventory, fork ForkPoint) map[string][]ConsistencyViolation` — batch check all anchors in all branches
- `ConsistencyViolation` struct: AnchorID, InputID, InputBranch (which branch the input belongs to), Description

This is the anti-drift mechanism. An anchor on Branch A ("no dark matter") whose prediction chain includes Ω_DM (a Branch B input) is flagged as branch-inconsistent.

Test cases:
- Clean branch (all inputs from shared prefix or own branch) — no violations
- Contaminated branch (one input from the other branch) — one violation
- Shared prefix anchor used by both branches — no violation (it's shared)

**Blocked by:** #2, #23

**Acceptance:** Detects cross-branch contamination in a test fixture with an intentionally planted violation. Zero violations on a clean fixture.
EOF
)"

# ---------- 26 ----------
create "Branch-locked MuninnDB vault — store/muninn.go extension" "v0.2.0 — Walk Complete" "type:store,phase:walk,priority:p0" "$(cat <<'EOF'
Extend MuninnDB store with branch-aware activation:

- `ActivateOnBranch(branchID string, context []string, maxResults int) ([]Anchor, error)` — only returns anchors from the shared prefix + specified branch
- Branch-specific vault tags: engrams tagged with `branch:{id}` or `shared`
- Activation filter: exclude engrams tagged with other branches
- `RegisterBranchTrigger(branchID string, pattern []string, callback func(Engram)) error` — triggers scoped to a branch

This prevents the activation pipeline from surfacing cross-branch assumptions. When a BMA agent is working on Branch A, MuninnDB only surfaces Branch A + shared prefix engrams.

**Blocked by:** #18, #23

**Acceptance:** Activate on Branch A with context "rotation curves" — surfaces α₀ = 0.846 conformal gravity, does NOT surface Ω_DM or NFW profiles. Activate on Branch B with same context — surfaces Ω_DM halo profiles, does NOT surface α₀ gravity corrections.
EOF
)"

# ---------- 27 ----------
create "Dark matter fork test fixture" "v0.1.0 — Crawl Complete" "type:test,phase:crawl,priority:p0" "$(cat <<'EOF'
Create `testdata/qbp_dm_fork.json` — a fork fixture based on the QBP dark matter fork analysis:

- Shared prefix: all Lean proofs, Hessian spectrum, eigenvalues, four materials predictions, gauge group derivation
- Branch A (no DM, minimal): PRED-no-dm-particle, α₀ = 0.846 gravity corrections, conformal gravity chains, spectral action inputs
- Branch B (DM exists, extended): Ω_DM = 0.26, NFW profile, halo mass function, DM particle mass (unknown), DM cross-section (unknown)
- Forked observations: rotation curves (two interpretations), Bullet Cluster (two interpretations), CMB peaks (two interpretations), JWST galaxies (two interpretations)
- One intentionally branch-inconsistent anchor for testing #25
- Hand-computed per-branch health metrics for regression testing

**Blocked by:** #23

**Acceptance:** Fixture parses. Known values documented. Branch A has lower deficit. Inconsistent anchor is detectable.
EOF
)"

echo "all 27 issues seeded for $REPO"
