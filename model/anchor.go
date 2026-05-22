package model

import "fmt"

// TheoremRef references a single theorem inside a proof file.
// status discriminates verified / written / not_started states; the
// not_started value is the §I4 Invariant 5 phantom-artifact rule's
// declaration of intent without artifact (theorem name must not appear
// in proof_file on disk until status transitions to written).
type TheoremRef struct {
	Name     string        `json:"name"`
	Blockers string        `json:"blockers,omitempty"`
	Status   TheoremStatus `json:"status"`
}

// LibraryRef captures library-pin information for verification reproducibility.
// SHA is the immutable ground truth; ref is a human-readable tag/branch; url
// is the source URL (in case the library is forked).
type LibraryRef struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
	URL string `json:"url,omitempty"`
}

// VerificationRecord captures the complete reproducibility context of a
// verified or partially-verified proof. Per design §5 + Invariant 2, the
// Toolchain + Libraries[*].SHA + VerifiedAt + Verifier + Result fields are
// all required when proof_state is verified or partial.
type VerificationRecord struct {
	Libraries  map[string]LibraryRef `json:"libraries"`
	Toolchain  string                `json:"toolchain"`
	VerifiedAt string                `json:"verified_at"`
	Verifier   string                `json:"verifier"`
	Result     string                `json:"result"`
}

// AdditionalVerification captures the v0.3 §E mixed-language-proof fallback:
// when a claim is verified in multiple proof languages, the primary surfaces
// at Anchor-level; additional verifications ride here.
type AdditionalVerification struct {
	Verification  *VerificationRecord `json:"verification"`
	ProofLanguage string              `json:"proof_language"`
	ProofFile     string              `json:"proof_file"`
	Theorems      []TheoremRef        `json:"theorems"`
}

// Axiom is a Tier 0 anchor: a programme's underivable assumption.
// In merged programmes an axiom may be marked Derivable=true with
// DerivedFromAxioms populated; in that case it is "stated as an axiom
// for architectural convenience" and the Validate() invariant relaxes.
type Axiom struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Statement         string   `json:"statement"`
	InheritedFrom     string   `json:"inherited_from,omitempty"`
	Notes             string   `json:"notes,omitempty"`
	DerivedFromAxioms []string `json:"derived_from_axioms,omitempty"`
	Layer             int      `json:"layer,omitempty"`
	Derivable         bool     `json:"derivable"`
}

// Validate enforces the Tier 0 derivability invariant: an axiom marked
// derivable=true must list its parent axioms.
func (a Axiom) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("axiom: empty id")
	}
	if a.Derivable && len(a.DerivedFromAxioms) == 0 {
		return fmt.Errorf("axiom %s: derivable=true requires derived_from_axioms", a.ID)
	}
	return nil
}

// DerivedPrinciple is a Tier 1 named derivation, conventionally with id DERIV-*.
// It is a high-level corollary of axioms and proofs that gets a stable name
// rather than being buried in a chain.
type DerivedPrinciple struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Statement   string   `json:"statement"`
	DerivedFrom []string `json:"derived_from"`
	Layer       int      `json:"layer,omitempty"`
}

// Anchor is a Tier 1–3 node: a proof, measurement, prediction, or observation.
// Computed quantities (Domain, ResidualEntropy, ConfirmatoryInfo) are not
// stored here; package compute produces them on demand.
//
// v0.3 fields (ProvenanceKind, ProofState, ProofLanguage, Theorems,
// Verification, TheoryCitation, TheoryDOI, TheoryURL,
// AdditionalVerifications) are additive. Legacy fields (Provenance,
// ProofFile, ProofSystem, SorryCount, LeanTheorem, LeanCompanionTheorems)
// are preserved for transitional dual-field reading per design §7.
type Anchor struct {
	PredictedValue          *float64                 `json:"predicted_value,omitempty"`
	SorryCount              *int                     `json:"sorry_count,omitempty"`
	LastTestedAt            *string                  `json:"last_tested_at,omitempty"`
	DiscrepancyPct          *float64                 `json:"discrepancy_pct,omitempty"`
	MeasuredError           *float64                 `json:"measured_error,omitempty"`
	MeasuredValue           *float64                 `json:"measured_value,omitempty"`
	Verification            *VerificationRecord      `json:"verification,omitempty"`
	LeanTheorem             string                   `json:"lean_theorem,omitempty"`
	BridgeRole              string                   `json:"bridge_role,omitempty"`
	TheoryURL               string                   `json:"theory_url,omitempty"`
	TheoryDOI               string                   `json:"theory_doi,omitempty"`
	ProofFile               string                   `json:"proof_file,omitempty"`
	InheritedAt             string                   `json:"inherited_at,omitempty"`
	PredictedUnit           string                   `json:"predicted_unit,omitempty"`
	BranchID                string                   `json:"branch_id,omitempty"`
	Notes                   string                   `json:"notes,omitempty"`
	MeasuredSource          string                   `json:"measured_source,omitempty"`
	TestableWhen            string                   `json:"testable_when,omitempty"`
	Description             string                   `json:"description"`
	InheritedFrom           string                   `json:"inherited_from,omitempty"`
	TheoryCitation          string                   `json:"theory_citation,omitempty"`
	ProofSystem             string                   `json:"proof_system,omitempty"`
	Name                    string                   `json:"name"`
	ID                      string                   `json:"id"`
	ProofLanguage           string                   `json:"proof_language,omitempty"`
	LeanCompanionTheorems   []string                 `json:"lean_companion_theorems,omitempty"`
	PredictionChain         []string                 `json:"prediction_chain"`
	AdditionalVerifications []AdditionalVerification `json:"additional_verifications,omitempty"`
	Theorems                []TheoremRef             `json:"theorems,omitempty"`
	Tier                    Tier                     `json:"tier"`
	Provenance              Provenance               `json:"provenance"`
	Status                  Status                   `json:"status"`
	ProvenanceKind          ProvenanceKind           `json:"provenance_kind,omitempty"`
	ProofState              ProofState               `json:"proof_state,omitempty"`
}

// Validate enforces anchor-level invariants.
func (a Anchor) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("anchor: empty id")
	}
	if a.Tier < TierProof || a.Tier > TierPrediction {
		return fmt.Errorf("anchor %s: tier %d out of range [1,3]", a.ID, a.Tier)
	}

	// Invariant 4 (design §6): provenance_kind != "proof" ⟹ proof-* fields absent.
	// Restored to PR #74 form; CTH #88 P-legacy maps to theory (no proof_state) by
	// default per the design-invariant-preserving rule. Decisions-file override
	// can set provenance_kind="proof" + proof_state="written" + synthesize a
	// stub verification record for legacy P-with-proof-file anchors.
	if a.ProvenanceKind != ProvenanceKindProof && a.ProvenanceKind != ProvenanceKindUnknown {
		if a.ProofState != ProofStateUnknown {
			return fmt.Errorf("anchor %s: provenance_kind %s cannot carry proof_state", a.ID, a.ProvenanceKind)
		}
		if a.ProofLanguage != "" {
			return fmt.Errorf("anchor %s: provenance_kind %s cannot carry proof_language", a.ID, a.ProvenanceKind)
		}
		if len(a.Theorems) > 0 {
			return fmt.Errorf("anchor %s: provenance_kind %s cannot carry theorems", a.ID, a.ProvenanceKind)
		}
		if a.Verification != nil {
			return fmt.Errorf("anchor %s: provenance_kind %s cannot carry verification", a.ID, a.ProvenanceKind)
		}
	}

	// Invariant 3 (design §6): proof_state == "verified" ⟹ all theorems verified.
	if a.ProofState == ProofStateVerified {
		for _, t := range a.Theorems {
			if t.Status != TheoremStatusVerified {
				return fmt.Errorf("anchor %s: proof_state verified requires all theorems verified, got %s for %s", a.ID, t.Status, t.Name)
			}
		}
	}

	// Invariant 2 (design §6): proof_state ∈ {verified, partial} ⟹ verification non-null
	// + toolchain non-empty + each library has non-empty sha.
	if a.ProofState == ProofStateVerified || a.ProofState == ProofStatePartial {
		if a.Verification == nil {
			return fmt.Errorf("anchor %s: proof_state %s requires verification record", a.ID, a.ProofState)
		}
		if a.Verification.Toolchain == "" {
			return fmt.Errorf("anchor %s: verification requires non-empty toolchain", a.ID)
		}
		for lib, ref := range a.Verification.Libraries {
			if ref.SHA == "" {
				return fmt.Errorf("anchor %s: verification.libraries[%s] requires non-empty sha", a.ID, lib)
			}
		}
	}

	return nil
}

// Input is an underived parameter — an "eddy" in the river metaphor (§4.6).
// Its entropy contribution is 3.32 * SignificantFigures bits.
type Input struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	Status             string `json:"status"`
	Notes              string `json:"notes,omitempty"`
	SignificantFigures int    `json:"significant_figures,omitempty"`
}

// Validate enforces input-level invariants.
func (i Input) Validate() error {
	if i.ID == "" {
		return fmt.Errorf("input: empty id")
	}
	if i.Type != "input" {
		return fmt.Errorf("input %s: type must be \"input\", got %q", i.ID, i.Type)
	}
	return nil
}
