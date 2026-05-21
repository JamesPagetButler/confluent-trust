package leanlink

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// ToolchainSpec captures one verification environment read from a Lean corpus.
type ToolchainSpec struct {
	// Libraries is keyed by library name (e.g. "mathlib", "std").
	Libraries map[string]model.LibraryRef
	// Toolchain is the contents of lean-toolchain (e.g. "leanprover/lean4:v4.30.0-rc2").
	Toolchain string
}

// lakePackage is the per-package shape inside lake-manifest.json.
type lakePackage struct {
	URL      string `json:"url"`
	Rev      string `json:"rev"`
	Name     string `json:"name"`
	InputRev string `json:"inputRev"`
}

// lakeManifest is the top-level shape of lake-manifest.json (Lake 4 format).
type lakeManifest struct {
	Packages []lakePackage `json:"packages"`
}

// ReadToolchain reads <corpusRoot>/lean-toolchain (a single-line file).
// Returns an empty string + nil error when the file is absent.
func ReadToolchain(corpusRoot string) (string, error) {
	p := filepath.Join(corpusRoot, "lean-toolchain")
	data, err := os.ReadFile(p) // #nosec G304 -- corpusRoot is CLI-supplied; no traversal risk
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("leanlink: read lean-toolchain: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadLakeManifest reads <corpusRoot>/lake-manifest.json and extracts library
// refs keyed by library name. Returns an empty map + nil error when absent.
func ReadLakeManifest(corpusRoot string) (map[string]model.LibraryRef, error) {
	p := filepath.Join(corpusRoot, "lake-manifest.json")
	data, err := os.ReadFile(p) // #nosec G304 -- corpusRoot is CLI-supplied; no traversal risk
	if os.IsNotExist(err) {
		return map[string]model.LibraryRef{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leanlink: read lake-manifest.json: %w", err)
	}
	var manifest lakeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("leanlink: parse lake-manifest.json: %w", err)
	}
	libs := make(map[string]model.LibraryRef, len(manifest.Packages))
	for _, pkg := range manifest.Packages {
		libs[pkg.Name] = model.LibraryRef{
			SHA: pkg.Rev,
			Ref: pkg.InputRev,
			URL: pkg.URL,
		}
	}
	return libs, nil
}

// ReadToolchainSpec composes ReadToolchain + ReadLakeManifest.
func ReadToolchainSpec(corpusRoot string) (ToolchainSpec, error) {
	toolchain, err := ReadToolchain(corpusRoot)
	if err != nil {
		return ToolchainSpec{}, err
	}
	libs, err := ReadLakeManifest(corpusRoot)
	if err != nil {
		return ToolchainSpec{}, err
	}
	return ToolchainSpec{Toolchain: toolchain, Libraries: libs}, nil
}
