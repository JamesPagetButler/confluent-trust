// Package leanlink provides a regex-based Lean 4 corpus parser and
// cross-reference reconciler for the cth lean-link subcommand (CTH #54).
//
// Stdlib-only: no Lean toolchain dependency at parse time.
package leanlink

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TheoremDecl captures one declaration found in a .lean file.
type TheoremDecl struct {
	// Name is the theorem / lemma / example / def identifier.
	Name string
	// Kind is one of "theorem", "lemma", "example", or "def".
	Kind string
	// File is the path relative to the corpus root.
	File string
	// AxiomUsages lists identifiers declared with the axiom keyword inside
	// the theorem body (best-effort; not a full elaboration).
	AxiomUsages []string
	// Line is the 1-indexed line where the declaration starts.
	Line int
	// SorryCount is the count of sorry tokens in the theorem body
	// (comment text excluded).
	SorryCount int
}

var (
	// reDeclStart matches the start of a theorem/lemma/example/def declaration.
	// Group 1 = kind, group 2 = name.
	reDeclStart = regexp.MustCompile(`(?m)^\s*(theorem|lemma|example|def)\s+([A-Za-z_][A-Za-z0-9_'.]*)`)

	// reSorry matches the sorry keyword as a whole word.
	reSorry = regexp.MustCompile(`\bsorry\b`)

	// reAxiomDecl matches an inline axiom declaration and captures the name.
	// Group 1 = axiom name.
	reAxiomDecl = regexp.MustCompile(`\baxiom\s+([A-Za-z_][A-Za-z0-9_'.]*)`)
)

// ParseLeanFile parses one .lean file and returns all TheoremDecl entries.
// corpusRoot is used to compute the relative File path in each TheoremDecl.
func ParseLeanFile(path string, corpusRoot string) ([]TheoremDecl, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is corpus-relative; no user-supplied traversal risk
	if err != nil {
		return nil, fmt.Errorf("leanlink: read %s: %w", path, err)
	}

	rel, err := filepath.Rel(corpusRoot, path)
	if err != nil {
		rel = path
	}

	lines := splitLines(string(data))
	// Strip comments from lines before sorry/axiom counting.
	stripped := stripComments(lines)

	// Find all declaration start positions.
	type declPos struct {
		kind string
		name string
		line int // 0-indexed
	}

	var positions []declPos
	for i, line := range lines {
		m := reDeclStart.FindStringSubmatch(line)
		if m != nil {
			positions = append(positions, declPos{kind: m[1], name: m[2], line: i})
		}
	}

	decls := make([]TheoremDecl, 0, len(positions))
	for idx, pos := range positions {
		// Body = stripped lines from declaration start up to (but not including)
		// the next top-level declaration, or end of file.
		bodyEnd := len(stripped)
		if idx+1 < len(positions) {
			bodyEnd = positions[idx+1].line
		}
		body := strings.Join(stripped[pos.line:bodyEnd], "\n")

		sorryCount := len(reSorry.FindAllString(body, -1))

		var axiomUsages []string
		for _, m := range reAxiomDecl.FindAllStringSubmatch(body, -1) {
			axiomUsages = append(axiomUsages, m[1])
		}

		decls = append(decls, TheoremDecl{
			Name:        pos.name,
			Kind:        pos.kind,
			File:        rel,
			Line:        pos.line + 1, // convert to 1-indexed
			SorryCount:  sorryCount,
			AxiomUsages: axiomUsages,
		})
	}

	return decls, nil
}

// WalkCorpus walks corpusRoot for .lean files and returns all TheoremDecl
// entries. Skips .lake/, .git/, and lake-manifest.json.
func WalkCorpus(corpusRoot string) ([]TheoremDecl, error) {
	var all []TheoremDecl
	err := filepath.WalkDir(corpusRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".lake" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if name == "lake-manifest.json" {
			return nil
		}
		if !strings.HasSuffix(name, ".lean") {
			return nil
		}
		decls, err := ParseLeanFile(path, corpusRoot)
		if err != nil {
			return err
		}
		all = append(all, decls...)
		return nil
	})
	return all, err
}

// splitLines splits src into individual lines, preserving empty lines.
func splitLines(src string) []string {
	scanner := bufio.NewScanner(strings.NewReader(src))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// stripComments removes Lean comment text from a slice of lines.
// Line comments (-- to end of line) and block comments (/- ... -/) are
// replaced with whitespace so line numbering is preserved.
func stripComments(lines []string) []string {
	out := make([]string, len(lines))
	inBlock := false
	for i, line := range lines {
		var sb strings.Builder
		j := 0
		for j < len(line) {
			if inBlock {
				// Look for end of block comment.
				if j+1 < len(line) && line[j] == '-' && line[j+1] == '/' {
					inBlock = false
					j += 2
					continue
				}
				sb.WriteByte(' ')
				j++
				continue
			}
			// Not in block: look for comment starts.
			if j+1 < len(line) && line[j] == '-' && line[j+1] == '-' {
				// Line comment: skip rest of line.
				break
			}
			if j+1 < len(line) && line[j] == '/' && line[j+1] == '-' {
				inBlock = true
				j += 2
				continue
			}
			sb.WriteByte(line[j])
			j++
		}
		out[i] = sb.String()
	}
	return out
}
