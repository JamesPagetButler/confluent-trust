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

	// ClassAwaiting: a missing lean_theorem resolution on an anchor whose
	// status is "untested" (confluent-trust#130, Harlequin/Gleipnir GL-014;
	// cth-implementor steward ruling). An open-root discharge target names a
	// theorem that doesn't exist yet, by design — that's not a stale
	// reference, it's a target still awaiting its proof. Never counts toward
	// --strict's default findings; always listed by name in the report so a
	// new entry (e.g. a typo'd lean_theorem) is a visible diff, not a silent
	// one (qbp-architecture §I4 condition 1).
	ClassAwaiting Class = "awaiting"

	// ClassDischargedWhileOpen: the anchor was awaiting (status "untested",
	// lean_theorem unresolved) and WalkCorpus now resolves it to a real
	// theorem/lemma. Reported loudly rather than silently folding into
	// ClassProven — this is exactly the event Gleipnir's GL-014 tripwire
	// wants surfaced. It still requires a human/CI act to flip status to
	// verified; lean-link reports, it doesn't auto-promote.
	ClassDischargedWhileOpen Class = "discharged-while-open"

	// ClassProvenanceKindMismatch: an anchor with provenance_kind "proof"
	// resolves its lean_theorem to a declaration whose kind is not theorem
	// or lemma (def/abbrev/example/structure). This must never classify
	// ClassProven nor propose a proof_state advance — a statement is not a
	// proof of anything, however it happens to be named.
	ClassProvenanceKindMismatch Class = "provenance-kind-mismatch"

	// ClassVerificationRegression: against a supplied base inventory
	// (DetectRegressions), an anchor's status flipped from something other
	// than "untested" to "untested". A deleted or broken proof must never
	// read the same as a theorem that was never written yet — this is its
	// own loud line, never silently absorbed into ClassAwaiting
	// (qbp-architecture §I4 condition 2).
	ClassVerificationRegression Class = "verification-regression"
)

// isProofKind reports whether a parsed declaration's kind counts as an
// actual proof obligation (theorem/lemma) rather than a statement
// (def/abbrev/example/structure) that merely happens to share a name.
func isProofKind(kind string) bool {
	return kind == "theorem" || kind == "lemma"
}

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
			referencedNames[bareTheoremName(name)] = true
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
				if a.Status == model.StatusUntested {
					r.Classifications = append(r.Classifications, AnchorClassification{
						AnchorID:    a.ID,
						Class:       ClassAwaiting,
						TheoremName: ref,
						Detail:      fmt.Sprintf("theorem %s not yet written (status: untested)", ref),
					})
					continue
				}
				r.Classifications = append(r.Classifications, AnchorClassification{
					AnchorID:    a.ID,
					Class:       ClassStaleRef,
					TheoremName: ref,
					Detail:      fmt.Sprintf("theorem %s not found in %s", ref, a.ProofFile),
				})
				continue
			}

			// A "<module>::<theorem>" join key's module part is an identity
			// claim (design §3.4), not decoration: if present, it must
			// resolve to the file the bare name was actually found in, or
			// a wrong-module key would silently certify an identity that
			// was never checked (CTH #124 fix-round finding).
			if module, mismatch := joinKeyModuleMismatch(ref, found.File); mismatch {
				r.Classifications = append(r.Classifications, AnchorClassification{
					AnchorID:    a.ID,
					Class:       ClassStaleRef,
					TheoremName: ref,
					Detail: fmt.Sprintf("join-key module %q does not resolve to %s (theorem found in %s)",
						module, moduleToPath(module), found.File),
				})
				continue
			}

			// No proof credit for a non-proof declaration (confluent-trust#130
			// Ask 2; cth-implementor steward ruling, ruled stronger than
			// Harlequin's minimum-bar warn). A def/abbrev/example/structure
			// sharing a name with the discharge target is not a proof of
			// anything — this must preempt the phantom/drift/proven checks
			// below, all of which presuppose an actual proof obligation.
			if a.ProvenanceKind == model.ProvenanceKindProof && !isProofKind(found.Kind) {
				r.Classifications = append(r.Classifications, AnchorClassification{
					AnchorID:    a.ID,
					Class:       ClassProvenanceKindMismatch,
					TheoremName: ref,
					Detail: fmt.Sprintf("provenance_kind=proof but %s is a %s, not a theorem/lemma (in %s)",
						ref, found.Kind, found.File),
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

			// Proven & wired — or, if the anchor is still status:untested,
			// discharged-while-open: the theorem now exists but the anchor
			// hasn't been promoted yet. Reported as its own loud class
			// rather than silently folding into ClassProven (confluent-
			// trust#130 Ask 1; Gleipnir GL-014's tripwire). lean-link
			// reports this; it never auto-promotes a.Status itself.
			anyProven = true
			class := ClassProven
			if a.Status == model.StatusUntested {
				class = ClassDischargedWhileOpen
			}
			r.Classifications = append(r.Classifications, AnchorClassification{
				AnchorID:    a.ID,
				Class:       class,
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
	proven, orphan, staleRef, drift, phantom, awaiting, discharged, kindMismatch, regression := classCount(r)
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- Proof-bearing anchors examined: %d\n", r.AnchorCount)
	fmt.Fprintf(&b, "- Theorems found in corpus: %d\n", r.ParsedTheorems)
	fmt.Fprintf(&b, "- Proven & wired: %d\n", proven)
	fmt.Fprintf(&b, "- Orphan theorems: %d\n", len(r.Orphans))
	fmt.Fprintf(&b, "- Stale references: %d\n", staleRef)
	fmt.Fprintf(&b, "- Sorry-count drift: %d\n", drift)
	fmt.Fprintf(&b, "- Phantom-theorem (Invariant 5 violations): %d\n", phantom)
	// Awaiting is always listed by name, with a count, even at zero — a new
	// entry (e.g. a typo'd lean_theorem) must be a visible diff, never a
	// silent one (confluent-trust#130, qbp-architecture §I4 condition 1).
	fmt.Fprintf(&b, "- Awaiting (status:untested, theorem not yet written): %d\n", awaiting)
	fmt.Fprintf(&b, "- Discharged while open: %d\n", discharged)
	fmt.Fprintf(&b, "- Provenance-kind mismatches: %d\n", kindMismatch)
	fmt.Fprintf(&b, "- Verification regressions (against base): %d\n", regression)
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

	if awaiting > 0 {
		fmt.Fprint(&b, "## Awaiting (open-root discharge targets)\n\n")
		fmt.Fprint(&b, "`status: untested` anchors whose theorem doesn't exist yet, by design. Does not count toward `--strict`'s default findings; see `--awaiting-max` / `--awaiting-allow` to bound it.\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassAwaiting {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if discharged > 0 {
		fmt.Fprint(&b, "## Discharged while open\n\n")
		fmt.Fprint(&b, "A theorem now exists for an anchor still marked `status: untested` — the event Gleipnir's GL-014 tripwire watches for. lean-link reports this; it does not flip `status` itself.\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassDischargedWhileOpen {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if kindMismatch > 0 {
		fmt.Fprint(&b, "## Provenance-kind mismatches\n\n")
		fmt.Fprint(&b, "`provenance_kind: proof` anchors whose resolved declaration is a def/abbrev/example/structure, not a theorem/lemma. No proof credit given.\n\n")
		fmt.Fprint(&b, "| Anchor ID | Theorem | Detail |\n|---|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassProvenanceKindMismatch {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", cl.AnchorID, cl.TheoremName, cl.Detail)
			}
		}
		fmt.Fprint(&b, "\n")
	}

	if regression > 0 {
		fmt.Fprint(&b, "## Verification regressions (against base)\n\n")
		fmt.Fprint(&b, "An anchor's `status` flipped from something other than `untested` to `untested`, against the supplied base inventory. Demotion is allowed; it is never quiet.\n\n")
		fmt.Fprint(&b, "| Anchor ID | Detail |\n|---|---|\n")
		for _, cl := range r.Classifications {
			if cl.Class == ClassVerificationRegression {
				fmt.Fprintf(&b, "| %s | %s |\n", cl.AnchorID, cl.Detail)
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

// findByName returns the first TheoremDecl with the given name, or nil. name
// may be a bare declaration name, or a "<module>::<theorem>" join key per
// doc/design/proven-theorem-anchoring.md §3.4 — only the bare theorem part
// is matched here, since decls is already filtered to the anchor's
// proof_file by the caller (CTH #124). The module part, when present, is
// validated separately by joinKeyModuleMismatch.
func findByName(decls []TheoremDecl, name string) *TheoremDecl {
	bare := bareTheoremName(name)
	for i := range decls {
		if decls[i].Name == bare {
			return &decls[i]
		}
	}
	return nil
}

// splitJoinKey splits a "<module>::<theorem>" ref into its module and bare
// theorem parts, per doc/design/proven-theorem-anchoring.md §3.4. ok is
// false for a bare name (no "::"), in which case bare == ref.
func splitJoinKey(ref string) (module, bare string, ok bool) {
	idx := strings.LastIndex(ref, "::")
	if idx < 0 {
		return "", ref, false
	}
	return ref[:idx], ref[idx+2:], true
}

// bareTheoremName returns just the declaration-name part of ref, stripping
// an optional "<module>::" join-key prefix.
func bareTheoremName(ref string) string {
	_, bare, _ := splitJoinKey(ref)
	return bare
}

// moduleToPath converts a Lean module path ("A.B.C") to the relative file
// path lean-link expects it to resolve to ("A/B/C.lean").
func moduleToPath(module string) string {
	return strings.ReplaceAll(module, ".", "/") + ".lean"
}

// joinKeyModuleMismatch reports whether ref is a join key whose module part
// does NOT resolve to file. A bare name never mismatches. The module part
// of a join key is an identity claim (design §3.4), not decoration —
// skipping this check would let a wrong-module key silently certify an
// identity that was never actually checked.
//
// Comparison is one direction only (file must end with the module's path),
// compared on whole path components: file may legitimately be longer than
// the module's path (e.g. an absolute proof_file vs. a corpus-relative
// module), but the module's path may never be longer than file, and a
// partial component ("Rank" inside "OctonionRank") never counts as a
// match (CTH #124, second fix round — both directions of a plain
// strings.HasSuffix let a wrong module slip through).
func joinKeyModuleMismatch(ref, file string) (module string, mismatch bool) {
	module, _, ok := splitJoinKey(ref)
	if !ok {
		return "", false
	}
	if pathHasComponentSuffix(file, moduleToPath(module)) {
		return module, false
	}
	return module, true
}

// pathHasComponentSuffix reports whether path's trailing path components
// equal suffix's components exactly, component by component — never a
// partial-component match, and never true when suffix has more components
// than path (which would mean accepting a longer claimed path merely
// because the real, shorter path happens to be its tail).
func pathHasComponentSuffix(path, suffix string) bool {
	pathParts := strings.Split(path, "/")
	suffixParts := strings.Split(suffix, "/")
	if len(suffixParts) > len(pathParts) {
		return false
	}
	offset := len(pathParts) - len(suffixParts)
	for i, part := range suffixParts {
		if pathParts[offset+i] != part {
			return false
		}
	}
	return true
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

// classCount returns per-class counts for the nine classes.
func classCount(r Report) (proven, orphan, staleRef, drift, phantom, awaiting, discharged, kindMismatch, regression int) {
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
		case ClassAwaiting:
			awaiting++
		case ClassDischargedWhileOpen:
			discharged++
		case ClassProvenanceKindMismatch:
			kindMismatch++
		case ClassVerificationRegression:
			regression++
		}
	}
	return proven, orphan, staleRef, drift, phantom, awaiting, discharged, kindMismatch, regression
}

// DetectRegressions compares current against a base inventory and reports
// any anchor whose status flipped from something other than "untested" to
// "untested" — a deleted or broken proof must never read the same as a
// theorem that was never written yet (confluent-trust#130 Ask 1, condition
// 2; qbp-architecture §I4: "demotion is allowed, but it is never quiet,"
// mirroring inter#238 R11). Anchors present in only one of the two
// inventories are not compared — a regression needs the same anchor ID on
// both sides.
func DetectRegressions(base, current model.Inventory) []AnchorClassification {
	baseByID := make(map[string]model.Status, len(base.Anchors))
	for _, a := range base.Anchors {
		baseByID[a.ID] = a.Status
	}

	var out []AnchorClassification
	for _, a := range current.Anchors {
		baseStatus, ok := baseByID[a.ID]
		if !ok {
			continue
		}
		if baseStatus != model.StatusUntested && a.Status == model.StatusUntested {
			out = append(out, AnchorClassification{
				AnchorID: a.ID,
				Class:    ClassVerificationRegression,
				Detail:   fmt.Sprintf("status flipped %s -> untested against base", baseStatus),
			})
		}
	}
	return out
}
