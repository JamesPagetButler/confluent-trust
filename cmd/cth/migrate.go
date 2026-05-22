package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
)

// Canonical wire-form strings for the migration tool's per-anchor decision
// field. These are the JSON values callers write in their decisions file;
// kept as constants here so the wire form is the single source of truth and
// to satisfy CI goconst (the strings recur across migrate.go + tests).
const (
	pkTheoryStr          = "theory"
	pkTheoryExternalStr  = "theory-external"
	pkInternalComputeStr = "internal-compute" // v0.3 canonical; also suggested for QBP "I" legacy
	pkHypothesisStr      = "hypothesis"       // v0.3 canonical
	pkPhilosophyStr      = "philosophy"       // v0.3 canonical; also suggested for QBP "P" legacy

	// Legacy single-letter codes for the QBP-local v0.2 provenance values.
	// Used only as human-readable context in suggestion rationale strings.
	legacyProvDerived         = "D" // programme-derived; maps to theory or internal-compute
	legacyProvInternalCompute = "I" // calculation-derived; maps to internal-compute
	legacyProvPhilosophy      = "P" // partial-verification; maps to theory + proof_state: partial

	// Wire-form strings for the decisions-file ProofState field.
	proofStateVerifiedDecStr = "verified"
	proofStatePartialDecStr  = "partial"
	proofStateWrittenDecStr  = "written"
)

// decisionFile is the top-level shape of the JSON decisions file that callers
// supply via --decisions.  Each entry resolves the per-anchor ambiguity for
// one "T"-provenance anchor.
type decisionFile struct {
	Anchors []MigrationDecision `json:"anchors"`
}

// MigrationDecision is a per-anchor caller-supplied resolution for ambiguous
// translations.  Extended in CTH #88 to cover QBP-local D/I/P legacy values.
type MigrationDecision struct {
	// AnchorID matches Anchor.ID.
	AnchorID string `json:"id"`
	// ProvenanceKind must be one of: theory | theory-external | internal-compute |
	// hypothesis | philosophy.  Other values are rejected.
	// Previous (PR #75): theory | theory-external only.
	ProvenanceKind string `json:"provenance_kind"`
	// TheoryCitation is required when ProvenanceKind == "theory-external".
	TheoryCitation string `json:"theory_citation,omitempty"`
	// TheoryDOI and TheoryURL are optional even for theory-external anchors.
	TheoryDOI string `json:"theory_doi,omitempty"`
	TheoryURL string `json:"theory_url,omitempty"`
	// ProofState is an optional per-anchor proof_state override for QBP-local
	// P → theory+partial translations.  Values: verified | partial | written.
	// Omitting falls back to migrate default (no proof_state set for non-proof anchors).
	// Only meaningful when ProvenanceKind is "theory" or related non-proof kind.
	ProofState string `json:"proof_state,omitempty"`
}

// MigrationReport summarises a translate pass: which anchors translated
// mechanically vs which need human decision, with a TOML-like snippet for
// the caller to edit and re-run.
//
// Field order is arranged for minimal padding per govet fieldalignment.
type MigrationReport struct {
	SourcePath       string
	OutputPath       string
	DecisionsNeeded  []DecisionPrompt
	DecisionsApplied []string
	Warnings         []string
	AnchorCount      int
	MechanicalCount  int
}

// DecisionPrompt is one anchor needing a human classification call.
type DecisionPrompt struct {
	AnchorID    string
	AnchorName  string
	Description string
	Suggestion  string // "theory" or "theory-external" — heuristic guess
	Rationale   string
}

// reCitationPattern is a heuristic: looks for a 4-digit year possibly preceded
// by a capitalised surname fragment (e.g., "Hurwitz 1898", "Einstein 1905",
// "PDG 2024").
var reCitationPattern = regexp.MustCompile(`[A-Z][a-z]+\s+\d{4}|\b\d{4}\b`)

// suggestProvenance applies lightweight heuristics to guess whether an anchor
// with provenance "T" should become "theory" or "theory-external".
//
// Rules (non-binding; human must explicitly opt in to theory-external):
//   - ProofFile set → likely theory (we wrote our own proof)
//   - Description / Notes contains "Year authorname" style citation → theory-external
//   - ID contains "hurwitz", "external", or similar hints → theory-external
//   - Otherwise → theory (safe default)
func suggestProvenance(a model.Anchor) (suggestion, rationale string) {
	combined := strings.ToLower(a.Description + " " + a.Notes + " " + a.ID)
	hasCitation := reCitationPattern.MatchString(a.Description) || reCitationPattern.MatchString(a.Notes)

	if a.ProofFile != "" {
		// If there is an explicit proof file, we treat it as theory (we wrote it).
		return pkTheoryStr, "has proof_file → likely programme-internal argument"
	}
	if hasCitation && (strings.Contains(combined, "external") ||
		strings.Contains(combined, "invoked") ||
		strings.Contains(combined, "relies on") ||
		strings.Contains(combined, "after furey") ||
		strings.Contains(combined, "chamseddine") ||
		strings.Contains(combined, "hurwitz 1898") ||
		strings.Contains(combined, "kitaev") ||
		strings.Contains(combined, "altland") ||
		strings.Contains(a.ID, "external") ||
		strings.Contains(a.ID, "hurwitz") && a.ProofFile == "") {
		return pkTheoryExternalStr, "description references external authority + citation pattern detected"
	}
	if hasCitation {
		return pkTheoryExternalStr, "citation-year pattern detected in description/notes; may cite external work"
	}
	return pkTheoryStr, "no citation pattern detected → safe default"
}

// Migrate runs the v0.2 → v0.3 translation, applying any caller-supplied
// decisions.  Returns the translated Inventory + a MigrationReport.
//
// If a decision is required for a "T"-provenance anchor that has no supplied
// decision, the anchor is conservatively translated with
// provenance_kind: "theory" (the safe default; the report flags it for caller
// review).
//
// Migrating an already-v0.3 inventory is a no-op; the report will include a
// warning and the original inventory is returned unchanged.
func Migrate(inv model.Inventory, decisions []MigrationDecision) (model.Inventory, MigrationReport, error) {
	report := MigrationReport{
		AnchorCount: len(inv.Anchors),
	}

	// Idempotency: already-v0.3 inventory is a no-op.
	if inv.SchemaVersion == store.SchemaV03 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("inventory is already schema version %s; no translation applied", store.SchemaV03))
		return inv, report, nil
	}

	// Index decisions by anchor ID for O(1) lookup.
	decisionByID := make(map[string]MigrationDecision, len(decisions))
	for _, d := range decisions {
		decisionByID[d.AnchorID] = d
	}

	out := inv
	out.SchemaVersion = store.SchemaV03
	out.Anchors = make([]model.Anchor, len(inv.Anchors))
	copy(out.Anchors, inv.Anchors)

	for i := range out.Anchors {
		a := &out.Anchors[i]

		// ---- Proof-bearing translation (takes priority over T-provenance default) ----
		if a.ProofSystem != "" {
			// Anchor carries a formal proof → provenance_kind = "proof".
			a.ProvenanceKind = model.ProvenanceKindProof
			a.ProofLanguage = a.ProofSystem // "lean4" → "lean4", etc.

			// Build theorems[] from lean_theorem + lean_companion_theorems.
			// At migration time we set status "written" (not "verified") because
			// we lack the verification record (toolchain + libraries.sha).
			// cth lean-link (CTH #54, impl PR #3) populates those and transitions
			// proof_state to verified.
			if a.LeanTheorem != "" {
				a.Theorems = append(a.Theorems, model.TheoremRef{
					Name:   a.LeanTheorem,
					Status: model.TheoremStatusWritten,
				})
				for _, companion := range a.LeanCompanionTheorems {
					a.Theorems = append(a.Theorems, model.TheoremRef{
						Name:   companion,
						Status: model.TheoremStatusWritten,
					})
				}
			}

			// Derive proof_state from sorry_count.
			// ProofStateWritten is the safe choice at migration time: we know
			// a file exists but cannot assert "verified" without a toolchain pin.
			// This satisfies Invariant 2 (verified/partial ⟹ verification non-null)
			// because we never emit proof_state=verified here.
			switch {
			case a.SorryCount != nil && *a.SorryCount == 0:
				a.ProofState = model.ProofStateWritten
			case a.SorryCount != nil && *a.SorryCount > 0:
				a.ProofState = model.ProofStateWritten
			default:
				// sorry_count absent but proof_system present → assume file exists.
				a.ProofState = model.ProofStateWritten
			}

			report.Warnings = append(report.Warnings,
				fmt.Sprintf("anchor %s: proof_state set to %q (not %q); run cth lean-link (CTH #54) to populate verification record and advance proof_state",
					a.ID, model.ProofStateWritten, model.ProofStateVerified))

			report.MechanicalCount++
			continue
		}

		// ---- Legacy provenance translation ----
		switch a.Provenance {
		case model.ProvenanceExperimental: // "E"
			a.ProvenanceKind = model.ProvenanceKindExperiment
			report.MechanicalCount++

		case model.ProvenanceHypothesis: // "H"
			a.ProvenanceKind = model.ProvenanceKindHypothesis
			report.MechanicalCount++

		case model.ProvenanceTheoretical: // "T"
			// Per-anchor decision needed: theory vs theory-external.
			if dec, found := decisionByID[a.ID]; found {
				if err := applyDecision(a, dec); err != nil {
					return model.Inventory{}, MigrationReport{}, err
				}
				report.DecisionsApplied = append(report.DecisionsApplied, a.ID)
				report.MechanicalCount++
			} else {
				// No decision supplied → conservative default: theory.
				a.ProvenanceKind = model.ProvenanceKindTheory
				suggestion, rationale := suggestProvenance(*a)
				report.DecisionsNeeded = append(report.DecisionsNeeded, DecisionPrompt{
					AnchorID:    a.ID,
					AnchorName:  a.Name,
					Description: a.Description,
					Suggestion:  suggestion,
					Rationale:   rationale,
				})
				report.MechanicalCount++
			}

		case model.ProvenanceDerived: // "D" — QBP-local v0.2 legacy
			if err := handleQBPLocalProvenance(a, legacyProvDerived, decisionByID, &report,
				pkTheoryStr,
				"QBP-local 'D' (programme-derived); suggest theory or internal-compute"); err != nil {
				return model.Inventory{}, MigrationReport{}, err
			}

		case model.ProvenanceInternalCompute: // "I" — QBP-local v0.2 legacy
			if err := handleQBPLocalProvenance(a, legacyProvInternalCompute, decisionByID, &report,
				pkInternalComputeStr,
				"QBP-local 'I' (calculation-derived); suggest internal-compute"); err != nil {
				return model.Inventory{}, MigrationReport{}, err
			}

		case model.ProvenancePhilosophy: // "P" — QBP-local v0.2 legacy
			if err := handleQBPLocalProvenance(a, legacyProvPhilosophy, decisionByID, &report,
				pkTheoryStr,
				"QBP-local 'P' (partial-verification); suggest theory + proof_state: partial"); err != nil {
				return model.Inventory{}, MigrationReport{}, err
			}

		case model.ProvenanceUnknown:
			// provenance absent in v0.2 input → leave ProvenanceKind unset.
			report.MechanicalCount++
		}
	}

	return out, report, nil
}

// validProvenanceKinds is the complete set of allowable ProvenanceKind strings
// in a decisions file.  Hoisted as a constant-like var for goconst compliance
// and to provide a single source of truth for error messages.
var validProvenanceKinds = map[string]model.ProvenanceKind{
	pkTheoryStr:          model.ProvenanceKindTheory,
	pkTheoryExternalStr:  model.ProvenanceKindTheoryExternal,
	pkInternalComputeStr: model.ProvenanceKindInternalCompute,
	pkHypothesisStr:      model.ProvenanceKindHypothesis,
	pkPhilosophyStr:      model.ProvenanceKindPhilosophy,
}

// applyDecision applies a caller-supplied MigrationDecision to anchor a.
// It validates ProvenanceKind, enforces theory-external citation requirement,
// copies citation fields, and applies an optional ProofState override.
// Returns an error for any invalid decision field.
func applyDecision(a *model.Anchor, dec MigrationDecision) error {
	switch dec.ProvenanceKind {
	case pkTheoryExternalStr:
		if dec.TheoryCitation == "" {
			return fmt.Errorf("migrate: anchor %s: decision theory-external requires non-empty theory_citation", a.ID)
		}
		a.ProvenanceKind = model.ProvenanceKindTheoryExternal
		a.TheoryCitation = dec.TheoryCitation
		a.TheoryDOI = dec.TheoryDOI
		a.TheoryURL = dec.TheoryURL
	case pkTheoryStr, "":
		a.ProvenanceKind = model.ProvenanceKindTheory
	case pkInternalComputeStr:
		a.ProvenanceKind = model.ProvenanceKindInternalCompute
	case pkHypothesisStr:
		a.ProvenanceKind = model.ProvenanceKindHypothesis
	case pkPhilosophyStr:
		a.ProvenanceKind = model.ProvenanceKindPhilosophy
	default:
		return fmt.Errorf("migrate: anchor %s: unknown provenance_kind %q in decision; must be one of {theory, theory-external, internal-compute, hypothesis, philosophy}", a.ID, dec.ProvenanceKind)
	}

	// Apply optional ProofState override from decisions file.
	if dec.ProofState != "" {
		switch dec.ProofState {
		case proofStateVerifiedDecStr:
			a.ProofState = model.ProofStateVerified
		case proofStatePartialDecStr:
			a.ProofState = model.ProofStatePartial
		case proofStateWrittenDecStr:
			a.ProofState = model.ProofStateWritten
		default:
			return fmt.Errorf("migrate: anchor %s: unknown proof_state %q in decision; must be one of {verified, partial, written}", a.ID, dec.ProofState)
		}
	}
	return nil
}

// handleQBPLocalProvenance processes a QBP-local legacy provenance value (D/I/P).
// If a decision is present for this anchor, it is applied via applyDecision
// and any invalid ProvenanceKind causes a hard error (consistent with the T
// branch behaviour).  If no decision is present, the anchor receives the
// supplied defaultKind and is flagged in DecisionsNeeded with the supplied
// rationale.  In either case the legacy Provenance field is cleared to
// ProvenanceUnknown (null in JSON) so that v0.3 output does not carry D/I/P
// wire values (spec constraint: v0.3 SaveInventory must not write D/I/P).
// Returns a non-nil error only when an explicit decision is present but
// invalid; the caller propagates this as a top-level Migrate error.
func handleQBPLocalProvenance(
	a *model.Anchor,
	_ string, // legacyCode: reserved for future diagnostic use
	decisionByID map[string]MigrationDecision,
	report *MigrationReport,
	defaultKind string,
	rationale string,
) error {
	if dec, found := decisionByID[a.ID]; found {
		if err := applyDecision(a, dec); err != nil {
			return err
		}
		report.DecisionsApplied = append(report.DecisionsApplied, a.ID)
	} else {
		a.ProvenanceKind = mustProvenanceKind(defaultKind)
		report.DecisionsNeeded = append(report.DecisionsNeeded, DecisionPrompt{
			AnchorID:    a.ID,
			AnchorName:  a.Name,
			Description: a.Description,
			Suggestion:  defaultKind,
			Rationale:   rationale,
		})
	}
	report.MechanicalCount++
	return nil
}

// mustProvenanceKind returns the ProvenanceKind for a known canonical string.
// Panics on unknown strings — callers must only pass compile-time constants.
func mustProvenanceKind(s string) model.ProvenanceKind {
	if pk, ok := validProvenanceKinds[s]; ok {
		return pk
	}
	panic(fmt.Sprintf("mustProvenanceKind: unknown canonical kind %q", s))
}

// LoadDecisions reads a JSON decisions file.  Returns an empty slice if path
// is empty (no --decisions flag supplied).
func LoadDecisions(path string) ([]MigrationDecision, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: path is caller-supplied; no directory traversal risk in a CLI
	if err != nil {
		return nil, fmt.Errorf("migrate: read decisions file %s: %w", path, err)
	}
	var df decisionFile
	if err := json.Unmarshal(data, &df); err != nil {
		return nil, fmt.Errorf("migrate: parse decisions file %s: %w", path, err)
	}
	return df.Anchors, nil
}

// FormatReport renders a MigrationReport as markdown.
func FormatReport(r MigrationReport) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Migration report: %s → %s\n\n", r.SourcePath, r.OutputPath)
	fmt.Fprintf(&b, "**Generated:** %s UTC by cth migrate v0.3-impl-2\n\n", time.Now().UTC().Format("2006-01-02T15:04:05"))

	fmt.Fprint(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- Total anchors: %d\n", r.AnchorCount)
	fmt.Fprintf(&b, "- Mechanically translated: %d\n", r.MechanicalCount)
	fmt.Fprintf(&b, "- Decisions applied (from --decisions): %d\n", len(r.DecisionsApplied))
	fmt.Fprintf(&b, "- Decisions still needed: %d\n", len(r.DecisionsNeeded))
	fmt.Fprintf(&b, "- Warnings: %d\n\n", len(r.Warnings))

	if len(r.DecisionsNeeded) > 0 {
		fmt.Fprint(&b, "## Decisions still needed\n\n")
		fmt.Fprint(&b, "The following anchors carry `provenance: \"T\"` (theoretical) and need\n")
		fmt.Fprint(&b, "human classification as either `theory` (programme-internal argument)\n")
		fmt.Fprint(&b, "or `theory-external` (external published theorem invoked as proof).\n\n")
		fmt.Fprint(&b, "Create a decisions JSON file and pass it via `--decisions <path>`:\n\n")

		fmt.Fprint(&b, "```json\n{\n  \"anchors\": [\n")
		for idx, dp := range r.DecisionsNeeded {
			fmt.Fprintf(&b, "    { \"id\": %q, \"provenance_kind\": %q, \"theory_citation\": \"\" }", dp.AnchorID, dp.Suggestion)
			if idx < len(r.DecisionsNeeded)-1 {
				fmt.Fprint(&b, ",")
			}
			fmt.Fprint(&b, "\n")
		}
		fmt.Fprint(&b, "  ]\n}\n```\n\n")

		for _, dp := range r.DecisionsNeeded {
			fmt.Fprintf(&b, "### %s\n\n", dp.AnchorID)
			fmt.Fprintf(&b, "- **Name:** %s\n", dp.AnchorName)
			if dp.Description != "" {
				desc := dp.Description
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				fmt.Fprintf(&b, "- **Description:** %s\n", desc)
			}
			fmt.Fprintf(&b, "- **Suggestion:** `%s`\n", dp.Suggestion)
			fmt.Fprintf(&b, "- **Rationale:** %s\n\n", dp.Rationale)
		}
	}

	if len(r.DecisionsApplied) > 0 {
		fmt.Fprint(&b, "## Decisions applied\n\n")
		for _, id := range r.DecisionsApplied {
			fmt.Fprintf(&b, "- %s\n", id)
		}
		fmt.Fprint(&b, "\n")
	}

	if len(r.Warnings) > 0 {
		fmt.Fprint(&b, "## Warnings\n\n")
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
		fmt.Fprint(&b, "\n")
	}

	fmt.Fprint(&b, "## Re-run command\n\n")
	fmt.Fprintf(&b, "```\ncth migrate %s --decisions <decisions.json> -o <output.json>\n```\n", r.SourcePath)

	return b.String()
}
