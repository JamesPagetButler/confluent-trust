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
	pkTheoryStr         = "theory"
	pkTheoryExternalStr = "theory-external"
)

// decisionFile is the top-level shape of the JSON decisions file that callers
// supply via --decisions.  Each entry resolves the per-anchor ambiguity for
// one "T"-provenance anchor.
type decisionFile struct {
	Anchors []MigrationDecision `json:"anchors"`
}

// MigrationDecision is a per-anchor caller-supplied resolution for ambiguous
// translations (currently only the "T" → theory/theory-external split).
type MigrationDecision struct {
	// AnchorID matches Anchor.ID.
	AnchorID string `json:"id"`
	// ProvenanceKind must be "theory" or "theory-external"; other values are rejected.
	ProvenanceKind string `json:"provenance_kind"`
	// TheoryCitation is required when ProvenanceKind == "theory-external".
	TheoryCitation string `json:"theory_citation,omitempty"`
	// TheoryDOI and TheoryURL are optional even for theory-external anchors.
	TheoryDOI string `json:"theory_doi,omitempty"`
	TheoryURL string `json:"theory_url,omitempty"`
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
				switch dec.ProvenanceKind {
				case pkTheoryExternalStr:
					if dec.TheoryCitation == "" {
						return model.Inventory{}, MigrationReport{},
							fmt.Errorf("migrate: anchor %s: decision theory-external requires non-empty theory_citation", a.ID)
					}
					a.ProvenanceKind = model.ProvenanceKindTheoryExternal
					a.TheoryCitation = dec.TheoryCitation
					a.TheoryDOI = dec.TheoryDOI
					a.TheoryURL = dec.TheoryURL
				case pkTheoryStr, "":
					a.ProvenanceKind = model.ProvenanceKindTheory
				default:
					return model.Inventory{}, MigrationReport{},
						fmt.Errorf("migrate: anchor %s: unknown provenance_kind %q in decision; must be \"theory\" or \"theory-external\"", a.ID, dec.ProvenanceKind)
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

		case model.ProvenanceUnknown:
			// provenance absent in v0.2 input → leave ProvenanceKind unset.
			report.MechanicalCount++
		}

		// Handle QBP-local provenance extensions that LoadInventory accepted
		// via the permissive v0.2 schema.  These are encoded as the raw string
		// in the JSON and are decoded via ProvenanceKind's UnmarshalJSON if the
		// input already carried provenance_kind.  For anchors that only have the
		// legacy Provenance field these cases can't arise (the Provenance enum
		// only has T/E/H + Unknown).  Nothing to do here.
	}

	return out, report, nil
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
