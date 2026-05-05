# QBP-CTH Analysis Report v3.2

**Companion to:** `Confluent-Trust-Hypergraph-Theory-v0_2.md`, reference [6].

This is the QBP-specific worked example referenced as `[6]` from the v0.2 theory doc. It was originally §§5–11 of the v0.1 theory paper (`Confluent-Trust-Hypergraph-Theory.md` in the parent CTH/Archive/) and has been extracted here so v0.2 can stand independently as the formal framework while still pointing at concrete numbers from the QBP programme at version 3.2.

The QBP v3.2 inventory itself is **not** in this repository's `testdata/` directory — it predates the v0.2 schema (lowercase provenance values, missing required fields, null `chain_id` placeholders, `DerivedPrinciple` ids without the `DERIV-` prefix). A follow-up issue will port that inventory to v0.2 schema and seed it as `testdata/qbp_v3_2.json`. Until then, the numbers in this companion are the ground truth recorded by the v2 Python engine (`~/Documents/CTH/Archive/cth_engine_v2.py`).

---

## 5. Worked Example: QBP Programme (v3.2)

We apply the framework to the QBP programme as documented in the confluent trust inventory v3.2 [6], which extends v2.1 with 21 new anchors covering topological materials (Bi₂Se₃, MATBG, α-RuCl₃) and their Lean-verified algebraic foundations.

### 5.1 Programme Structure

| Category | v2.1 | v3.2 | Change |
|----------|------|------|--------|
| Anchors | 38 | 59 | +21 |
| Derivation chains | 12 | 21 | +9 |
| Confluence points | 3 | 8 | +5 |
| Tier 1 (proofs) | 10 | 24 | +14 |
| Tier 2 (measurements/obs) | 14 | 29 | +15 |
| Tier 3 (predictions) | 3 | 6 | +3 |
| Coherent | 19 | 38 | +19 |
| Marginal | 5 | 7 | +2 |
| Incoherent | 5 | 5 | 0 |
| Untested predictions | 9 | 9 | 0 |
| Irreducible inputs | 5 | 5 | 0 |

The growth pattern is information-theoretically significant: 14 new Tier 1 proofs (zero conditional entropy), 5 new confluence points (error-detecting capacity), and zero new irreducible inputs. The programme's information deficit $\Delta(\mathcal{G})$ is unchanged while its verification coverage has more than doubled.

### 5.2 Entropy Computation

**Axioms** (unchanged):
- AXIOM-1 (information preserved): $\eta \approx 2.0$ bits
- AXIOM-2 (encoding maximal): $\eta \approx 2.0$ bits

**New Tier 1 proofs (all $\eta = 0$ conditional on axioms):**

The materials extension adds 14 proofs with zero conditional entropy. These include the quaternion closure table (Q1-Q2), SU(2) Lie algebra (Q3-Q4), Kramers theorem (Q5-Q7), Hurwitz norm multiplicativity in ℍ (Q8-Q9), Z₂ double cover (Q10), honeycomb Z₃ cyclic symmetry (G1-G3), (C₂zT)² = +1 fragile topology (G6-G7), helicity obstruction (G8-G9), plaquette Z₂ gauge (K1-K3), Clifford-Majorana algebra (K4-K5), non-abelian braid statistics (K6), Majorana central charge c = 1/2 (K7), and bond-type completeness (K8). Each is a lossless channel from axioms to theorem — perfectly laminar flow.

**Selected measurements (conditional entropy given axioms):**
- MEAS-alpha ($+0.08\%$): $\eta \approx 0.0012$ bits
- MEAS-sin2tw ($+1.5\%$): $\eta \approx 0.022$ bits
- MEAS-bi2se3-topo ($0\%$ error, structural): $\eta = 0$ (exact match)
- MEAS-matbg-fragile ($0\%$ error, helicity = 2): $\eta = 0$ (exact match)
- MEAS-rucl3-jeff ($0\%$ error, j_eff = 1/2): $\eta = 0$ (exact match)
- MEAS-kitaev-z2gauge ($0\%$ error, structural): $\eta = 0$ (exact match)
- OBS-thermal-hall-half ($0\%$ nominal, but debated): $\eta \approx 0.15$ bits (status: marginal)
- FLAG-J ($-41\%$): $\eta \approx 0.59$ bits

The materials anchors are striking: four structural predictions with zero discrepancy. These are not precision measurements — they are topological classifications (yes/no, integer-valued). In information-theoretic terms, they are single-bit channels with zero error rate. The channel capacity is exactly 1 bit, and it is fully utilised.

**Irreducible inputs** (unchanged): $\Delta(\mathcal{G}) \approx 63.0$ bits.

The critical observation: the programme grew by 55% in anchors without adding a single bit to its information deficit. All new knowledge was *derived* from existing axioms through proven chains.

### 5.3 River Map

The v3.2 river system has the same banks and main current as v2.1, but has grown a major new tributary system:

**Main current (unchanged):** Axioms → CD hierarchy → ZD locus → Hessian → Gauge groups. Entirely laminar.

**New tributary: The Materials Delta.** A major river system branches from the main current at the Hessian (PROOF-hessian), flows through the SU(2) eigenspace (PROOF-su2-lie), and then fans into a delta with four distributaries:

1. **Kramers channel → Bi₂Se₃.** SU(2) → quaternion closure → T² = −1 (Kramers) → Z₂ double cover → topological insulator classification. Entirely laminar. Meets a boulder (measured SOC of Bi, 1.25 eV) and confirms: band inversion → topological. Zero discrepancy.

2. **Honeycomb channel → MATBG.** Quaternion closure → Z₃ cyclic product → (C₂zT)² = +1 → helicity obstruction → fragile topology. Entirely laminar. Meets a boulder (measured helicity count = 2) and confirms: Wannier obstruction → fragile bands. Zero discrepancy.

3. **Kitaev channel → α-RuCl₃.** Quaternion closure → triple product e₁e₂e₃ = −e₀ → Z₂ gauge → Majorana decomposition → non-abelian anyons. Entirely laminar. Meets Kitaev's exact solution (2006) as an independent tributary — smooth confluence. The half-quantized thermal Hall (c = 1/2 = dim(ℝ)/dim(ℂ)) is a marginal boulder: some groups see it, some don't.

4. **REBCO channel (existing).** The sediment-laden tributary from v2.1, carrying the J turbulence.

**The Materials Delta is the cleanest flow in the programme.** Fourteen proven steps, four materials, four topological classes, zero discrepancy on structural predictions. This delta has no sediment: every step is either a Lean proof or a structural measurement with integer-valued outcome.

### 5.4 Confluence Analysis

The v3.2 inventory has 8 confluence points, up from 3. We classify them into two types that carry different epistemic weight:

**Type 1 — Internal confluences** (two QBP chains converge on the same target):

1. *CONFL-rebco-stability:* Tolerance factor ∩ electronic structure → REBCO. Smooth. (v2.1)
2. *CONFL-coupling-ratios:* Eigenvalue ratio ∩ RGE → sin²θ_W. Smooth. (v2.1)
3. *CONFL-f0-three-couplings:* Three couplings test f(0). Smooth with structured discrepancy. (v2.1)
4. *CONFL-born-berry-kramers* (new, deepest): Born rule (QM) ∩ Berry phase π (topology) ∩ Kramers degeneracy (materials) = ONE algebraic fact: Hurwitz norm multiplicativity in ℍ. **Three-way confluence across three physics domains.**
5. *CONFL-four-materials* (new): Bi₂Se₃ ∩ MATBG ∩ REBCO ∩ α-RuCl₃ = ONE multiplication table. **Four-way fan confluence** — one source, four independent channels, four confirmed destinations.

**Type 2 — External confluences** (QBP chain agrees with independently discovered physics):

6. *CONFL-tenfold-qbp* (new): QBP division algebra hierarchy ∩ Altland-Zirnbauer tenfold way (Schnyder et al. 2008 [23]). The same algebraic structure (ℝ, ℂ, ℍ) discovered by an independent research community for the same physical reason.
7. *CONFL-kitaev-exact* (new): QBP algebraic proofs K1-K6 ∩ Kitaev's exact solution (2006 [24]). Same algebra, same result, independent derivation.

External confluences are qualitatively stronger than internal ones. An internal confluence shows that two paths *within the framework* agree — a parity check on the derivation. An external confluence shows that the framework agrees with an *independently developed theory* — a parity check on the axioms themselves. In the river metaphor, internal confluences are tributary junctions within the same watershed; external confluences are where two river systems from different mountain ranges meet at the same valley floor.

### 5.5 The Four-Material Quaternion Convergence

The most significant structural feature of v3.2 is CONFL-four-materials. Four materials — each with a different topological/phase classification, studied by different experimental communities — derive their symmetry structure from the same 4×4 multiplication table:

| Material | Class | Algebraic fact | Physical consequence | Discrepancy |
|----------|-------|---------------|---------------------|-------------|
| Bi₂Se₃ | Z₂ robust (AII) | $e_i^2 = -1$ | Kramers T²=−1 → topological protection | 0% |
| MATBG | Z fragile (AI) | $e_1^2 \cdot e_2^2 = +1$ | (C₂zT)²=+1 → fragile, not robust | 0% |
| REBCO | d-wave nodal | SU(2) orbital overlap | Mott insulator → superconductor | 0% (structural) |
| α-RuCl₃ | Z₂ gauge | $e_1 e_2 e_3 = -e_0$ | Plaquette flux → Z₂ gauge theory | 0% |

Each row uses a *different algebraic identity* from the *same table*. The table has 16 entries encoding $\log_2(16) = 4$ bits of structural information. The four materials collectively decode all 4 bits through independent physical channels. The probability that four independent systems accidentally agree with all 16 entries of a random 4×4 table is $\sim 2^{-16} \approx 1.5 \times 10^{-5}$.

In the river metaphor, this is a highland plateau (PROOF-quat-closure) where water collects and then flows through four distinct valleys — each reaching the sea at an experimentally confirmed location. The agreement is structural (zero discrepancy), not quantitative, which means it is robust against measurement noise. The channels carry 1-bit messages (topological yes/no, gauge yes/no) at full channel capacity with zero error rate.

**Confluence coverage:** $\kappa \approx 20\%$ (counting anchors reachable by ≥2 independent paths), up from 8%. The new confluences span three physics domains (particle physics, condensed matter topology, condensed matter magnetism), meaning their noise sources are physically uncorrelated — an error in Slater screening cannot produce a false positive in the Kramers theorem proof.

### 5.5 Diagnostic Summary

1. **The main current is strong** (unchanged). The axiom-to-gauge-group chain remains the backbone.

2. **The Materials Delta is the programme's strongest new feature.** Fourteen Lean proofs, four materials, zero structural discrepancy. In information-theoretic terms, this is a zero-entropy expansion of the framework's reach — new derivations at no information cost.

3. **The REBCO tributary remains silted.** The J error is unchanged. But the Materials Delta provides a *bypass*: the structural topology of REBCO (d-wave, Mott physics) is now derived through a clean channel independent of the quantitative J chain. The programme can make topological statements about REBCO without passing through the turbulent J zone.

4. **Five new confluences provide distributed error detection.** The confluence coverage has nearly doubled, and the new confluences span multiple domains (QM, topology, materials), making them harder to satisfy by accident.

5. **The information deficit is unchanged at ~63 bits.** The five irreducible inputs remain the programme's entropy floor. The f(0) derivation (task A1/D5) at ~16.6 bits is still the single highest-value task.

6. **The thermal Hall observation is the programme's most important pending measurement.** If confirmed, OBS-thermal-hall-half becomes the most direct experimental signature of the division algebra hierarchy in condensed matter — the ratio dim(ℝ)/dim(ℂ) = 1/2 measured as a transport coefficient.

### 5.6 Eddies (Potential Recirculations)

1. **f(0):** Currently an external input. If derived (task A1/D5), creates a recirculation loop that reduces $\Delta(\mathcal{G})$ by ~16.6 bits — 26% of total information deficit. This remains the single highest-value task.
2. **θ = 2/9:** Currently irreducible. The Z₃ cyclic symmetry proven in G1-G3 (PROOF-z3-cyclic) may provide the mechanism — the Koide phase 2/9 could be related to the Z₃ rotation angle. If derived, reduces $\Delta(\mathcal{G})$ by ~6.6 bits. The v3.2 extension makes this more plausible: the same Z₃ structure that governs honeycomb symmetry and MATBG topology might constrain the lepton mass ratio.
3. **Magic angle α ≈ 1/√3:** Currently a post-hoc observation (OBS-alpha-invsqrt3, marginal). If a derivation connects the Z₃ geometric normalisation (1,1,1)/√3 to the Bistritzer-MacDonald flat band condition, this creates a new eddy. The information content is small (~3 bits) but the physical significance is high — it would mean the magic angle is algebraically forced.

---

## 6. Relationship to QBP-Native Units

The QBP programme identifies ℏ, $k$, and $c$ as conversion factors between boundary computation units and bulk observable units [6, 21]:

| Constant | Boundary Meaning | Bulk Meaning |
|----------|-----------------|-------------|
| ℏ | Information per Γ-step | Action (J·s) |
| $k$ | Information per DOF per Γ-rate | Temperature (K) |
| $c$ | Max displacement per Γ-step | Speed (m/s) |

The CTH entropy $\eta(v)$ is measured in bits, which are Shannon's unit. The QBP conversion ℏ = information per Γ-step provides a bridge: each bit of entropy in the trust graph corresponds to a physical action of $\hbar \ln 2$ in the boundary computation. This means the information deficit $\Delta(\mathcal{G})$ has a physical interpretation: it is the minimum action (in units of ℏ) required to specify all the externally supplied inputs.

This bridge is formal, not metaphorical. If the QBP axioms are correct, then the information deficit of the QBP programme is a physical quantity measured in natural units. The goal of the programme — to minimise $\Delta(\mathcal{G})$ — is equivalent to showing that physics has minimal Kolmogorov complexity given the two axioms.

---

## 7. Limitations and Open Questions

### 7.1 Axiom Entropy is Subjective

The entropy assigned to axioms (§4.2) depends on the choice of reference class: "how many alternatives were there?" For AXIOM-2, we estimated $\log_2(4) = 2$ bits. But if the reference class is "all possible algebraic structures" rather than "all normed division algebras," the entropy would be much larger. The framework does not resolve this ambiguity; it merely makes it explicit and quantifiable.

### 7.2 No Natural Viscosity

As discussed in §3.4, the epistemic river has no natural damping mechanism. Error propagation through chains is multiplicative, not dissipative. The three candidate damping mechanisms (redundant measurements, proven error bounds, confluence-based detection) must be engineered into the programme design. A CTH-aware programme should prioritise creating confluence points at high-sediment locations — i.e., should design experiments that provide independent verification of claims downstream of approximate steps.

### 7.3 The Eddy Closure Problem

When an irreducible input is derived (closing an eddy), the entropy reduction propagates through the entire downstream flow pattern. The current framework computes this propagation as a simple subtraction ($\Delta \to \Delta - \eta(v)$), but in practice, closing an eddy may change the *structure* of the hypergraph (new derivation paths become available, new confluence points emerge). A full treatment would require dynamic recomputation of the flow field after each structural change.

### 7.4 Kolmogorov Complexity is Uncomputable

The information deficit $\Delta(\mathcal{G})$ relies on entropy estimates that approximate Kolmogorov complexity. Since $K(x)$ is uncomputable in general [7, 8], our estimates are upper bounds. For practical purposes, the significant-figures approximation ($\eta \approx 3.32n$ bits for $n$ significant figures) is adequate, but it may overestimate the true irreducible complexity of inputs that have hidden structure.

---

## 8. Methods for Insight Derivation

The CTH is not merely a bookkeeping system — it is a *telescope* that reveals structure in a research programme invisible from within any single derivation chain. This section formalises seven methods for using the CTH to derive actionable insights and guide research decisions.

### 8.1 Method 1: Entropy-Guided Research Prioritisation

**Principle.** Rank all open tasks by the information deficit reduction $\delta\eta$ they would achieve if completed, normalised by estimated effort.

**Procedure.**
1. For each open task $T_k$ (from the predictions register), identify the irreducible input $v$ it would derive, or the incoherent anchor it would resolve.
2. Compute $\delta\eta_k = \eta(v)$, the entropy that would be eliminated.
3. Estimate effort $w_k$ in researcher-hours.
4. Compute priority score $P_k = \delta\eta_k / w_k$ (bits per hour).
5. Rank tasks by $P_k$.

**QBP application (v3.2):**

| Task | $\delta\eta$ (bits) | Effort (hours) | $P$ (bits/hr) |
|------|---------------------|----------------|----------------|
| A1/D5: Derive f(0) | 16.6 | ~200 | 0.083 |
| A10: Derive θ = 2/9 | 6.6 | ~100 | 0.066 |
| A9: Post-3d screening | 3.3 | ~40 | 0.083 |
| A6: Proton decay check | 0 (confirmation) | ~4 | — |
| A2/EXP-11: GW-GRB search | 0 (test, not deficit reduction) | ~80 | — |

Tasks that reduce deficit and tasks that *test* predictions occupy different roles. Deficit-reducing tasks improve the compression ratio $\rho$ (§8.7). Testing tasks improve confluence coverage $\kappa$ (§2.2). Both are valuable; the priority score captures only the first. A complete prioritisation should weight both: $P_k^* = \alpha \cdot \delta\eta_k / w_k + (1-\alpha) \cdot \delta\kappa_k / w_k$, where $\delta\kappa_k$ is the increase in confluence coverage and $\alpha$ balances compression against verification.

### 8.2 Method 2: Confluence-Driven Experiment Design

**Principle.** Design experiments that create confluence points at locations of maximum uncertainty — the turbulent zones.

**Procedure.**
1. Identify all anchors with status = incoherent or status = marginal.
2. For each, check whether a second independent derivation chain exists. If not, the anchor has *single-path vulnerability*.
3. Design an experiment or computation that provides a second path to the same target.
4. The resulting confluence point converts a single-path vulnerability into a parity check.

**QBP application.** FLAG-J (exchange coupling, −41%) has one derivation chain: U_dd → Δ → t_pd → J via perturbative superexchange. The Willow VQE proposal creates a second path: U_dd → Δ → t_pd → *ab initio many-body calculation* → J. If both paths agree on J ≈ 76 meV, the formula is correct and the upstream parameters need refinement. If the VQE gives J ≈ 130 meV (matching experiment), the formula is wrong and the parameters are right. Either outcome localises the error.

**Design criterion.** The ideal experiment for confluence creation satisfies three conditions: (a) it reaches the same target as an existing chain, (b) it shares *no* intermediate steps with the existing chain (maximising independence), and (c) its own fidelity is high (a low-fidelity second path adds little information). The VQE approach satisfies all three: it shares only the upstream parameters (U_dd, Δ, t_pd) with the perturbative chain, uses a completely different computational method, and has high intrinsic fidelity (controlled accuracy via VQE ansatz depth).

### 8.3 Method 3: Sediment Budget Tracking

**Principle.** Track the cumulative fidelity loss along each derivation chain. Flag chains whose accumulated sediment exceeds a threshold.

**Procedure.**
1. For each chain $C$, compute the cumulative fidelity $\mu(C) = \prod_i \mu(e_i)$.
2. Define a sediment budget threshold $\mu_{\min}$ (e.g., 0.90).
3. Flag chains where $\mu(C) < \mu_{\min}$ for one of two interventions:
   - **Proof formalisation:** Convert the weakest approximate step into a Lean proof ($\mu \to 1.0$), converting sediment to bedrock.
   - **Independent measurement:** Add a boulder (measurement) at the high-sediment location to constrain the flow.

**QBP application.**

| Chain | $\mu(C)$ | Action |
|-------|----------|--------|
| Axioms → Gauge groups | 1.000 | None needed (laminar) |
| Axioms → Kramers → Bi₂Se₃ | 1.000 | None needed (laminar) |
| Axioms → Z₃ → MATBG | 1.000 | None needed (laminar) |
| Axioms → Kitaev Z₂ gauge | 1.000 | None needed (laminar) |
| Axioms → f(0) → α_em | ~0.999 | Low sediment (f(0) is input, not approximation) |
| α_em → Slater → U_dd | ~0.95 | Moderate (Slater screening ~5% systematic) |
| U_dd → t_pd → J | ~0.56 | **Heavy sediment.** Priority target for proof or measurement. |

The sediment budget reveals a clean structural partition: the *algebraic* chains (axioms → proofs → structural predictions) carry zero sediment, while the *physical chemistry* chains (axioms → screening → electronic structure → material properties) carry progressively more. This partition is not obvious from the inventory alone — it emerges from the multiplicative fidelity computation. The implication is that the programme should grow preferentially along the clean channels (more materials, more topological predictions) rather than along the dirty ones (more precise REBCO parameters), unless it can first clean the dirty channels (derive t_pd, replace Slater with Hartree-Fock).

### 8.4 Method 4: Eddy Hunting

**Principle.** Scan for irreducible inputs that are close to derivation — *near-eddies* — where closing the loop would yield large entropy reduction.

**Procedure.**
1. For each irreducible input $v \in V_{\text{irreducible}}$, find the nearest proven node in the hypergraph (by chain distance).
2. Count the number of unproven steps $d(v)$ separating $v$ from the nearest proven ancestor.
3. Compute the *eddy proximity* $\pi(v) = \eta(v) / d(v)$ — entropy per unproven step.
4. Rank irreducible inputs by $\pi(v)$. High $\pi$ = large entropy, small gap = best target for eddy closure.

**QBP application.**

| Input | $\eta$ (bits) | Unproven gap $d$ | $\pi$ (bits/step) | Status |
|-------|---------------|-------------------|--------------------|---------| 
| INST-f0 | 16.6 | 1 (mechanism for $4C/\pi$ unknown) | 16.6 | **Near-eddy.** C=12 proven, 4=Tr/dim proven. One step away. |
| INST-theta | 6.6 | ~2 (Z₃ link + Fano structure) | 3.3 | Moderate eddy. |
| INST-07correction | 3.3 | ~3 (DFT-level physics) | 1.1 | Distant eddy. |
| INST-TcJ-ratio | 3.3 | ∞ (38-year open problem) | ~0 | No eddy. Irreducible. |
| INST-Jc | 33.2 | ∞ (materials science, defect-dependent) | ~0 | No eddy. Irreducible. |

The eddy proximity ranking agrees with the entropy ranking for f(0) but diverges for INST-Jc: despite its high entropy (33.2 bits), it has no realistic path to derivation, so its eddy proximity is zero. This distinguishes "large and solvable" (f(0)) from "large and irreducible" (Jc) — a distinction the raw entropy ranking misses.

### 8.5 Method 5: Cross-Domain Transfer Detection

**Principle.** When a new anchor is added, automatically check whether its prediction chain shares nodes with chains in other domains. Shared nodes are *cross-domain bridges*.

**Procedure.**
1. Partition anchors by domain (particle physics, condensed matter, cosmology, biology, etc.).
2. For each node $v$ that appears in chains from more than one domain, flag $v$ as a bridge node.
3. Rank bridge nodes by the number of domains they connect and the fidelity of the connecting chains.
4. Bridge nodes are high-priority targets for proof formalisation, since a proven bridge strengthens multiple domains simultaneously.

**QBP application.** The top bridge nodes in v3.2:

| Node | Domains connected | Chain count |
|------|-------------------|-------------|
| PROOF-quat-closure (Q1-Q2) | Particle physics, TI, MATBG, Kitaev, REBCO | 8+ |
| PROOF-kramers (Q5-Q7) | TI, Kitaev, general QM | 4 |
| PROOF-eigenratios | Particle physics, REBCO, MATBG (Mott parallel) | 3 |
| PROOF-hurwitz | Particle physics, QM (Born rule), topology (Berry phase) | 3 |

PROOF-quat-closure is the single most connected node in the programme — it is the hub of the Materials Delta. Any strengthening of this node (additional Lean verification, independent re-derivation) propagates through all four material channels simultaneously. This is the information-theoretic reason why the quaternion closure proof was worth doing: it is a *multiplier node* whose verification yield scales with the number of downstream domains.

The same pattern applies beyond QBP. In the Species Hypergraph, IGF1 is the master hub in dogs just as PROOF-quat-closure is the master hub in materials. The CTH provides a domain-agnostic method for identifying such hubs: find nodes with high betweenness centrality in the directed hypergraph, weighted by the fidelity of their connecting chains.

### 8.6 Method 6: Automated Incoherence Localisation

**Principle.** When a new measurement creates an incoherent anchor, walk backwards through the chain and use confluence points as checkpoints to localise the error.

**Procedure.**
1. Given an incoherent anchor $v$ with derivation chain $C = (e_1, \ldots, e_n)$, walk backwards from $v$ to the axioms.
2. At each step $e_i$, check: does the target $t_{e_i}$ participate in a confluence point?
3. If yes, check whether the confluence is coherent. If the confluence is still smooth, the error is *downstream* of this point. If the confluence has become incoherent, the error is *at or upstream* of this point.
4. The error is localised to the segment between the last coherent confluence and the incoherent anchor.

**QBP application.** The J chain: Axioms → eigenratios → α_em → Slater → U_dd → Δ → t_pd → J.

- CONFL-coupling-ratios (at sin²θ_W): coherent → error is downstream of this confluence.
- CONFL-rebco-stability (at tolerance factor, which shares the Slater → ionic radii segment): coherent → error is downstream of Slater, in the electronic structure branch.
- No confluence between U_dd and J → error cannot be localised further without the Willow VQE confluence.

The procedure identifies the segment [U_dd → J] as the error zone and specifically t_pd as the weakest link, consistent with the known −41% discrepancy. The Willow VQE would add a confluence at J itself, enabling the final step: is the error in the formula ($4t_{\text{eff}}^2/U$) or in t_pd?

### 8.7 Method 7: Programme Compression Ratio

**Principle.** Define a single scalar metric that captures whether the programme is generating more confirmed information than it consumes.

**Definition.** The *compression ratio* of a programme is:

$$\rho = \frac{I_{\text{confirmed}}}{H_{\text{axioms}} + \Delta(\mathcal{G})}$$

where $I_{\text{confirmed}}$ is the total information content of all confirmed predictions, $H_{\text{axioms}}$ is the entropy of the axioms, and $\Delta(\mathcal{G})$ is the information deficit.

**Computing $I_{\text{confirmed}}$.** For each confirmed measurement with fractional error $\delta$:

$$I_v = \log_2(1/|\delta|)$$

This is the number of bits the measurement verifies — the precision, in bits. For structural predictions (topological class, gauge group) with exact match: $I_v = 1$ bit (one binary question answered correctly).

**QBP application (v3.2):**

| Anchor | $|\delta|$ | $I_v$ (bits) |
|--------|-----------|---------------|
| MEAS-alpha (0.08%) | 0.0008 | 10.3 |
| MEAS-sin2tw (1.5%) | 0.015 | 6.1 |
| MEAS-koide (0.009%) | 0.00009 | 13.4 |
| MEAS-udd (1.3%) | 0.013 | 6.3 |
| MEAS-jd (2.2%) | 0.022 | 5.5 |
| MEAS-tolfac (0%, 11/11) | exact | 11.0 (=$\log_2 \binom{11}{0}^{-1}$, approximate) |
| MEAS-proton-stable (0%) | exact | 1.0 |
| MEAS-bi2se3-topo (0%) | exact | 1.0 |
| MEAS-matbg-fragile (0%) | exact | 1.0 |
| MEAS-rucl3-jeff (0%) | exact | 1.0 |
| MEAS-kitaev-z2gauge (0%) | exact | 1.0 |
| **Total** | | **~58 bits** |

**Compression ratio:**

$$\rho = \frac{58}{4 + 63} = \frac{58}{67} \approx 0.87$$

The programme is *close to* but not yet at compression parity ($\rho = 1$). It consumes 67 bits (4 axiom + 63 deficit) and produces ~58 bits of confirmed predictions. To cross $\rho = 1$, the programme needs either more confirmed predictions (the thermal Hall measurement at c = 1/2 would add ~1 bit; the GW-GRB correlation would add more) or fewer irreducible inputs (deriving f(0) would remove 16.6 bits from the denominator, jumping $\rho$ to $58/50.4 \approx 1.15$).

**Tracking $\rho$ over time:**

| Version | Anchors | $I_{\text{confirmed}}$ | $H + \Delta$ | $\rho$ |
|---------|---------|------------------------|--------------|--------|
| v2.1 | 38 | ~43 | 67 | 0.64 |
| v3.2 | 59 | ~58 | 67 | 0.87 |
| v3.2 + f(0) derived | 59 | ~58 | 50.4 | 1.15 |

The trajectory is clear: each version improves $\rho$ by adding confirmed bits without adding deficit bits. The f(0) derivation would push $\rho$ above 1.0 — the point at which the programme is a net compressor of physics.

---

## 9. Validation: Testing the CTH Framework Itself

The CTH makes claims about how to measure and manage epistemic health. These claims are themselves testable. This section identifies experiments and analyses that would validate or falsify the framework.

### 9.1 Retrospective Validation: Did the CTH Predict Known Outcomes?

**Test R1: Error localisation accuracy.** Apply Method 6 (automated incoherence localisation) retroactively to the J error discovery. The CTH should localise the error to the [U_dd → J] segment with t_pd as the weakest link. If the procedure identifies a different segment, the chain fidelity assignments are miscalibrated.

**Test R2: Priority ranking prediction.** Ask: if the CTH priority ranking (Method 1) had been available at v2.1, would it have correctly predicted which tasks were most productive? The v2.1 → v3.2 transition added 14 materials proofs (zero deficit, zero entropy) — these were high bits-per-hour tasks even though they were not deficit-reducing. This reveals a gap in Method 1's pure deficit-reduction ranking, which the combined metric $P_k^*$ (§8.1) is designed to address.

**Test R3: Cross-domain bridge prediction.** The CTH identifies PROOF-quat-closure as the top bridge node (§8.5). Retroactively, every materials extension in v3.2 flows through this node. The test: does the *next* extension also flow through it? If the programme expands to (say) topological superconductors or quantum error correction, check whether PROOF-quat-closure remains the hub. If it does, the bridge detection is predictive. If a different node becomes the hub, the method needs refinement.

### 9.2 Prospective Validation: What Should the CTH Predict Next?

**Test P1: Willow VQE as confluence creation.** The CTH predicts (§8.2) that the Willow VQE will create a confluence at FLAG-J, enabling error localisation between "formula wrong" and "parameters wrong." This is a concrete prediction about the *outcome structure* of the experiment: regardless of the numerical result, the CTH predicts that the result will disambiguate between two specific hypotheses. If the VQE returns a number but does not disambiguate (e.g., J ≈ 100 meV, midway between 76 and 130), the confluence is weaker than predicted and the method needs a finer-grained fidelity model.

**Test P2: Compression ratio trajectory.** The CTH predicts that $\rho$ will increase monotonically as the programme adds confirmed predictions without new irreducible inputs. Track $\rho$ at each version. If $\rho$ decreases (a new extension requires a new irreducible input), the decrease quantifies the information cost of the extension. If $\rho$ increases, the extension was pure compression. The prediction: for any programme that is "working" in the intuitive sense, $\rho$ should trend upward.

**Test P3: Thermal Hall as confluence strengthener.** The CTH predicts that confirmation of the half-quantized thermal Hall effect in α-RuCl₃ would convert OBS-thermal-hall-half from marginal to coherent, strengthening CONFL-kitaev-exact and adding ~1 bit to $I_{\text{confirmed}}$. The prediction is that this specific measurement has disproportionate impact on programme health — more than its 1 bit would suggest — because it strengthens a chain that connects to PROOF-majorana-charge (c = 1/2 = dim(ℝ)/dim(ℂ)), which is the most algebraically pure prediction in the Kitaev channel.

### 9.3 Framework-Level Falsification

**Test F1: Find a programme where the CTH gives wrong priorities.** Apply the CTH to a well-documented historical research programme (e.g., the development of QCD, the discovery of the Higgs boson) where the "correct" priorities are known in retrospect. If the CTH's entropy-guided ranking (Method 1) would have led researchers *away* from the breakthrough path, the method is falsified for that class of programme.

**Test F2: Compression ratio for known-wrong frameworks.** Compute $\rho$ for a framework that is *known* to be wrong (e.g., Ptolemaic astronomy with epicycles, SU(5) GUT after proton decay bounds). The CTH predicts that $\rho$ should stagnate or decrease as the framework requires more epicycles (irreducible inputs) to accommodate new data. If a known-wrong framework maintains $\rho > 1$ indefinitely, the compression ratio is not a reliable health metric.

**Test F3: Does confluence coverage correlate with reliability?** Across multiple research programmes, check whether higher $\kappa$ (confluence coverage) correlates with lower eventual retraction or correction rate. This requires a dataset of research programmes with enough structure to compute $\kappa$ — likely feasible for well-documented programmes in physics, chemistry, and biology where derivation chains are explicit in the literature. If $\kappa$ does not correlate with reliability, the CTH's emphasis on confluence is misplaced.

### 9.4 Implementation Validation

**Test I1: Go implementation produces correct entropy values.** Implement the entropy computation (§4.2) in Go against the v3.2 JSON. Verify that the computed values match the hand-calculated values in §5.2. This is a unit test, not a scientific test, but it validates the computational pipeline.

**Test I2: MuninnDB hypergraph query performance.** The CTH methods (especially Method 5: cross-domain bridge detection and Method 6: incoherence localisation) require graph traversal queries on the hypergraph. Test whether MuninnDB's query performance is adequate for real-time use in BMA: can Method 6 localise an error in < 1 second on the v3.2 graph (59 nodes, 21 chains)? Can it scale to 500 nodes and 200 chains?

**Test I3: BMA agent navigation.** Deploy a BMA agent with the CTH flow field (§3.7) and test whether it makes better research recommendations than an agent without it. "Better" is defined as: does the agent more often suggest tasks that actually reduce $\Delta(\mathcal{G})$ or increase $\kappa$ when executed? This is an A/B test between CTH-aware and CTH-unaware agents on the same programme.

### 9.5 Cross-Programme Validation

**Test C1: Apply CTH to Materia-Bio Species Hypergraph.** The Species Hypergraph has 86/86 predictions across two kingdoms (Brassica and dogs). Apply the CTH: compute entropy for each prediction, identify irreducible inputs (e.g., the "why IGF1?" question), map confluences (tree-vs-lattice as a cross-kingdom structural confluence), and compute $\rho$. If the framework produces sensible results — in particular, if it identifies IGF1's hub status as a bridge node (Method 5) without being told — it validates domain-agnosticity.

**Test C2: Apply CTH to Möbius fusion reactor programme.** The Bench-to-Grid programme has its own chain structure: axioms (topoRet ≥ 0.10) → RF waveguide topology → plasma control → HVDC integration → grid economics. Apply the CTH and check: does it correctly identify topoRet as the single largest source of uncertainty (it's the unverified central claim)? Does it correctly identify Phase 0A (RF waveguide test) as the highest-priority task (it's the cheapest test of the most uncertain claim)?

**Test C3: Apply CTH to an external programme.** Select a published, well-documented research programme (candidate: the muon g-2 anomaly analysis, which has explicit derivation chains, known uncertainties, and recent controversy about hadronic vacuum polarisation calculations). Apply the CTH. Check whether the framework correctly identifies the hadronic VP calculation as the primary source of turbulence and the lattice QCD / dispersive cross-section discrepancy as an incoherent confluence. If so, the CTH produces non-trivial insight about a programme it was not designed for.

---

## 10. Conclusion

The confluent trust hypergraph provides a rigorous information-theoretic framework for assessing the epistemic health of scientific research programmes. By assigning Shannon entropy to claims, channel fidelity to derivation steps, and mutual information to confluence points, the framework transforms qualitative judgments ("this result is well-established" or "this derivation is approximate") into computable quantities that support automated monitoring, priority-setting, and error localisation.

The hydrodynamic metaphor — axioms as riverbanks, proofs as laminar flow, errors as turbulence, confluences as river junctions, and recirculation as eddies — is not merely illustrative. Each element maps onto a formal quantity, and the metaphor reveals structural features (the absence of viscosity, the importance of eddy closure) that are not obvious from the algebra alone.

Seven methods (§8) operationalise the framework: entropy-guided prioritisation ranks tasks by bits-per-hour; confluence-driven experiment design targets parity checks at turbulent zones; sediment budget tracking flags chains whose accumulated approximation error exceeds a threshold; eddy hunting identifies near-derivable irreducible inputs; cross-domain transfer detection finds bridge nodes that connect multiple research domains; automated incoherence localisation walks backwards through chains using confluences as checkpoints; and the compression ratio $\rho$ provides a single scalar metric for whether the programme is generating more confirmed information than it consumes.

Applied to the QBP programme (v3.2), the framework identifies the f(0) derivation as the single highest-value task (26% of total information deficit), confirms the REBCO t_pd correction as the primary source of turbulence, and validates the programme's eight confluence points — including two external confluences with independently discovered physics (Altland-Zirnbauer tenfold way, Kitaev exact solution) — as functional parity checks. The four-material quaternion convergence (Bi₂Se₃, MATBG, REBCO, α-RuCl₃) demonstrates that 14 new proofs and 4 zero-entropy structural measurements can be derived from the same axioms with zero increase in information deficit.

The v2.0 refinement cycle (§11) produced a critical correction: the honest compression ratio $\rho_{\text{net}} = 0.765$, not the naively computed $\rho_{\text{gross}} = 1.112$. The programme is not yet a net compressor when input costs are properly allocated. Deriving f(0) pushes $\rho_{\text{net}}$ to 1.017 — the threshold between consuming more information than the programme produces and producing more than it consumes. This single number transforms f(0) from "the highest-priority task" to "the task that determines whether the programme works." The compression velocity $\dot{\rho}_{\text{net}} = 0.0126$ per anchor is positive, confirming the programme is on the right trajectory.

Fourteen validation tests (§9) — retrospective, prospective, framework-level, implementation, and cross-programme — provide concrete criteria for evaluating the CTH itself. The most stringent tests are F2 (compute $\rho$ for known-wrong frameworks to verify it detects pathology) and C3 (apply the CTH to an external programme like muon g-2 to verify domain-agnosticity).

The framework is domain-agnostic. Any research programme that can express its claims as anchors, its derivations as hyperedges, and its validations as measurements can be equipped with a CTH and monitored using the metrics and methods defined here. The Go implementation targeting MuninnDB [19] provides a concrete path to deployment in the BMA architecture [22].

## 11. Refinement Results (v2.0 Engine)

Applying the seven refinements (§8) to the QBP v3.2 inventory produced three corrections, two confirmations, and two structural findings.

### 11.1 Correction: ρ(net) = 0.765, Not 1.112

The v1.0 engine computed $\rho_{\text{gross}} = 1.112$, suggesting the programme was already a net compressor. The v2.0 engine introduces *net confirmed bits* — subtracting the allocated input cost from each anchor's gross contribution. The result:

| Metric | Value |
|--------|-------|
| Gross confirmed | 74.6 bits |
| Input cost (allocated) | 23.3 bits |
| Net confirmed | 51.4 bits |
| $\rho_{\text{gross}}$ | 1.112 |
| $\rho_{\text{net}}$ | 0.765 |

The difference is substantial. The Koide formula drops from 16.8 gross to 10.1 net bits (after subtracting 6.6 bits for INST-theta). The three coupling constant measurements (α_em, sin²θ_W, α_s) each pay 4.2 bits for their share of INST-f0 (16.6 bits split across 4 consumers).

The honest conclusion: **the programme is not yet a net compressor.** It is 15.7 bits short of parity. Deriving f(0) would push $\rho_{\text{net}}$ to 1.017 — barely above parity. This makes the f(0) derivation not just the highest-value task but the *threshold* task: it is the difference between a programme that consumes more information than it produces and one that doesn't.

This correction matters epistemologically. A programme that claims $\rho > 1$ is claiming to explain more than it assumes. The v1.0 metric made that claim prematurely by double-counting: it credited the programme for confirming predictions that depended on external inputs, without debiting the cost of those inputs. The v2.0 metric is honest.

### 11.2 Correction: Automated Gap Differs From Hand-Assigned

The v1.0 engine assigned f(0) a gap of 1 (one step from derivation). The v2.0 BFS computation found gap = 2: the nearest proven ancestor is PROOF-eigenratios, with two unproven steps between them (the spectral action mechanism and the heat kernel computation). This halves the eddy proximity from π = 16.6 to π = 8.3 bits/step.

The ranking is unchanged — f(0) is still the top eddy — but the magnitude is more conservative. Similarly, INST-TcJ-ratio computed to gap = 5 (not ∞ as hand-assigned), meaning it has a finite but distant path through the REBCO chain. INST-Jc and INST-07correction remain at gap = ∞, confirming they are genuinely irreducible at the current level.

### 11.3 Correction: Two Structural Bridge Hubs, Not One

The v1.0 engine identified PROOF-quat-closure as the top bridge. The v2.0 non-axiom analysis reveals *two* co-equal hubs at 4 domains each:

| Hub | Domains Connected | Role |
|-----|-------------------|------|
| PROOF-eigenratios | algebraic_foundation, general, particle_cosmo, rebco | Hub of the *particle physics / REBCO* tributary |
| PROOF-kramers | algebraic_foundation, kitaev, matbg, topological_insulator | Hub of the *materials* tributary |
| PROOF-quat-closure | algebraic_foundation, kitaev, matbg, topological_insulator | Co-hub with Kramers (tied domains) |

The programme has two parallel hub structures, one for each major tributary. This is a richer picture than "one hub" — it shows that the programme's cross-domain connectivity is distributed, not centralised. Strengthening either hub propagates to 4 domains.

### 11.4 Confirmation: Sharp Sediment Partition

The automated partition detection confirms: REBCO is the *only* dirty-only domain. All other domains (algebraic_foundation, matbg, kitaev, particle_cosmo, general) appear in the laminar partition. The implication is precise: the programme's five incoherent anchors (FLAG-J, FLAG-Tc, FLAG-xi, FLAG-Hc2, FLAG-postd-IE) are all downstream of a single dirty channel (Slater → t_pd). The programme's structural integrity depends on quarantining this channel while growing the clean ones.

### 11.5 Confirmation: Compression Velocity is Positive

$\dot{\rho}_{\text{gross}} = 0.0225$ per anchor (v2.1 → v3.2). $\dot{\rho}_{\text{net}} = 0.0126$ per anchor. Both positive, meaning each new anchor improves the compression ratio. At the net rate, the programme needs approximately 19 more anchors — all derived without new irreducible inputs — to reach $\rho_{\text{net}} = 1.0$ without deriving f(0). Alternatively, deriving f(0) alone achieves parity immediately.

### 11.6 Finding: Confluence Depth is Uniformly Shallow

The recursive confluence depth computation found depth = 1 for 13 materials anchors (they depend on PROOF-quat-closure, which is a confluence target) but depth = 0 at the chain level. This means the programme has *no* internal sub-confluences: its confluences are all top-level. For comparison, the lattice QCD path in muon g-2 had depth ≥ 2 (5 groups agreeing, window decomposition agreeing internally).

This is a growth target: the programme should seek *internal* replication of its key computations. If a second group independently verified the 42 ZD enumeration, or the Kramers theorem proof, the confluence depth would increase and the programme's robustness would improve qualitatively.

### 11.7 Finding: Ab Initio Preference Cannot Be Tested Yet

The v3.2 chain structure does not contain multi-path targets in the JSON (confluences are recorded separately from chains). The ab initio preference method requires the data model to represent alternative paths to the same target explicitly. This is a schema refinement for v4.0: add a `alternative_chains` field to confluence points that links to the specific chain IDs being compared.

---

[1] C. E. Shannon, "A Mathematical Theory of Communication," *Bell System Technical Journal*, vol. 27, no. 3, pp. 379–423, 1948.

[2] A. P. Dempster, "Upper and Lower Probabilities Induced by a Multivalued Mapping," *Annals of Mathematical Statistics*, vol. 38, no. 2, pp. 325–339, 1967.

[3] G. Shafer, *A Mathematical Theory of Evidence*. Princeton University Press, 1976.

[4] M. H. A. Newman, "On Theories with a Combinatorial Definition of 'Equivalence'," *Annals of Mathematics*, vol. 43, no. 2, pp. 223–243, 1942.

[5] G. Huet, "Confluent Reductions: Abstract Properties and Applications to Term Rewriting Systems," *Journal of the ACM*, vol. 27, no. 4, pp. 797–821, 1980.

[6] J. P. Butler, "Quaternion-Based Physics: Confluent Trust Inventory v3.2," Working document, Helpful Engineering, 2026.

[7] A. N. Kolmogorov, "Three Approaches to the Quantitative Definition of Information," *Problems of Information Transmission*, vol. 1, no. 1, pp. 1–7, 1965.

[8] G. J. Chaitin, "On the Length of Programs for Computing Finite Binary Sequences," *Journal of the ACM*, vol. 13, no. 4, pp. 547–569, 1966.

[9] E. T. Jaynes, *Probability Theory: The Logic of Science*. Cambridge University Press, 2003.

[10] J. Pearl, "Reasoning with Belief Functions: An Analysis of Compatibility," *International Journal of Approximate Reasoning*, vol. 4, no. 5–6, pp. 363–389, 1990.

[11] R. Jirousek and P. P. Shenoy, "A New Definition of Entropy of Belief Functions in the Dempster-Shafer Theory," *International Journal of Approximate Reasoning*, vol. 92, pp. 1–19, 2018.

[12] A. Church and J. B. Rosser, "Some Properties of Conversion," *Transactions of the American Mathematical Society*, vol. 39, no. 3, pp. 472–482, 1936.

[13] C. Berge, *Graphs and Hypergraphs*. North-Holland, 1973.

[14] G. Gallo, G. Longo, and S. Pallottino, "Directed Hypergraphs and Applications," *Discrete Applied Mathematics*, vol. 42, no. 2, pp. 177–201, 1993.

[15] Y. Feng, H. You, Z. Zhang, R. Ji, and Y. Gao, "Hypergraph Neural Networks," *Proceedings of the AAAI Conference on Artificial Intelligence*, vol. 33, pp. 3558–3565, 2019.

[16] G. Gao, Z. Chen, S. Li, Z. Zhao, J. Li, and Y. Gao, "Hypergraph Computation," *Engineering*, vol. 38, pp. 188–201, 2024.

[17] A. Jøsang, "A Logic for Uncertain Probabilities," *International Journal of Uncertainty, Fuzziness and Knowledge-Based Systems*, vol. 9, no. 3, pp. 279–311, 2001.

[18] G. K. Batchelor, *An Introduction to Fluid Dynamics*. Cambridge University Press, 1967.

[19] MuninnDB: Hypergraph knowledge store. Go implementation, BSL 1.1 license.

[20] A. Hurwitz, "Ueber die Composition der quadratischen Formen von beliebig vielen Variablen," *Nachrichten von der Gesellschaft der Wissenschaften zu Göttingen*, pp. 309–316, 1898.

[21] J. P. Butler, "QBP: Three Structures Hidden Inside Time," Working document, Helpful Engineering, 2026.

[22] J. P. Butler, "Biological Mind Architecture: Confluent Trust Specification v1.0," Working document, Helpful Engineering, 2026.

[23] A. P. Schnyder, S. Ryu, A. Furusaki, and A. W. W. Ludwig, "Classification of Topological Insulators and Superconductors in Three Spatial Dimensions," *Physical Review B*, vol. 78, no. 19, 195125, 2008.

[24] A. Kitaev, "Anyons in an Exactly Solved Model and Beyond," *Annals of Physics*, vol. 321, no. 1, pp. 2–111, 2006.

[25] Y. Kasahara et al., "Majorana Quantization and Half-Integer Thermal Quantum Hall Effect in a Kitaev Spin Liquid," *Nature*, vol. 559, pp. 227–231, 2018.

[26] H. Zhang et al., "Topological Insulators in Bi₂Se₃, Bi₂Te₃ and Sb₂Te₃ with a Single Dirac Cone on the Surface," *Nature Physics*, vol. 5, pp. 438–442, 2009.

[27] Y. Cao et al., "Unconventional Superconductivity in Magic-Angle Graphene Superlattices," *Nature*, vol. 556, pp. 43–50, 2018.

[28] C. L. Furey, "Standard Model Physics from an Algebra?" PhD thesis, University of Waterloo, 2015.

[29] T. Aoyama et al., "The Anomalous Magnetic Moment of the Muon in the Standard Model," *Physics Reports*, vol. 887, pp. 1–166, 2020.

[30] I. Lakatos, "Falsification and the Methodology of Scientific Research Programmes," in *Criticism and the Growth of Knowledge*, Cambridge University Press, pp. 91–196, 1970.
