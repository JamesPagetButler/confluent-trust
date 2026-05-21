package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/JamesPagetButler/confluent-trust/internal/leanlink"
	"github.com/JamesPagetButler/confluent-trust/store"
)

// leanLinkFlags holds the parsed flags for the lean-link subcommand.
type leanLinkFlags struct {
	// out is the optional -o <path> for the markdown report.
	out string
	// updateInventory applies ProposedUpdates to the inventory and writes back.
	updateInventory bool
	// strict causes a non-zero exit when any orphan, stale-ref, drift, or
	// phantom-theorem finding is present (useful for CI gates).
	strict bool
}

// parseLeanLinkFlags parses flags for cth lean-link:
//   - -o <path>            write report markdown to path (default: stdout)
//   - --update-inventory   apply ProposedUpdates + write back atomically
//   - --strict             exit non-zero on any finding (orphan/stale/drift/phantom)
//
// Positional arguments (inventory path + corpus root) are returned separately.
func parseLeanLinkFlags(args []string) (lf leanLinkFlags, positional []string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", flagO:
			if i+1 >= len(args) {
				return leanLinkFlags{}, nil, fmt.Errorf("lean-link: -o requires a path argument")
			}
			lf.out = args[i+1]
			i++
		case "--update-inventory":
			lf.updateInventory = true
		case "--strict":
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
		_, _, staleRef, drift, phantom := strictCounts(report)
		hasFindings := len(report.Orphans) > 0 || staleRef > 0 || drift > 0 || phantom > 0
		if hasFindings {
			return fmt.Errorf("lean-link: strict mode: findings detected (orphans=%d stale-ref=%d drift=%d phantom=%d)",
				len(report.Orphans), staleRef, drift, phantom)
		}
	}

	return nil
}

// strictCounts tallies the non-passing classes from a report for --strict mode.
func strictCounts(r leanlink.Report) (proven, orphan, staleRef, drift, phantom int) {
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
		}
	}
	return proven, orphan, staleRef, drift, phantom
}
