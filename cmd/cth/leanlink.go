package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/internal/leanlink"
	"github.com/JamesPagetButler/confluent-trust/store"
)

// flagStrict is the --strict flag literal, pulled out as a constant since
// it recurs across this file and its tests (golangci-lint goconst).
const flagStrict = "--strict"

// leanLinkFlags holds the parsed flags for the lean-link subcommand.
// Field order is arranged to minimise GC pointer-scan bytes per govet
// fieldalignment: the single-pointer map field comes before the two
// two-word string fields, then the pointer-free int/bool fields.
type leanLinkFlags struct {
	// awaitingAllow is the optional --awaiting-allow <id1,id2,...> allow-list.
	// Set (non-nil) only when the flag is passed, so an explicitly empty
	// list ("--awaiting-allow ''") still enforces zero tolerance rather than
	// being indistinguishable from the flag being absent.
	awaitingAllow map[string]bool
	// out is the optional -o <path> for the markdown report.
	out string
	// base is the optional --base <path> inventory to diff against for
	// verification-regression detection (confluent-trust#130).
	base string
	// awaitingMax is the optional --awaiting-max <N> ceiling. -1 means unset.
	awaitingMax int
	// updateInventory applies ProposedUpdates to the inventory and writes back.
	updateInventory bool
	// strict causes a non-zero exit when any orphan, stale-ref, drift,
	// phantom-theorem, provenance-kind-mismatch, or verification-regression
	// finding is present (useful for CI gates). Awaiting findings never
	// count by default — see awaitingMax / awaitingAllow.
	strict bool
}

// parseLeanLinkFlags parses flags for cth lean-link:
//   - -o <path>              write report markdown to path (default: stdout)
//   - --update-inventory     apply ProposedUpdates + write back atomically
//   - --strict               exit non-zero on any finding (orphan/stale/drift/
//     phantom/provenance-kind-mismatch/verification-regression)
//   - --base <path>          a base inventory to diff for verification regressions
//   - --awaiting-max <N>     under --strict, also fail if awaiting count > N
//   - --awaiting-allow <ids> under --strict, also fail if any awaiting anchor ID
//     (comma-separated) is not in this list
//
// Positional arguments (inventory path + corpus root) are returned separately.
func parseLeanLinkFlags(args []string) (lf leanLinkFlags, positional []string, err error) {
	lf.awaitingMax = -1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", flagO:
			if i+1 >= len(args) {
				return leanLinkFlags{}, nil, fmt.Errorf("lean-link: -o requires a path argument")
			}
			lf.out = args[i+1]
			i++
		case "--base":
			if i+1 >= len(args) {
				return leanLinkFlags{}, nil, fmt.Errorf("lean-link: --base requires a path argument")
			}
			lf.base = args[i+1]
			i++
		case "--awaiting-max":
			if i+1 >= len(args) {
				return leanLinkFlags{}, nil, fmt.Errorf("lean-link: --awaiting-max requires an integer argument")
			}
			n, convErr := strconv.Atoi(args[i+1])
			if convErr != nil {
				return leanLinkFlags{}, nil, fmt.Errorf("lean-link: --awaiting-max: %w", convErr)
			}
			lf.awaitingMax = n
			i++
		case "--awaiting-allow":
			if i+1 >= len(args) {
				return leanLinkFlags{}, nil, fmt.Errorf("lean-link: --awaiting-allow requires a comma-separated id list")
			}
			lf.awaitingAllow = map[string]bool{}
			for _, id := range strings.Split(args[i+1], ",") {
				id = strings.TrimSpace(id)
				if id != "" {
					lf.awaitingAllow[id] = true
				}
			}
			i++
		case "--update-inventory":
			lf.updateInventory = true
		case flagStrict:
			lf.strict = true
		default:
			positional = append(positional, args[i])
		}
	}
	return lf, positional, nil
}

// runLeanLink implements `cth lean-link <inventory.json> <lean-corpus-root> [flags]`.
//
// Modes:
//   - default (read-only): walk corpus, reconcile, emit markdown report
//   - --update-inventory: also apply ProposedUpdates + write inventory back
//   - --strict: exit non-zero if any orphan / stale-ref / drift / phantom-theorem
func runLeanLink(args []string) error {
	lf, pos, err := parseLeanLinkFlags(args)
	if err != nil {
		return err
	}
	if len(pos) != 2 {
		return errors.New("lean-link: expects two arguments: <inventory.json> <lean-corpus-root>")
	}
	invPath := pos[0]
	corpusRoot := pos[1]

	inv, err := store.LoadInventory(invPath)
	if err != nil {
		return err
	}

	theorems, err := leanlink.WalkCorpus(corpusRoot)
	if err != nil {
		return fmt.Errorf("lean-link: walk corpus %s: %w", corpusRoot, err)
	}

	spec, err := leanlink.ReadToolchainSpec(corpusRoot)
	if err != nil {
		return fmt.Errorf("lean-link: read toolchain spec: %w", err)
	}

	report := leanlink.Reconcile(inv, theorems, spec)
	report.InventoryPath = invPath
	report.CorpusRoot = corpusRoot

	if lf.base != "" {
		baseInv, baseErr := store.LoadInventory(lf.base)
		if baseErr != nil {
			return fmt.Errorf("lean-link: load base inventory %s: %w", lf.base, baseErr)
		}
		report.Classifications = append(report.Classifications, leanlink.DetectRegressions(baseInv, inv)...)
	}

	if lf.updateInventory {
		updated, applyErr := leanlink.Apply(inv, report, spec)
		if applyErr != nil {
			return fmt.Errorf("lean-link: apply updates: %w", applyErr)
		}
		if err := store.SaveInventory(updated, invPath); err != nil {
			return fmt.Errorf("lean-link: save inventory %s: %w", invPath, err)
		}
		fmt.Fprintf(os.Stderr, "lean-link: wrote updated inventory %s\n", invPath)
	}

	md := leanlink.FormatReport(report)
	if err := writeOutput(lf.out, md); err != nil {
		return err
	}

	if lf.strict {
		if err := checkStrict(report, lf); err != nil {
			return err
		}
	}

	return nil
}

// checkStrict applies --strict's gating rules to a completed report.
//
// Orphan/stale-ref/drift/phantom/provenance-kind-mismatch/
// verification-regression findings always fail. Awaiting findings never
// fail on their own (confluent-trust#130 Ask 1) — but --awaiting-max bounds
// the count, and --awaiting-allow requires every awaiting anchor ID to be
// on an explicit list, so a typo'd lean_theorem can't hide silently forever.
func checkStrict(report leanlink.Report, lf leanLinkFlags) error {
	_, _, staleRef, drift, phantom, awaiting, _, kindMismatch, regression := strictCounts(report)

	var violations []string
	if len(report.Orphans) > 0 {
		violations = append(violations, fmt.Sprintf("orphans=%d", len(report.Orphans)))
	}
	if staleRef > 0 {
		violations = append(violations, fmt.Sprintf("stale-ref=%d", staleRef))
	}
	if drift > 0 {
		violations = append(violations, fmt.Sprintf("drift=%d", drift))
	}
	if phantom > 0 {
		violations = append(violations, fmt.Sprintf("phantom=%d", phantom))
	}
	if kindMismatch > 0 {
		violations = append(violations, fmt.Sprintf("provenance-kind-mismatch=%d", kindMismatch))
	}
	if regression > 0 {
		violations = append(violations, fmt.Sprintf("verification-regression=%d", regression))
	}

	if lf.awaitingMax >= 0 && awaiting > lf.awaitingMax {
		violations = append(violations, fmt.Sprintf("awaiting=%d exceeds --awaiting-max=%d", awaiting, lf.awaitingMax))
	}
	if lf.awaitingAllow != nil {
		var notAllowed []string
		for _, cl := range report.Classifications {
			if cl.Class == leanlink.ClassAwaiting && !lf.awaitingAllow[cl.AnchorID] {
				notAllowed = append(notAllowed, cl.AnchorID)
			}
		}
		if len(notAllowed) > 0 {
			violations = append(violations, fmt.Sprintf("awaiting anchors not on --awaiting-allow: %s", strings.Join(notAllowed, ", ")))
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("lean-link: strict mode: findings detected (%s)", strings.Join(violations, " "))
	}
	return nil
}

// strictCounts tallies the per-class counts from a report for --strict mode.
func strictCounts(r leanlink.Report) (proven, orphan, staleRef, drift, phantom, awaiting, discharged, kindMismatch, regression int) {
	for _, cl := range r.Classifications {
		switch cl.Class {
		case leanlink.ClassProven:
			proven++
		case leanlink.ClassOrphan:
			orphan++
		case leanlink.ClassStaleRef:
			staleRef++
		case leanlink.ClassDrift:
			drift++
		case leanlink.ClassPhantomTheorem:
			phantom++
		case leanlink.ClassAwaiting:
			awaiting++
		case leanlink.ClassDischargedWhileOpen:
			discharged++
		case leanlink.ClassProvenanceKindMismatch:
			kindMismatch++
		case leanlink.ClassVerificationRegression:
			regression++
		}
	}
	return proven, orphan, staleRef, drift, phantom, awaiting, discharged, kindMismatch, regression
}
