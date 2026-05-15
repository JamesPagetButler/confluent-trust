// Command cth is the CTH CLI. Subcommands:
//
//	cth analyse <inventory.json>            full markdown report
//	cth health <inventory.json>             compact dashboard only
//	cth merge <a.json> <b.json>             merge two programmes (writes JSON)
//	cth compare <old.json> <new.json>       compression velocity Δρ/Δn
//	cth fork <inventory.json>               per-branch health comparison
//	cth check-branch <inventory.json>       branch consistency report
//	cth score <inventory.json>              score predictions against observations
//
// Each command reads from the named file(s) and writes to stdout (or the
// path given by -o). Inventories are loaded through store.LoadInventory
// so legacy v0.1 binary confluences are migrated automatically.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/compute"
	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/report"
	"github.com/JamesPagetButler/confluent-trust/store"
)

const usage = `cth — Confluent Trust Hypergraph CLI

Usage:
  cth analyse <inventory.json> [-o out.md]                    full markdown report
  cth health <inventory.json> [-o out.txt]                    compact dashboard
  cth merge <a.json> <b.json> [-o out.json]                   merge two programmes
  cth compare <old.json> <new.json> [-o out.md]               compression velocity
  cth fork <inventory.json> [-o out.md]                       per-branch health
  cth check-branch <inventory.json> [-o out.md]               branch consistency
  cth score <inventory.json> [-o out.md]                      score predictions
  cth score <inventory.json> --prediction <ID> [-o out.md]    single-anchor detail
  cth score <inventory.json> --regime [-o out.md]             group by regime

The library API (model + compute + store + report packages) is the
recommended consumption path for Go callers; this CLI exists for
operators, scripts, and CI/CD.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	if err := dispatch(cmd, args); err != nil {
		fmt.Fprintf(os.Stderr, "cth %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func dispatch(cmd string, args []string) error {
	switch cmd {
	case "analyse", "analyze":
		return runAnalyse(args)
	case "health":
		return runHealth(args)
	case "merge":
		return runMerge(args)
	case "compare":
		return runCompare(args)
	case "fork":
		return runFork(args)
	case "check-branch":
		return runCheckBranch(args)
	case "score":
		return runScore(args)
	case "help", "-h", "--help":
		_, err := fmt.Fprintln(os.Stdout, usage)
		return err
	default:
		return fmt.Errorf("unknown subcommand %q\n\n%s", cmd, usage)
	}
}

// parseFlags is a small wrapper around flag.NewFlagSet that pulls out a
// -o option and returns it plus the remaining positional arguments.
// Accepts flags in any position relative to positional args (e.g.
// `cth merge a.json b.json -o out.json` and `cth merge -o out.json
// a.json b.json` both work).
func parseFlags(name string, args []string) (out string, positional []string, err error) {
	// Split into flag tokens and positional tokens manually so flags can
	// come before, after, or interleaved with positionals.
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--o":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("%s: -o requires a path argument", name)
			}
			flagArgs = append(flagArgs, args[i], args[i+1])
			i++
		default:
			positional = append(positional, args[i])
		}
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&out, "o", "", "write output to this path (default stdout)")
	if err := fs.Parse(flagArgs); err != nil {
		return "", nil, err
	}
	return out, positional, nil
}

// writeOutput sends content either to stdout or to the given path.
// When path is non-empty the write is atomic (path.tmp + rename).
func writeOutput(out, content string) error {
	if out == "" {
		_, err := io.WriteString(os.Stdout, content)
		return err
	}
	tmp := out + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil { // #nosec G306 -- output files are user content
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, out); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, out, err)
	}
	return nil
}

func runAnalyse(args []string) error {
	out, pos, err := parseFlags("analyse", args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("analyse: expects one inventory.json argument")
	}
	inv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}
	fa := report.RunFullAnalysis(inv, nil)
	return writeOutput(out, report.MarkdownReport(inv, fa))
}

func runHealth(args []string) error {
	out, pos, err := parseFlags("health", args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("health: expects one inventory.json argument")
	}
	inv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}
	fa := report.RunFullAnalysis(inv, nil)
	return writeOutput(out, report.Dashboard(inv, fa))
}

func runMerge(args []string) error {
	out, pos, err := parseFlags("merge", args)
	if err != nil {
		return err
	}
	if len(pos) != 2 {
		return errors.New("merge: expects two inventory.json arguments")
	}
	a, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}
	b, err := store.LoadInventory(pos[1])
	if err != nil {
		return err
	}
	merged, _ := compute.MergeProgrammes(a, b)
	if out == "" {
		// Default merge output is JSON to stdout — most callers want
		// the merged inventory in a re-loadable form.
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(merged)
	}
	return store.SaveInventory(merged, out)
}

// scoreFlags holds the parsed flags for the score subcommand.
type scoreFlags struct {
	out        string
	prediction string // --prediction <ID>; empty means "all"
	byRegime   bool   // --regime
}

// parseScoreFlags parses the flags specific to the score subcommand:
// -o <path>, --prediction <ID>, and --regime. Positional arguments are
// returned separately so they can be validated by the caller.
func parseScoreFlags(args []string) (sf scoreFlags, positional []string, err error) {
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--o":
			if i+1 >= len(args) {
				return scoreFlags{}, nil, fmt.Errorf("score: -o requires a path argument")
			}
			flagArgs = append(flagArgs, "-o", args[i+1])
			i++
		case "--prediction":
			if i+1 >= len(args) {
				return scoreFlags{}, nil, fmt.Errorf("score: --prediction requires an ID argument")
			}
			flagArgs = append(flagArgs, "--prediction", args[i+1])
			i++
		case "--regime":
			flagArgs = append(flagArgs, "--regime")
		default:
			positional = append(positional, args[i])
		}
	}

	fs := flag.NewFlagSet("score", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&sf.out, "o", "", "write output to this path (default stdout)")
	fs.StringVar(&sf.prediction, "prediction", "", "score a single anchor by ID")
	fs.BoolVar(&sf.byRegime, "regime", false, "group output by regime")
	if err := fs.Parse(flagArgs); err != nil {
		return scoreFlags{}, nil, err
	}
	return sf, positional, nil
}

// runScore implements `cth score <inventory.json> [flags]`.
//
// Three modes (--prediction and --regime are mutually exclusive):
//   - default: table of all anchors with predicted_value set
//   - --prediction <ID>: single-anchor detail for the named anchor
//   - --regime: anchors grouped into the four ScoreRegime sections
func runScore(args []string) error {
	sf, pos, err := parseScoreFlags(args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("score: expects one inventory.json argument")
	}
	if sf.prediction != "" && sf.byRegime {
		return errors.New("score: --prediction and --regime are mutually exclusive")
	}

	inv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}

	switch {
	case sf.prediction != "":
		content, err := scoreOnePrediction(inv, sf.prediction)
		if err != nil {
			return err
		}
		return writeOutput(sf.out, content)
	case sf.byRegime:
		return writeOutput(sf.out, scoreByRegime(inv))
	default:
		return writeOutput(sf.out, scoreAllPredictions(inv))
	}
}

// scoreRow holds the computed columns for one row of the score table.
// Flat layout avoids a pointer indirection: Score fields are inlined.
// scored is false when no MeasuredValue exists (untested). Fields are
// ordered for optimal struct alignment.
type scoreRow struct {
	id             string
	status         string
	regime         string
	predicted      float64
	measured       float64
	discrepancyPct float64
	confirmInfo    float64
	scored         bool
}

// collectScoreRows gathers one scoreRow per anchor that has a predicted_value.
// Score fields are inlined (flat layout, no pointer indirection).
func collectScoreRows(inv model.Inventory) ([]scoreRow, error) {
	var rows []scoreRow
	for _, a := range inv.Anchors {
		if a.PredictedValue == nil {
			continue
		}
		row := scoreRow{
			id:        a.ID,
			status:    a.Status.String(),
			predicted: *a.PredictedValue,
		}
		if a.MeasuredValue != nil {
			s, err := compute.ScorePrediction(compute.KindScalar, *a.PredictedValue, *a.MeasuredValue)
			if err != nil {
				return nil, fmt.Errorf("score: anchor %s: %w", a.ID, err)
			}
			row.scored = true
			row.measured = *a.MeasuredValue
			row.discrepancyPct = s.DiscrepancyPct
			row.confirmInfo = s.ConfirmInfo
			row.regime = s.Regime.String()
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// scoreAllPredictions emits a markdown table of all anchors with predictions.
func scoreAllPredictions(inv model.Inventory) string {
	rows, err := collectScoreRows(inv)
	if err != nil {
		return fmt.Sprintf("_error: %v_\n", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Prediction scores — %s v%s\n\n", inv.Programme, inv.Version)

	scored, untested := 0, 0
	for _, r := range rows {
		if r.scored {
			scored++
		} else {
			untested++
		}
	}
	fmt.Fprintf(&b, "**%d anchors with predictions** (%d scored, %d untested, no observation)\n\n", len(rows), scored, untested)

	fmt.Fprint(&b, "| Anchor ID | Status | Predicted | Measured | Discrepancy | Regime | Confirm bits |\n")
	fmt.Fprint(&b, "|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		if r.scored {
			fmt.Fprintf(&b, "| %s | %s | %.6g | %.6g | %.4f%% | %s | %.4f |\n",
				r.id, r.status, r.predicted, r.measured,
				r.discrepancyPct, r.regime, r.confirmInfo)
		} else {
			fmt.Fprintf(&b, "| %s | %s | %.6g | — | — | — | — |\n",
				r.id, r.status, r.predicted)
		}
	}
	return b.String()
}

// scoreOnePrediction emits per-anchor detail for the named anchor ID.
func scoreOnePrediction(inv model.Inventory, id string) (string, error) {
	var found *model.Anchor
	for i := range inv.Anchors {
		if inv.Anchors[i].ID == id {
			found = &inv.Anchors[i]
			break
		}
	}
	if found == nil {
		return "", fmt.Errorf("score: anchor %q not found in inventory", id)
	}
	if found.PredictedValue == nil {
		return "", fmt.Errorf("score: anchor %q has no predicted_value", id)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Prediction score — %s\n\n", id)
	fmt.Fprintf(&b, "**Programme:** %s v%s\n\n", inv.Programme, inv.Version)
	fmt.Fprintf(&b, "| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| ID | %s |\n", found.ID)
	fmt.Fprintf(&b, "| Name | %s |\n", found.Name)
	fmt.Fprintf(&b, "| Status | %s |\n", found.Status.String())
	fmt.Fprintf(&b, "| Tier | %d |\n", int(found.Tier))
	fmt.Fprintf(&b, "| Predicted | %.6g |\n", *found.PredictedValue)
	if found.PredictedUnit != "" {
		fmt.Fprintf(&b, "| Unit | %s |\n", found.PredictedUnit)
	}

	if found.MeasuredValue == nil {
		fmt.Fprintf(&b, "| Measured | — |\n")
		fmt.Fprintf(&b, "| Discrepancy | — |\n")
		fmt.Fprintf(&b, "| Regime | — |\n")
		fmt.Fprintf(&b, "| Confirm bits | — |\n\n")
		fmt.Fprintf(&b, "_No observation yet (untested)._\n")
		return b.String(), nil
	}

	s, err := compute.ScorePrediction(compute.KindScalar, *found.PredictedValue, *found.MeasuredValue)
	if err != nil {
		return "", fmt.Errorf("score: anchor %s: %w", id, err)
	}

	fmt.Fprintf(&b, "| Measured | %.6g |\n", *found.MeasuredValue)
	if found.MeasuredError != nil {
		fmt.Fprintf(&b, "| Measured error | %.6g |\n", *found.MeasuredError)
	}
	if found.MeasuredSource != "" {
		fmt.Fprintf(&b, "| Measured source | %s |\n", found.MeasuredSource)
	}
	if found.LastTestedAt != nil {
		fmt.Fprintf(&b, "| Last tested at | %s |\n", *found.LastTestedAt)
	}
	fmt.Fprintf(&b, "| Delta | %.6g |\n", s.Delta)
	fmt.Fprintf(&b, "| Discrepancy | %.4f%% |\n", s.DiscrepancyPct)
	fmt.Fprintf(&b, "| Regime | %s |\n", s.Regime.String())
	fmt.Fprintf(&b, "| Confirm bits | %.4f |\n", s.ConfirmInfo)
	return b.String(), nil
}

// regimeSection is one bucket in the regime-grouped output.
type regimeSection struct {
	name    string
	entries []string
}

// scoreByRegime emits anchors grouped into four regime sections plus Untested.
func scoreByRegime(inv model.Inventory) string {
	rows, err := collectScoreRows(inv)
	if err != nil {
		return fmt.Sprintf("_error: %v_\n", err)
	}

	sections := []regimeSection{
		{name: "Laminar"},
		{name: "Low Sediment"},
		{name: "Moderate"},
		{name: "Heavy"},
		{name: "Untested"},
	}

	for _, r := range rows {
		if !r.scored {
			sections[4].entries = append(sections[4].entries, r.id)
			continue
		}
		switch r.regime {
		case compute.ScoreRegimeLaminar.String():
			sections[0].entries = append(sections[0].entries, r.id)
		case compute.ScoreRegimeLowSediment.String():
			sections[1].entries = append(sections[1].entries, r.id)
		case compute.ScoreRegimeModerate.String():
			sections[2].entries = append(sections[2].entries, r.id)
		case compute.ScoreRegimeHeavy.String():
			sections[3].entries = append(sections[3].entries, r.id)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Prediction scores by regime — %s v%s\n\n", inv.Programme, inv.Version)
	for _, sec := range sections {
		fmt.Fprintf(&b, "## %s\n\n", sec.name)
		if len(sec.entries) == 0 {
			fmt.Fprint(&b, "_None._\n\n")
			continue
		}
		for _, id := range sec.entries {
			fmt.Fprintf(&b, "- %s\n", id)
		}
		fmt.Fprint(&b, "\n")
	}
	return b.String()
}

func runCompare(args []string) error {
	out, pos, err := parseFlags("compare", args)
	if err != nil {
		return err
	}
	if len(pos) != 2 {
		return errors.New("compare: expects two inventory.json arguments (old, new)")
	}
	oldInv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}
	newInv, err := store.LoadInventory(pos[1])
	if err != nil {
		return err
	}
	prev, _ := compute.NetCompression(oldInv, nil)
	cur, _ := compute.NetCompression(newInv, nil)
	velocity := compute.CompressionVelocity(
		compute.VersionSnapshot{Rho: prev, AnchorCount: anchorCount(oldInv)},
		compute.VersionSnapshot{Rho: cur, AnchorCount: anchorCount(newInv)},
	)
	body := fmt.Sprintf("# Compression velocity\n\n- old %s v%s: ρ_net = %.4f (n=%d)\n- new %s v%s: ρ_net = %.4f (n=%d)\n- Δρ/Δn = %g\n",
		oldInv.Programme, oldInv.Version, prev, anchorCount(oldInv),
		newInv.Programme, newInv.Version, cur, anchorCount(newInv),
		velocity)
	return writeOutput(out, body)
}

func runFork(args []string) error {
	out, pos, err := parseFlags("fork", args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("fork: expects one inventory.json argument")
	}
	inv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}
	if len(inv.ForkPoints) == 0 {
		return writeOutput(out, fmt.Sprintf("# Fork health — %s v%s\n\n_No fork_points in this inventory._\n",
			inv.Programme, inv.Version))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Fork health — %s v%s\n\n", inv.Programme, inv.Version)
	for _, fp := range inv.ForkPoints {
		cmp := compute.CompareBranches(inv, fp, nil)
		fmt.Fprintf(&b, "## Fork %s — %s\n\n", fp.ID, fp.Question)
		fmt.Fprintf(&b, "Branches: %d. Lower-deficit: **%s**. Higher-ρ_net: **%s**.\n\n",
			len(cmp.Branches), cmp.LowerDeficitBranch, cmp.HigherRhoNetBranch)
		fmt.Fprint(&b, "| Branch | ρ_net | Δ(deficit) | R_c | anchors |\n|---|---|---|---|---|\n")
		for _, r := range cmp.Branches {
			fmt.Fprintf(&b, "| %s | %.4f | %.4f | %.4f | %d |\n",
				r.BranchID, r.RhoNet, r.InformationDeficit, r.CoherenceRatio, r.AnchorCount)
		}
		fmt.Fprint(&b, "\n")
	}
	return writeOutput(out, b.String())
}

func runCheckBranch(args []string) error {
	out, pos, err := parseFlags("check-branch", args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("check-branch: expects one inventory.json argument")
	}
	inv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Branch consistency — %s v%s\n\n", inv.Programme, inv.Version)
	if len(inv.ForkPoints) == 0 {
		fmt.Fprintln(&b, "_No fork_points to check._")
		return writeOutput(out, b.String())
	}

	var totalViolations int
	for _, fp := range inv.ForkPoints {
		fmt.Fprintf(&b, "## Fork %s\n\n", fp.ID)
		violations := compute.CheckAllAnchors(inv, fp)
		for _, branchID := range branchIDsSorted(violations) {
			vs := violations[branchID]
			if len(vs) == 0 {
				fmt.Fprintf(&b, "- branch **%s**: ✓ no violations\n", branchID)
				continue
			}
			totalViolations += len(vs)
			fmt.Fprintf(&b, "- branch **%s**: %d violation(s)\n", branchID, len(vs))
			for _, v := range vs {
				fmt.Fprintf(&b, "  - %s — %s\n", v.AnchorID, v.Description)
			}
		}
		fmt.Fprint(&b, "\n")
	}
	fmt.Fprintf(&b, "**Total violations across all forks: %d**\n", totalViolations)
	return writeOutput(out, b.String())
}

func anchorCount(inv model.Inventory) int {
	return len(inv.Axioms) + len(inv.Anchors)
}

// branchIDsSorted returns the keys of m in deterministic order.
func branchIDsSorted(m map[string][]compute.ConsistencyViolation) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// We don't import sort here to keep the import set tight; bubble
	// sort over a typically-2-element slice is fine.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
