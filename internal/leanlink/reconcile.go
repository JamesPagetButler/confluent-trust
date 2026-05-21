package leanlink

import (
	"fmt"
	"strings"
	"time"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// Class names the five reconciliation categories for PROOF-bearing anchors.
type Class string

// The five reconciliation classes from CTH #54 + Invariant 5 (design §6).
const (
	// ClassProven: anchor has proof_file pointing at a real file with matching
	// theorem; sorry_count is consistent.
	ClassProven Class = "proven"

	// ClassOrphan: Lean theorem exists but no PROOF-* anchor references it.
	ClassOrphan Class = "orphan"

	// ClassStaleRef: anchor's proof_file doesn't resolve OR theorem name not
	// found in the file.
	ClassStaleRef Class = "stale-ref"

	// ClassDrift: anchor's sorry_count != actual sorry count in the file.
	ClassDrift Class = "drift"

	// ClassPhantomTheorem: anchor declares theorem.status = "not_started" but
	// theorem name DOES appear in proof_file (Invariant 5 violation, design §6).
	ClassPhantomTheorem Class = "phantom-theorem"
)

// AnchorClassification records one anchor's reconciliation outcome.
// Field order is arranged to minimise padding per govet fieldalignment.
type AnchorClassification struct {
	// Detail is a human-readable explanation of the class assignment.
	Detail string
	// TheoremName is set when the class refers to a specific theorem.
	TheoremName string
	// AnchorID is the inventory anchor identifier.
	AnchorID string
	// Class is the reconciliation outcome for this anchor.
	Class Class
}

// OrphanTheorem records a Lean theorem with no anchor reference.
type OrphanTheorem struct {
	File string
	Name string
	Kind string
}

// ProposedUpdate is a single inventory mutation that --update-inventory applies.
// Field order is arranged to minimise padding per govet fieldalignment.
type ProposedUpdate struct {
	// Description is a human-readable summary of the change.
	Description string
	// AnchorID is the anchor to mutate.
	AnchorID string
	// Field names the logical field being updated.
	// One of: "sorry_count", "last_tested_at", "verification", "theorems",
	// "proof_state".
	Field string
}

// Report is the full reconciliation output produced by Reconcile.
type Report struct {
	// InventoryPath is the path of the loaded inventory.
	InventoryPath string
	// CorpusRoot is the corpus root that was walked.
	CorpusRoot string
	// Generated is an ISO 8601 timestamp.
	Generated string
	// Classifications holds per-anchor reconciliation outcomes.
	Classifications []AnchorClassification
	// Orphans holds Lean theorems with no referencing anchor.
	Orphans []OrphanTheorem
	// Updates holds proposed inventory mutations (populated in --update-inventory mode).
	Updates []ProposedUpdate
	// ParsedTheorems is the total count of TheoremDecl entries found.
	ParsedTheorems int
	// AnchorCount is the total count of proof-bearing anchors examined.
	AnchorCount int
}

// Reconcile is the pure reconciliation function. It cross-references parsed
// theorems against inventory anchors and returns a Report. The inventory and
// theorems slices are not mutated.
func Reconcile(inv model.Inventory, theorems []TheoremDecl, spec ToolchainSpec) Report {
	now := time.Now().UTC().Format(time.RFC3339)
	r := Report{
		Generated:      now,
		ParsedTheorems: len(theorems),
	}

	// Build a set of all theorem names referenced by anchors so we can
	// identify orphans.
	referencedNames := make(map[string]bool)

	// Process each anchor.
	for i := range inv.Anchors {
		a := &inv.Anchors[i]
		if !isProofBearing(a) {
			continue
		}
		r.AnchorCount++

		refs := collectTheoremRefs(a)
		for _, name := range refs {
			referencedNames[name] = true
		}

		if a.ProofFile == "" {
			r.Classifications = append(r.Classifications, AnchorClassification{
				AnchorID: a.ID,
				Class:    ClassStaleRef,
				Detail:   "anchor has no proof_file",
			})
			continue
		}

		// Check all theorems referenced by this anchor.
		fileTheorems := filterByFile(theorems, a.ProofFile)
		anyProven := false

		for _, ref := range refs {
			found := findByName(fileTheorems, ref)
			if found == nil {
				r.Classifications = append(r.Classifications, AnchorClassification{
					AnchorID:    a.ID,
					Class:       ClassStaleRef,
					TheoremName: ref,
					Detail:      fmt.Sprintf("theorem %s not found in %s", ref, a.ProofFile),
				})
				continue
			}

			// Invariant 5: not_started + theorem present in file = phantom-artifact.
			if statusOf(a, ref) == model.TheoremStatusNotStarted {
				r.Classifications = append(r.Classifications, AnchorClassification{
					AnchorID:    a.ID,
					Class:       ClassPhantomTheorem,
					TheoremName: ref,
					Detail:      fmt.Sprintf("theorem %s is status=not_started but appears in %s", ref, a.ProofFile),
				})
				continue
			}

			// Sorry drift check.
			if a.SorryCount != nil && *a.SorryCount != found.SorryCount {
				r.Classifications = append(r.Classifications, AnchorClassification{
					AnchorID:    a.ID,
					Class:       ClassDrift,
					TheoremName: ref,
					Detail: fmt.Sprintf("sorry_count %d (anchor) != %d (file)",
						*a.SorryCount, found.SorryCount),
				})
				r.Updates = append(r.Updates, ProposedUpdate{
					AnchorID:    a.ID,
					Field:       "sorry_count",
					Description: fmt.Sprintf("set sorry_count to %d (actual in file)", found.SorryCount),
				})
				continue
			}

			// Proven & wired.
			anyProven = true
			r.Classifications = append(r.Classifications, AnchorClassification{
				AnchorID:    a.ID,
				Class:       ClassProven,
				TheoremName: ref,
			})

			// Propose verification record when toolchain is available and anchor has none.
			if spec.Toolchain != "" && a.Verification == nil {
				result := "zero-sorry"
				if found.SorryCount > 0 {
					result = "partial"
				}
				r.Updates = append(r.Updates, ProposedUpdate{
					AnchorID: a.ID,
					Field:    "verification",
					Description: fmt.Sprintf("populate verification record: toolchain=%s result=%s",
						spec.Toolchain, result),
				})
			}
		}

		// Propose proof_state → verified when anchor is written + no sorries in
		// ALL referenced theorems in the file.
		if anyProven && a.ProofState == model.ProofStateWritten {
			allZeroSorry := true
			for _, ref := range refs {
				if found := findByName(fileTheorems, ref); found != nil && found.SorryCount > 0 {
					allZeroSorry = false
					break
				}
			}
			if allZeroSorry {
				r.Updates = append(r.Updates, ProposedUpdate{
					AnchorID:    a.ID,
					Field:       "proof_state",
					Description: "advance proof_state from written to verified (all sorry_count == 0)",
				})
			}
		}
	}

	// Collect orphans: theorems in the corpus not referenced by any anchor.
	for _, t := range theorems {
		if !referencedNames[t.Name] {
			r.Orphans = append(r.Orphans, OrphanTheorem{
				File: t.File,
				Name: t.Name,
				Kind: t.Kind,
			})
		}
	}

	return r
}

// Apply takes a Report (with Updates populated) and applies the proposed
// updates to a copy of the inventory. Returns the mutated inventory.
func Apply(inv model.Inventory, report Report, spec ToolchainSpec) (model.Inventory, error) {
	out := inv
	out.Anchors = make([]model.Anchor, len(inv.Anchors))
	copy(out.Anchors, inv.Anchors)

	now := time.Now().UTC().Format(time.RFC3339)

	for _, u := range report.Updates {
		idx := findAnchorIndex(out.Anchors, u.AnchorID)
		if idx < 0 {
			return model.Inventory{}, fmt.Errorf("leanlink apply: anchor %s not found", u.AnchorID)
		}
		a := &out.Anchors[idx]

		switch u.Field {
		case "sorry_count":
			// Re-derive the actual count from the classification.
			for _, cl := range report.Classifications {
				if cl.AnchorID == u.AnchorID && cl.Class == ClassDrift && cl.TheoremName != "" {
					// Parse from the detail string: "sorry_count N (anchor) != M (file)"
					var anchorCount, fileCount int
					_, err := fmt.Sscanf(cl.Detail, "sorry_count %d (anchor) != %d (file)", &anchorCount, &fileCount)
					if err == nil {
						a.SorryCount = &fileCount
					}
				}
			}
			a.LastTestedAt = &now

		case "last_tested_at":
			a.LastTestedAt = &now

		case "verification":
			if spec.Toolchain == "" {
				continue
			}
			result := "zero-sorry"
			if a.SorryCount != nil && *a.SorryCount > 0 {
				result = "partial"
			}
			a.Verification = &model.VerificationRecord{
				Toolchain:  spec.Toolchain,
				Libraries:  spec.Libraries,
				VerifiedAt: now,
				Verifier:   "cth-lean-link",
				Result:     result,
			}

		case "proof_state":
			a.ProofState = model.ProofStateVerified
			// Invariant 3: mark all theorems verified when advancing proof_state.
			for j := range a.Theorems {
				a.Theorems[j].Status = model.TheoremStatusVerified
			}
			// Invariant 2: if we're advancing to verified we also need a
			// verification record with toolchain. Only create it if toolchain known.
			if spec.Toolchain != "" && a.Verification == nil {
				a.Verification = &model.VerificationRecord{
					Toolchain:  spec.Toolchain,
					Libraries:  spec.Libraries,
					VerifiedAt: now,
					Verifier:   "cth-lean-link",
					Result:     "zero-sorry",
				}
			}

		case "theorems":
			// Future use; no-op at v0.1.
		}
	}

	return out, nil
}

// FormatReport renders a Report as a markdown string.
func FormatReport(r Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# lean-link reconciliation report\n\n")
	fmt.Fprintf(&b, "**Inventory:** %s\n\n", r.InventoryPath)
	fmt.Fprintf(&b, "**Corpus root:** %s\n\n", r.CorpusRoot)
	fmt.Fprintf(&b, "**Generated:** %s\n\n", r.Generated)

	// Summary counts.
	proven, orphan, staleRef, drift, phantom := classCount(r)
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- Proof-bearing anchors examined: %d\n", r.AnchorCount)
	fmt.Fprintf(&b, "- Theorems found in corpus: %d\n", r.ParsedTheorems)
	fmt.Fprintf(&b, "- Proven & wired: %d\n", proven)
	fmt.Fprintf(&b, "- Orphan theorems: %d\n", len(r.Orphans))
	fmt.Fprintf(&b, "- Stale references: %d\n", staleRef)
	fmt.Fprintf(&b, "- Sorry-count drift: %d\n", drift)
	fmt.Fprintf(&b, "- Phantom-theorem (Invariant 5 violations): %d\n", phantom)
	fmt.Fprintf(&b, "- Proposed updates: %d\n\n", len(r.Updates))
	_ = orphan // orphan count is len(r.Orphans) — already printed above

	// Per-class sections.
	if proven > 0 {
		fmt.Fprint(&b, "## Proven & wired\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassProven {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if len(r.Orphans) > 0 {
		fmt.Fprint(&b, "## Orphan theorems (no anchor reference)\n\n")
		fmt.Fprint(&b, "| File | Theorem | Kind |\n|---|---|---|\n")
		for _, o := range r.Orphans {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", o.File, o.Name, o.Kind)
		}
		fmt.Fprint(&b, "\n")
	}

	if staleRef > 0 {
		fmt.Fprint(&b, "## Stale references\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassStaleRef {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if drift > 0 {
		fmt.Fprint(&b, "## Sorry-count drift\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassDrift {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if phantom > 0 {
		fmt.Fprint(&b, "## Phantom-theorem violations (Invariant 5)\n\n")
		fmt.Fprint(&b, "Anchors declare `status: not_started` but the theorem name already appears in `proof_file`.\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassPhantomTheorem {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if len(r.Updates) > 0 {
		fmt.Fprint(&b, "## Proposed updates\n\n")
		fmt.Fprint(&b, "| Anchor ID | Field | Description |\n|---|---|---|\n")
		for _, u := range r.Updates {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", u.AnchorID, u.Field, u.Description)
		}
		fmt.Fprint(&b, "\n")
		fmt.Fprint(&b, "_Run `cth lean-link ... --update-inventory` to apply these updates._\n")
	}

	return b.String()
}

// ---- helpers ----

// isProofBearing returns true when an anchor carries proof obligations that
// lean-link should reconcile.
func isProofBearing(a *model.Anchor) bool {
	if a.ProvenanceKind == model.ProvenanceKindProof {
		return true
	}
	if a.ProofFile != "" {
		return true
	}
	if a.ProofSystem != "" {
		return true
	}
	return false
}

// collectTheoremRefs gathers all theorem names referenced by an anchor via
// the v0.3 Theorems[] field, the legacy LeanTheorem field, and the legacy
// LeanCompanionTheorems field.
func collectTheoremRefs(a *model.Anchor) []string {
	seen := make(map[string]bool)
	var refs []string

	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			refs = append(refs, name)
		}
	}

	for _, t := range a.Theorems {
		add(t.Name)
	}
	add(a.LeanTheorem)
	for _, c := range a.LeanCompanionTheorems {
		add(c)
	}
	return refs
}

// filterByFile returns the subset of decls whose File field matches the
// anchor's proof_file. The comparison is done as a path suffix match so
// that "Foundations/Hurwitz.lean" matches both absolute and corpus-relative
// forms.
func filterByFile(decls []TheoremDecl, proofFile string) []TheoremDecl {
	var out []TheoremDecl
	for _, d := range decls {
		if d.File == proofFile || strings.HasSuffix(d.File, proofFile) || strings.HasSuffix(proofFile, d.File) {
			out = append(out, d)
		}
	}
	return out
}

// findByName returns the first TheoremDecl with the given name, or nil.
func findByName(decls []TheoremDecl, name string) *TheoremDecl {
	for i := range decls {
		if decls[i].Name == name {
			return &decls[i]
		}
	}
	return nil
}

// statusOf returns the TheoremStatus for a specific theorem name in an anchor.
// Falls back to TheoremStatusUnknown when the theorem name is only in the
// legacy LeanTheorem / LeanCompanionTheorems fields (no status recorded there).
func statusOf(a *model.Anchor, name string) model.TheoremStatus {
	for _, t := range a.Theorems {
		if t.Name == name {
			return t.Status
		}
	}
	return model.TheoremStatusUnknown
}

// findAnchorIndex returns the index of the anchor with the given ID, or -1.
func findAnchorIndex(anchors []model.Anchor, id string) int {
	for i := range anchors {
		if anchors[i].ID == id {
			return i
		}
	}
	return -1
}

// classCount returns per-class counts for the five classes.
func classCount(r Report) (proven, orphan, staleRef, drift, phantom int) {
	for _, cl := range r.Classifications {
		switch cl.Class {
		case ClassProven:
			proven++
		case ClassOrphan:
			orphan++
		case ClassStaleRef:
			staleRef++
		case ClassDrift:
			drift++
		case ClassPhantomTheorem:
			phantom++
		}
	}
	return proven, orphan, staleRef, drift, phantom
}
