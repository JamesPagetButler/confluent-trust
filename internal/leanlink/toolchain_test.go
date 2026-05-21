package leanlink_test

import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/internal/leanlink"
)

const testCorpusRoot = "../../testdata/lean"

func TestReadToolchain_ReturnsString(t *testing.T) {
	t.Parallel()
	tc, err := leanlink.ReadToolchain(testCorpusRoot)
	if err != nil {
		t.Fatalf("ReadToolchain: %v", err)
	}
	const want = "leanprover/lean4:v4.30.0-rc2"
	if tc != want {
		t.Errorf("ReadToolchain: want %q, got %q", want, tc)
	}
}

func TestReadToolchain_AbsentFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tc, err := leanlink.ReadToolchain(dir)
	if err != nil {
		t.Fatalf("ReadToolchain (absent): %v", err)
	}
	if tc != "" {
		t.Errorf("ReadToolchain (absent): want empty, got %q", tc)
	}
}

func TestReadLakeManifest_ParsesMathlibAndStd(t *testing.T) {
	t.Parallel()
	libs, err := leanlink.ReadLakeManifest(testCorpusRoot)
	if err != nil {
		t.Fatalf("ReadLakeManifest: %v", err)
	}
	if len(libs) != 2 {
		t.Fatalf("ReadLakeManifest: want 2 libs, got %d", len(libs))
	}

	mathlib, ok := libs["mathlib"]
	if !ok {
		t.Fatal("ReadLakeManifest: expected mathlib entry")
	}
	if mathlib.SHA != "abc123def456789012345678901234567890abcd" {
		t.Errorf("mathlib.SHA: got %q", mathlib.SHA)
	}
	if mathlib.Ref != "v4.30.0" {
		t.Errorf("mathlib.Ref: got %q", mathlib.Ref)
	}
	if mathlib.URL != "https://github.com/leanprover-community/mathlib4" {
		t.Errorf("mathlib.URL: got %q", mathlib.URL)
	}

	std, ok := libs["std"]
	if !ok {
		t.Fatal("ReadLakeManifest: expected std entry")
	}
	if std.SHA != "deadbeef12345678901234567890abcdef123456" {
		t.Errorf("std.SHA: got %q", std.SHA)
	}
}

func TestReadLakeManifest_AbsentFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	libs, err := leanlink.ReadLakeManifest(dir)
	if err != nil {
		t.Fatalf("ReadLakeManifest (absent): %v", err)
	}
	if len(libs) != 0 {
		t.Errorf("ReadLakeManifest (absent): want empty map, got %d entries", len(libs))
	}
}

func TestReadToolchainSpec_Composed(t *testing.T) {
	t.Parallel()
	spec, err := leanlink.ReadToolchainSpec(testCorpusRoot)
	if err != nil {
		t.Fatalf("ReadToolchainSpec: %v", err)
	}
	if spec.Toolchain == "" {
		t.Error("ReadToolchainSpec: Toolchain should not be empty")
	}
	if len(spec.Libraries) == 0 {
		t.Error("ReadToolchainSpec: Libraries should not be empty")
	}
}
