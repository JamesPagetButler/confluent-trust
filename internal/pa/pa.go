// Package pa computes proof-assistant (PA) grades from Notary evidence
// (confluent-trust#108, inter#150 criticality grading).
//
// PA is DERIVED, never typed. A claim's grade counts only proof assistants
// whose evidence, re-derived at its pinned ref, is kernel-clean (no
// compiled-path tactic, standard axioms), whose declared fields match the
// derived ones, whose output hash reproduces, that the Notary did not veto, and
// whose attestation role is accepted via the inter#149 verify() interface (never
// by parsing keys). Two assistants count as two only if a NON-PRODUCER recorded
// that their statements correspond. Effective PA is the minimum along derivation
// edges. The grade is 0 (none), 1 (one clean), or 2 (two-or-more clean +
// corresponding) — the PA-2 gate.
//
// The package is stdlib-only apart from the attestation interface.
package pa

import (
	"slices"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/attestation"
)

// Grade is a PA level: PA0, PA1, or PA2 (the gate).
type Grade int

// PA levels: PA2 is the gate.
const (
	PA0 Grade = 0
	PA1 Grade = 1
	PA2 Grade = 2
)

// Declared is the producer's self-reported evidence. Informational only — the
// engine NEVER grades from it; it is compared against Derived to catch a label
// that hides a compiled path (the vm_compute-behind-reflexivity case).
type Declared struct {
	Mode       string   `json:"mode"`
	OutputHash string   `json:"output_hash"`
	Axioms     []string `json:"axioms"`
	ExitCode   int      `json:"exit_code"`
}

// Derived is what the engine recomputes itself at EvidenceRef — the graded
// truth: tactics scanned from the proof source, axioms from the tool's own
// output, the reproduced exit code and output hash, and whether the result is
// kernel-clean. In unit tests these are supplied by the fixture (the oracle of
// a real re-execution); the live engine populates them by actually re-running.
type Derived struct {
	OutputHash  string   `json:"output_hash"`
	TacticsUsed []string `json:"tactics_used"`
	Axioms      []string `json:"axioms"`
	ExitCode    int      `json:"exit_code"`
	KernelClean bool     `json:"kernel_clean"`
}

// Assistant is one proof-assistant attestation on a claim.
type Assistant struct {
	Declared    Declared `json:"declared"`
	Assistant   string   `json:"assistant"`
	EvidenceRef string   `json:"evidence_ref"`
	Producer    string   `json:"producer"`
	TrustCheck  string   `json:"trust_check"`
	// SourceSHA is the source sha this evidence was produced against. When it no
	// longer matches the claim's PinnedSHA the evidence is stale (notary#3): it
	// counts 0, so a changed proof cannot keep an old pass (stale-while-passing).
	SourceSHA string  `json:"source_sha"`
	Derived   Derived `json:"derived"`
}

// Correspondence records whether two assistants' statements correspond and who
// checked it. CheckedBy must differ from every Producer (C1), else the pair is
// treated as self-attested and counts as one.
type Correspondence struct {
	Basis       string `json:"basis"`
	CheckedBy   string `json:"checked_by"`
	CheckedAt   string `json:"checked_at"`
	Corresponds bool   `json:"corresponds"`
}

// Claim is a gradable claim: its assistants, the correspondence record, an
// optional corroboration annotation (never a grade), and the consumer that pins
// it (used to shape a shortfall for notary#3).
type Claim struct {
	ID             string         `json:"claim"`
	Consumer       string         `json:"consumer"`
	PinnedSHA      string         `json:"pinned_sha"`
	Assistants     []Assistant    `json:"proof_assistants"`
	Correspondence Correspondence `json:"correspondence"`
	CorroboratedBy []string       `json:"corroborated_by"`
}

// Policy pins the signature requirement. Flipping RequireSignature to true is a
// beekeeper ruling recorded on inter#149 — never a deploy side effect.
type Policy struct {
	RequireSignature bool `json:"require_signature"`
}

// Result is a claim's PA grade with the reasons.
type Result struct {
	ClaimID    string   `json:"claim_id"`
	Flags      []string `json:"flags"`
	PA         Grade    `json:"pa"`
	CleanCount int      `json:"clean_count"`
}

// flag names.
const (
	FlagDeclaredMismatch       = "declared_mismatch"
	FlagHashMismatch           = "hash_mismatch"
	FlagNotKernelClean         = "not_kernel_clean"
	FlagTrustCheckFail         = "trust_check_fail"
	FlagWrongRole              = "wrong_role"
	FlagSelfAttested           = "self_attested"
	FlagSelfAttestedCorrespond = "self_attested_correspondence"
	FlagUncorresponded         = "uncorresponded"
	FlagStale                  = "stale"
	FlagReexecFailed           = "reexec_failed"
	FlagNoEvidence             = "no_evidence"
)

// assistantCounts reports whether a contributes to PA, deriving the verdict and
// never trusting Declared or a bare TrustCheck:pass. It appends any flags.
func assistantCounts(a Assistant, pinnedSHA string, pol Policy, v attestation.Verifier, flags *[]string) bool {
	ok := true
	// Every guard below REQUIRES a good value rather than rejecting a bad one, and
	// fails closed on a missing field — a producer who omits a field must not pass
	// (confluent-trust#110 §I4: the declare-to-pass hole derive-don't-declare closes).
	//
	// Staleness (notary#3), fail closed: a pinned claim's evidence MUST carry the
	// source sha it was produced against AND it must match the pinned sha. Unknown
	// provenance (empty source_sha) can't prove freshness, so it counts 0.
	if pinnedSHA != "" && (a.SourceSHA == "" || a.SourceSHA != pinnedSHA) {
		*flags = append(*flags, a.Assistant+":"+FlagStale)
		ok = false
	}
	// AC3: kernel-cleanliness is DERIVED by the engine from the assistant's own
	// axioms and tactics (a per-assistant whitelist) — never taken from the
	// producer's kernel_clean bool, which may only further reject, never grant.
	if !derivedKernelClean(a) {
		*flags = append(*flags, a.Assistant+":"+FlagNotKernelClean)
		ok = false
	}
	// derive-don't-declare: a label that disagrees with the re-derived truth counts 0.
	if declaredMismatch(a) {
		*flags = append(*flags, a.Assistant+":"+FlagDeclaredMismatch)
		ok = false
	}
	// AC6/R2: the re-execution must have SUCCEEDED (exit 0), else nothing was proven.
	if a.Derived.ExitCode != 0 {
		*flags = append(*flags, a.Assistant+":"+FlagReexecFailed)
		ok = false
	}
	// R3: fail closed on missing evidence — an empty derived output hash means
	// nothing re-executed, so the assistant cannot count.
	if a.Derived.OutputHash == "" {
		*flags = append(*flags, a.Assistant+":"+FlagNoEvidence)
		ok = false
	}
	// AC2/R3: the output hash must reproduce. A MISSING declared hash can't
	// reproduce (fail closed); a present one must equal the derived hash.
	if a.Declared.OutputHash == "" || a.Declared.OutputHash != a.Derived.OutputHash {
		*flags = append(*flags, a.Assistant+":"+FlagHashMismatch)
		ok = false
	}
	// AC2/R4: trust_check must be exactly "pass". Any other value — including a
	// missing one or "pending" — does not count. It stays veto-only in the sense
	// that matters: a "pass" is necessary, never sufficient (the derived guards
	// above still have to hold).
	if a.TrustCheck != "pass" {
		*flags = append(*flags, a.Assistant+":"+FlagTrustCheckFail)
		ok = false
	}
	// AC6: attestation comes ONLY from verify(); role and self-attestation checks.
	res := v.Verify([]byte(a.EvidenceRef), a.Producer)
	// Self-attestation: the signature's attester must be independent of the
	// evidence producer. If verify() resolves the signer to this assistant's own
	// producer (the "claim author" of the converged format), the attestation is
	// self-attested and the assistant is rejected regardless of policy. Comparing
	// against the producer (a seat-id), not the claim/anchor id, is what makes
	// this check able to fire at all.
	if a.Producer != "" && res.Signer == a.Producer {
		*flags = append(*flags, a.Assistant+":"+FlagSelfAttested)
		ok = false
	}
	if pol.RequireSignature {
		if !res.Verified || res.Role != attestation.RoleNotary {
			*flags = append(*flags, a.Assistant+":"+FlagWrongRole)
			ok = false
		}
	} else if res.Verified && res.Role != attestation.RoleNotary {
		// AC6: even when signatures aren't required, a record verify() marks
		// verified but with role != notary is rejected. (An unverified/unsigned
		// record under require_signature:false is fine — it counts via re-execution.)
		*flags = append(*flags, a.Assistant+":"+FlagWrongRole)
		ok = false
	}
	return ok
}

// declaredMismatch is true when the producer's declared fields disagree with the
// engine-derived ones on anything that affects grading.
func declaredMismatch(a Assistant) bool {
	// A declared mode that omits a compiled-path tactic the source actually used.
	for _, tac := range a.Derived.TacticsUsed {
		if isCompiledPath(tac) && a.Declared.Mode != tac {
			return true
		}
	}
	if a.Declared.ExitCode != 0 && a.Declared.ExitCode != a.Derived.ExitCode {
		return true
	}
	return false
}

// derivedKernelClean computes AC3 cleanliness from the assistant's OWN derived
// axioms and tactics (R1) — the engine decides, it does not read the producer's
// kernel_clean bool for the grant. A compiled-path tactic or an axiom outside the
// per-assistant whitelist makes it unclean; the kernel_clean input can only
// further reject (so a producer can mark something dirty, never clean).
func derivedKernelClean(a Assistant) bool {
	if slices.ContainsFunc(a.Derived.TacticsUsed, isCompiledPath) {
		return false
	}
	if !axiomsClean(a.Assistant, a.Derived.Axioms) {
		return false
	}
	return a.Derived.KernelClean
}

// axiomsClean reports whether an assistant's derived axiom closure is acceptable.
// lean4: axioms ⊆ the standard-safe set. coq/agda: a closed context (no axioms/
// postulates). An unknown assistant fails closed.
func axiomsClean(assistant string, axioms []string) bool {
	switch assistant {
	case "lean4":
		allowed := map[string]bool{"propext": true, "Classical.choice": true, "Quot.sound": true}
		for _, ax := range axioms {
			if !allowed[ax] {
				return false
			}
		}
		return true
	case "coq", "agda":
		return len(axioms) == 0
	default:
		return false
	}
}

// isCompiledPath reports whether a tactic is a trust-shortcut (PA-0) tactic.
func isCompiledPath(tactic string) bool {
	switch tactic {
	case "native_decide", "vm_compute", "native_compute", "ofReduceBool", "Admitted", "sorry", "sorryAx":
		return true
	}
	return false
}

// GradeClaim computes a claim's own PA grade (ignoring derivation edges).
// Effective PA across edges is EffectivePA.
func GradeClaim(c Claim, pol Policy, v attestation.Verifier) Result {
	r := Result{ClaimID: c.ID}
	clean := 0
	for _, a := range c.Assistants {
		if assistantCounts(a, c.PinnedSHA, pol, v, &r.Flags) {
			clean++
		}
	}
	r.CleanCount = clean
	switch clean {
	case 0:
		r.PA = PA0
	case 1:
		r.PA = PA1
	default:
		// AC5 + C1: two-or-more clean count as PA2 only with a non-self-attested
		// correspondence; otherwise they collapse to one.
		if !c.Correspondence.Corresponds {
			r.Flags = append(r.Flags, FlagUncorresponded)
			r.PA = PA1
		} else if c.Correspondence.CheckedBy == "" || checkerIsProducer(c) {
			r.Flags = append(r.Flags, FlagSelfAttestedCorrespond)
			r.PA = PA1
		} else {
			r.PA = PA2
		}
	}
	return r
}

// checkerIsProducer reports whether the correspondence checker is one of the
// assistants' producers (C1: correspondence must be non-self-attested).
func checkerIsProducer(c Claim) bool {
	for _, a := range c.Assistants {
		if a.Producer != "" && a.Producer == c.Correspondence.CheckedBy {
			return true
		}
	}
	return false
}

// Edge is a typed edge between claims. Only derivation edges lower effective PA;
// relevance/mention edges are carried in the graph but excluded by DerivationDeps.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// EdgeDerivation is the only edge type that lowers effective PA (AC4).
const EdgeDerivation = "derivation"

// DerivationDeps builds the derivation-only adjacency that EffectivePA consumes,
// dropping relevance/mention edges so they never lower effective PA (AC4). Keeping
// the filter here (not in each caller) gives the exclusion one place to be tested.
func DerivationDeps(edges []Edge) map[string][]string {
	m := map[string][]string{}
	for _, e := range edges {
		if e.Type == EdgeDerivation {
			m[e.From] = append(m[e.From], e.To)
		}
	}
	return m
}

// EffectivePA returns the minimum PA of target and every claim reachable from it
// along DERIVATION edges (AC4). derivationDeps maps a claim id to the claim ids
// it derives from (derivation edges only — relevance/mention edges are excluded
// by the caller). grades maps every claim id to its own Grade (from GradeClaim()).
func EffectivePA(target string, grades map[string]Grade, derivationDeps map[string][]string) Grade {
	seen := map[string]bool{}
	var walk func(id string) Grade
	walk = func(id string) Grade {
		if seen[id] {
			return PA2 // already folded in; neutral for min
		}
		seen[id] = true
		min := grades[id]
		for _, dep := range derivationDeps[id] {
			if d := walk(dep); d < min {
				min = d
			}
		}
		return min
	}
	return walk(target)
}

// Reason classifies a notary-request (notary#3 v1.3). The CTH engine emits only
// ReasonPAShortfall and ReasonStaleness; ReasonArtifactTieMismatch is the silicon
// gate's (#77) and is defined here only so the shared record can represent it.
type Reason string

// The notary-request reasons.
const (
	ReasonPAShortfall         Reason = "pa_shortfall"
	ReasonStaleness           Reason = "staleness"
	ReasonArtifactTieMismatch Reason = "artifact_tie_mismatch"
)

// EmitterCTH is this engine's emitter id. Per notary#3 Q1 the engine is a library
// that RETURNS the record; the caller emits. This id is stamped for standalone
// runs; a calling emitter (#692, #77) overwrites it with its own.
const EmitterCTH = "cth-pa-engine"

// SchemaVersion is the notary-request schema version this engine emits.
const SchemaVersion = "1"

// NotaryRequest is the locked notary#3 v1.3 emit record. The engine returns it so
// a caller can append it to the JSON-lines CI artifact the filer consumes; the
// filer dedupes on DedupeKey and is the only writer of issues.
type NotaryRequest struct {
	ArtifactDigest  *string  `json:"artifact_digest"` // null except artifact_tie_mismatch
	CurrentPA       *Grade   `json:"current_pa"`      // null for artifact_tie_mismatch
	SchemaVersion   string   `json:"schema_version"`
	LedgerVersion   string   `json:"ledger_version"`
	Emitter         string   `json:"emitter"`
	EmittedAt       string   `json:"emitted_at"`
	Reason          Reason   `json:"reason"`
	ClaimID         string   `json:"claim_id"`
	SourceRef       string   `json:"source_ref"`
	SourceSHA       string   `json:"source_sha"`
	Detail          string   `json:"detail,omitempty"` // optional short human string (locked v1.3)
	PinningConsumer []string `json:"pinning_consumer"` // [] for unpinned
	RequiredPA      Grade    `json:"required_pa"`
	Stale           bool     `json:"stale"`
}

// DedupeKey is the notary#3 R1 cross-emitter idempotency key:
// (claim_id, source_sha, reason, artifact_digest). A pa_shortfall and an
// artifact_tie_mismatch on the same claim+sha yield different keys (different
// remediations), and two emitters reporting the same key collapse to one request.
func (n NotaryRequest) DedupeKey() string {
	dig := ""
	if n.ArtifactDigest != nil {
		dig = *n.ArtifactDigest
	}
	return n.ClaimID + "\x00" + n.SourceSHA + "\x00" + string(n.Reason) + "\x00" + dig
}

// RequestMeta carries the pin/version context the engine stamps onto a returned
// record (the grading inputs don't include it). PinningConsumer is a list ([] for
// an unpinned critical claim); EmittedAt is an RFC3339 timestamp; Detail is an
// optional short human string.
type RequestMeta struct {
	LedgerVersion   string
	SourceRef       string
	SourceSHA       string
	EmittedAt       string
	Detail          string
	PinningConsumer []string
}

// ResultIsStale reports whether a graded Result carries a staleness flag on any
// assistant — so a caller derives the emit's stale/reason from the grade itself
// rather than passing a separate bool that could disagree with it (§I4 non-block).
func ResultIsStale(r Result) bool {
	for _, f := range r.Flags {
		if f == FlagStale || strings.HasSuffix(f, ":"+FlagStale) {
			return true
		}
	}
	return false
}

// RequestFor returns a notary-request when a pinned/critical claim is below its
// required effective PA or its evidence is stale, else nil. Staleness is taken
// from the target claim's own grade (r), not a separate argument, so the emitted
// reason can't disagree with the grade. It emits reason=staleness when the grade
// is stale, else pa_shortfall; it never emits artifact_tie_mismatch (that is
// #77's). The engine returns the record; the caller emits it.
func RequestFor(c Claim, r Result, effectivePA, required Grade, meta RequestMeta) *NotaryRequest {
	stale := ResultIsStale(r)
	if effectivePA >= required && !stale {
		return nil
	}
	reason := ReasonPAShortfall
	if stale {
		reason = ReasonStaleness
	}
	cur := effectivePA
	pins := meta.PinningConsumer
	if pins == nil {
		pins = []string{}
	}
	return &NotaryRequest{
		SchemaVersion:   SchemaVersion,
		LedgerVersion:   meta.LedgerVersion,
		Emitter:         EmitterCTH,
		EmittedAt:       meta.EmittedAt,
		Reason:          reason,
		ClaimID:         c.ID,
		SourceRef:       meta.SourceRef,
		SourceSHA:       meta.SourceSHA,
		Detail:          meta.Detail,
		ArtifactDigest:  nil,
		RequiredPA:      required,
		CurrentPA:       &cur,
		PinningConsumer: pins,
		Stale:           stale,
	}
}
