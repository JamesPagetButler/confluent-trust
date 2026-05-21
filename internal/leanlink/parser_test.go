package leanlink_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/internal/leanlink"
)

func TestParseLeanFile_SingleDeclaration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lean")
	src := `theorem foo : 1 + 1 = 2 := by ring`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	decls, err := leanlink.ParseLeanFile(path, dir)
	if err != nil {
		t.Fatalf("ParseLeanFile: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(decls))
	}
	d := decls[0]
	if d.Name != "foo" {
		t.Errorf("Name: want foo, got %q", d.Name)
	}
	if d.Kind != "theorem" {
		t.Errorf("Kind: want theorem, got %q", d.Kind)
	}
	if d.SorryCount != 0 {
		t.Errorf("SorryCount: want 0, got %d", d.SorryCount)
	}
	if d.Line != 1 {
		t.Errorf("Line: want 1, got %d", d.Line)
	}
}

func TestParseLeanFile_SorryCountMultiple(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.lean")
	src := `theorem bar : True ∧ True := by
  constructor
  · sorry
  · sorry
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	decls, err := leanlink.ParseLeanFile(path, dir)
	if err != nil {
		t.Fatalf("ParseLeanFile: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(decls))
	}
	if decls[0].SorryCount != 2 {
		t.Errorf("SorryCount: want 2, got %d", decls[0].SorryCount)
	}
}

func TestParseLeanFile_SkipSorriesInLineComments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "comments.lean")
	src := `-- This is not a sorry: sorry in a line comment
theorem baz : 1 = 1 := by
  rfl -- also not a sorry here
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	decls, err := leanlink.ParseLeanFile(path, dir)
	if err != nil {
		t.Fatalf("ParseLeanFile: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(decls))
	}
	if decls[0].SorryCount != 0 {
		t.Errorf("SorryCount: want 0 (sorry only in comments), got %d", decls[0].SorryCount)
	}
}

func TestParseLeanFile_SkipSorriesInBlockComments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "block.lean")
	src := `/- This block comment says sorry -/
theorem qux : 2 = 2 := by
  /- another sorry inside a block comment -/
  rfl
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	decls, err := leanlink.ParseLeanFile(path, dir)
	if err != nil {
		t.Fatalf("ParseLeanFile: %v", err)
	}
	if len(decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(decls))
	}
	if decls[0].SorryCount != 0 {
		t.Errorf("SorryCount: want 0 (sorry only in block comments), got %d", decls[0].SorryCount)
	}
}

func TestParseLeanFile_MultipleDeclarations(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "multi_decl.lean")
	src := `theorem alpha : True := trivial
lemma beta : 1 = 1 := rfl
def gamma := 42
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	decls, err := leanlink.ParseLeanFile(path, dir)
	if err != nil {
		t.Fatalf("ParseLeanFile: %v", err)
	}
	if len(decls) != 3 {
		t.Fatalf("expected 3 decls, got %d", len(decls))
	}
	wantNames := []string{"alpha", "beta", "gamma"}
	for i, d := range decls {
		if d.Name != wantNames[i] {
			t.Errorf("decl[%d].Name: want %q, got %q", i, wantNames[i], d.Name)
		}
	}
}

func TestWalkCorpus_FindsAllTheorems(t *testing.T) {
	t.Parallel()
	// Use the checked-in testdata/lean fixture corpus.
	corpusRoot := "../../testdata/lean"
	decls, err := leanlink.WalkCorpus(corpusRoot)
	if err != nil {
		t.Fatalf("WalkCorpus: %v", err)
	}
	// We expect to find at least 4 theorems across the fixture files.
	if len(decls) < 4 {
		t.Errorf("WalkCorpus: expected >= 4 theorems, got %d", len(decls))
	}
	// Verify known theorems are present.
	names := make(map[string]bool, len(decls))
	for _, d := range decls {
		names[d.Name] = true
	}
	for _, want := range []string{"hurwitz_theorem", "sedenion_assoc", "phantom_proof", "unreferenced_orphan"} {
		if !names[want] {
			t.Errorf("WalkCorpus: expected theorem %q to be found", want)
		}
	}
}

func TestWalkCorpus_SedenionHasOneSorry(t *testing.T) {
	t.Parallel()
	corpusRoot := "../../testdata/lean"
	decls, err := leanlink.WalkCorpus(corpusRoot)
	if err != nil {
		t.Fatalf("WalkCorpus: %v", err)
	}
	for _, d := range decls {
		if d.Name == "sedenion_assoc" {
			if d.SorryCount != 1 {
				t.Errorf("sedenion_assoc: SorryCount want 1, got %d", d.SorryCount)
			}
			return
		}
	}
	t.Error("sedenion_assoc not found in corpus")
}
