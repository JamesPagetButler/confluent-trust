package validate

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestSchemaCompiles ensures the embedded schema parses and compiles.
func TestSchemaCompiles(t *testing.T) {
	if _, err := loadSchema(); err != nil {
		t.Fatalf("schema failed to compile: %v", err)
	}
}

// TestSchemaInSync ensures internal/validate/schema.json (embedded) is
// semantically equivalent to the canonical schema/inventory.schema.json.
//
// Rationale: per sprint-1-closeout-2026-05-17 seq=12 Notary-bootstrap
// target #4 — Notary discipline upgrades verification from
// "compares-equal-bytes" to "validates-equivalent-meaning", catching
// drift only when the schemas actually differ. Key reordering, whitespace
// normalisation, or trailing-newline changes preserve semantics but would
// break a byte-equal check; this test is immune to such cosmetic changes.
// Update both files with `go generate ./...` if you change one.
func TestSchemaInSync(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "schema", "inventory.schema.json"))
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}

	var embeddedDoc, canonicalDoc any
	if err := json.Unmarshal(rawSchema, &embeddedDoc); err != nil {
		t.Fatalf("parse embedded schema: %v", err)
	}
	if err := json.Unmarshal(canonical, &canonicalDoc); err != nil {
		t.Fatalf("parse canonical schema: %v", err)
	}

	if !reflect.DeepEqual(embeddedDoc, canonicalDoc) {
		t.Fatal("internal/validate/schema.json semantically diverged from schema/inventory.schema.json — re-copy with `go generate ./...`")
	}
}

// TestSchemaInSync_ByteEqual is a hygiene signal: it logs (but does not fail)
// when the two schema files are byte-different. Semantic equality is the
// load-bearing invariant (see TestSchemaInSync above); byte equality is
// desirable for diff hygiene but not required for correctness.
func TestSchemaInSync_ByteEqual(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "schema", "inventory.schema.json"))
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}
	if !bytes.Equal(rawSchema, canonical) {
		t.Logf("internal/validate/schema.json byte-different from schema/inventory.schema.json — semantically still in sync per TestSchemaInSync, but consider re-copying for hygiene (`go generate ./...`)")
	}
}

// TestValidate_AllFixtures validates every JSON file in testdata/
// against the schema. New fixtures must pass without modification.
func TestValidate_AllFixtures(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata")
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("read testdata/: %v", err)
	}
	var found int
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(fixturesDir, name)
			data, err := os.ReadFile(path) // #nosec G304 -- test reads a fixed-prefix testdata path
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if err := Inventory(data); err != nil {
				t.Fatalf("validate %s: %v", path, err)
			}
		})
		found++
	}
	if found == 0 {
		t.Fatal("no .json fixtures found in testdata/")
	}
}

func TestSchemaVersion(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"absent defaults to v0.1", `{"programme":"X","version":"0.1","axioms":[],"anchors":[],"chains":[]}`, "v0.1"},
		{"explicit v0.2", `{"programme":"X","version":"0.1","schema_version":"v0.2","axioms":[],"anchors":[],"chains":[]}`, "v0.2"},
		{"whitespace trimmed", `{"programme":"X","version":"0.1","schema_version":"  v0.2  ","axioms":[],"anchors":[],"chains":[]}`, "v0.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SchemaVersion([]byte(tt.data))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestC1_DerivationMayCarryVerification exercises the 0.3.3 C1 relaxation:
// verification-fields are admitted on {proof, derivation}, still forbidden on
// every other provenance_kind (FAULT-S4-005 burn-down; cth-implementor ruling).
func TestC1_DerivationMayCarryVerification(t *testing.T) {
	const verif = `"proof_system":"lean4","proof_file":"proofs/X.lean","proof_state":"verified","sorry_count":0,` +
		`"verification":{"toolchain":"leanprover/lean4:v4.30.0","libraries":{"mathlib":{"ref":"abc123","sha":"abc123"}},` +
		`"verified_at":"2026-01-01T00:00:00Z","verifier":"ci","result":"verified","axiom_closure":["propext","Classical.choice","Quot.sound"]}`
	doc := func(pk, extra string) string {
		return `{"programme":"X","version":"0.1","schema_version":"v0.3","axioms":[],"chains":[],"anchors":[{` +
			`"id":"A-x","name":"x","tier":1,"provenance":"T","status":"coherent","description":"d","prediction_chain":[],` +
			`"provenance_kind":"` + pk + `"` + extra + `}]}`
	}
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"derivation WITH verification is admitted (the relaxation)", doc("derivation", ","+verif), false},
		{"bare derivation (no proof-fields) is fine", doc("derivation", ""), false},
		{"proof WITH verification still admitted", doc("proof", ","+verif), false},
		{"theory WITH verification still forbidden (C1 holds elsewhere)", doc("theory", `,"verification":{"toolchain":"t","libraries":{},"verified_at":"2026-01-01T00:00:00Z","verifier":"ci","result":"verified"}`), true},
		{"experiment WITH proof_state still forbidden", doc("experiment", `,"proof_state":"verified"`), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Inventory([]byte(tt.data))
			if tt.wantErr && err == nil {
				t.Fatalf("expected schema violation, got nil (C1 relaxation over-broad)")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected valid, got error: %v", err)
			}
		})
	}
}

// TestDecisionState_FourBucket exercises the 0.3.4 four-bucket root fields
// (#654 D3): decision_state + kill_condition (array<object>) on roots, the
// OpenNeedsKill invariant (open ⇒ non-empty kill_condition), the per-entry
// closure/discharge conditional, and the four QBP-general top-level root lists
// (cth-implementor, confluent-trust #102/#103).
func TestDecisionState_FourBucket(t *testing.T) {
	ax := func(extra string) string {
		return `{"programme":"X","version":"0.1","schema_version":"v0.3","chains":[],"anchors":[],"axioms":[{` +
			`"id":"AXIOM-1","name":"a","statement":"s","derivable":false` + extra + `}]}`
	}
	const killRuling = `,"decision_state":"open","kill_condition":[{"kill":"a real ledger-observable falsifier","closure":"ruling-rescope"}]`
	const killDeriv = `,"decision_state":"open","kill_condition":[{"kill":"a real ledger-observable falsifier","closure":"derivation","discharge":"PROOF-y"}]`
	lists := `{"programme":"X","version":"0.1","schema_version":"v0.3","chains":[],"anchors":[],"axioms":[],` +
		`"meta_principles":[{"id":"META-2","name":"m","statement":"s","decision_state":"open","kill_condition":[{"kill":"a real meta falsifier text","closure":"ruling-rescope"}]}],` +
		`"interpretations":[{"id":"INTERP-x","name":"i","statement":"s","provenance_kind":"philosophy","decision_state":"open","kill_condition":[{"kill":"a real interp falsifier text","closure":"ruling-rescope"}]}],` +
		`"retired_axioms":[{"id":"AXIOM-2","name":"r","statement":"s","notes":"content re-rooted to POST-x"}],` +
		`"retired_principles":[{"id":"DERIV-h","name":"r","statement":"s","notes":"editorial split"}]}`
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"open + ruling-rescope kill, no discharge", ax(killRuling), false},
		{"open + derivation kill + discharge", ax(killDeriv), false},
		{"settled, no kill", ax(`,"decision_state":"settled"`), false},
		{"plain axiom (no decision fields)", ax(``), false},
		{"four top-level root lists", lists, false},
		{"open with NO kill_condition (OpenNeedsKill fires)", ax(`,"decision_state":"open"`), true},
		{"open with EMPTY kill_condition", ax(`,"decision_state":"open","kill_condition":[]`), true},
		{"closure=derivation WITHOUT discharge", ax(`,"decision_state":"open","kill_condition":[{"kill":"a real falsifier text","closure":"derivation"}]`), true},
		{"kill entry missing closure", ax(`,"decision_state":"open","kill_condition":[{"kill":"a real falsifier text"}]`), true},
		{"kill too short (placeholder guard)", ax(`,"decision_state":"open","kill_condition":[{"kill":"short","closure":"ruling-rescope"}]`), true},
		{"bad decision_state value", ax(`,"decision_state":"maybe","kill_condition":[{"kill":"a real falsifier text","closure":"ruling-rescope"}]`), true},
		{"bad closure value", ax(`,"decision_state":"open","kill_condition":[{"kill":"a real falsifier text","closure":"vibes"}]`), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Inventory([]byte(tt.data))
			if tt.wantErr && err == nil {
				t.Fatalf("expected schema violation, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected valid, got error: %v", err)
			}
		})
	}
}

// anchorDoc builds a minimal valid inventory document with one anchor whose
// properties object is extended by extraAnchorJSON (a bare `,"k":v,...`
// fragment). Shared by the v0.3.6 (inter#238 landing, confluent-trust#132)
// test functions below.
func anchorDoc(extraAnchorJSON string) string {
	return `{"programme":"X","version":"0.1","schema_version":"v0.3","axioms":[],"chains":[],"anchors":[{` +
		`"id":"A-x","name":"x","tier":1,"provenance":"T","status":"coherent","description":"d","prediction_chain":[]` +
		extraAnchorJSON + `}]}`
}

func runValidateCases(t *testing.T, tests []struct {
	name    string
	data    string
	wantErr bool
}) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Inventory([]byte(tt.data))
			if tt.wantErr && err == nil {
				t.Fatalf("expected schema violation, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected valid, got error: %v", err)
			}
		})
	}
}

// TestEAxisSchema exercises the v0.3.6 e_axis field (inter#238 R1/R2;
// confluent-trust#132 AC1): post_hoc can never reach validated, and a
// reading with no stated sigma_pred is capped at consistent.
func TestEAxisSchema(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"consistent + post_hoc is valid (the boundary)", anchorDoc(`,"e_axis":{"value":"consistent","post_hoc":true}`), false},
		{"post_hoc + validated is rejected", anchorDoc(`,"e_axis":{"value":"validated","post_hoc":true,"sigma_pred":2.0}`), true},
		{"validated with sigma_pred stated is valid", anchorDoc(`,"e_axis":{"value":"validated","sigma_pred":2.0}`), false},
		{"validated with NO sigma_pred is rejected (no-uncertainty cap)", anchorDoc(`,"e_axis":{"value":"validated"}`), true},
		{"unconfronted with no sigma_pred is fine (cap only bites validated)", anchorDoc(`,"e_axis":{"value":"unconfronted"}`), false},
		{"falsified with no sigma_pred is fine (terminal, outside the order)", anchorDoc(`,"e_axis":{"value":"falsified"}`), false},
		{"bad value enum", anchorDoc(`,"e_axis":{"value":"proven"}`), true},
		{"missing required value", anchorDoc(`,"e_axis":{"post_hoc":false}`), true},
	}
	runValidateCases(t, tests)
}

// TestEvidenceRefExternalPrereg exercises the v0.3.6 evidence_ref[] field
// (inter#238 R3; confluent-trust#132 AC2): an external record claiming a
// pre-registration must point to a timestamped pre-reg artefact; an internal
// record needs no timestamp.
func TestEvidenceRefExternalPrereg(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			"internal record needs no timestamp (the boundary)",
			anchorDoc(`,"evidence_ref":[{"kind":"internal","value":1.0,"pre_reg_ref":"repo/path@sha"}]`),
			false,
		},
		{
			"external + pre_registered WITHOUT timestamp is rejected",
			anchorDoc(`,"evidence_ref":[{"kind":"external","pre_registered":true,"pre_reg_ref":"repo/path@sha"}]`),
			true,
		},
		{
			"external + pre_registered WITH ref and timestamp is valid",
			anchorDoc(`,"evidence_ref":[{"kind":"external","pre_registered":true,"pre_reg_ref":"repo/path@sha","pre_reg_timestamp":"2026-01-01T00:00:00Z"}]`),
			false,
		},
		{
			"external WITHOUT claiming pre-registration needs nothing extra",
			anchorDoc(`,"evidence_ref":[{"kind":"external","value":1.0,"publication_ref":"arXiv:2601.00000"}]`),
			false,
		},
		{"bad kind enum", anchorDoc(`,"evidence_ref":[{"kind":"both"}]`), true},
		{"missing required kind", anchorDoc(`,"evidence_ref":[{"value":1.0}]`), true},
	}
	runValidateCases(t, tests)
}

// TestTRLRecordShape exercises the v0.3.6 trl[] field (inter#238 R9';
// confluent-trust#132 AC5): environment is required except at the
// documentary TRL 1-2 stages, which have none to state.
func TestTRLRecordShape(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			"TRL 1 documentary record has no environment (the boundary)",
			anchorDoc(`,"trl":[{"trl_criterion":{"level":1,"text":"basic principles observed"},"satisfies":true,"fl_at_sha":"abc123"}]`),
			false,
		},
		{
			"TRL 2 documentary record has no environment",
			anchorDoc(`,"trl":[{"trl_criterion":{"level":2,"text":"concept formulated"},"satisfies":true,"fl_at_sha":"abc123"}]`),
			false,
		},
		{
			"TRL 4 WITHOUT environment is rejected",
			anchorDoc(`,"trl":[{"trl_criterion":{"level":4,"text":"component validated in lab"},"satisfies":true,"fl_at_sha":"abc123"}]`),
			true,
		},
		{
			"TRL 4 WITH environment is valid",
			anchorDoc(`,"trl":[{"environment":"lab","trl_criterion":{"level":4,"text":"component validated in lab"},"satisfies":true,"fl_at_sha":"abc123"}]`),
			false,
		},
		{"missing required trl_criterion", anchorDoc(`,"trl":[{"satisfies":true,"fl_at_sha":"abc123"}]`), true},
		{"bad environment enum", anchorDoc(`,"trl":[{"environment":"orbit","trl_criterion":{"level":9,"text":"flight proven"},"satisfies":true,"fl_at_sha":"abc123"}]`), true},
	}
	runValidateCases(t, tests)
}

// TestRemediationBothShas exercises the v0.3.6 remediation[] field
// (inter#238 R11.3'; confluent-trust#132 AC7): both the unfixed and fixed
// references are required, and soak_ref is required only when fault_state
// is not-reproduced.
func TestRemediationBothShas(t *testing.T) {
	base := `"unfixed_ref":"abc111","fixed_ref":"abc222","regression_test_ref":"TestFoo","scope_review_ref":"issuecomment-1"`
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{"both shas present, no fault_state: soak optional (the boundary)", anchorDoc(`,"remediation":[{` + base + `}]`), false},
		{"only unfixed_ref present is rejected (single sha)", anchorDoc(`,"remediation":[{"unfixed_ref":"abc111","regression_test_ref":"TestFoo","scope_review_ref":"issuecomment-1"}]`), true},
		{"only fixed_ref present is rejected (single sha)", anchorDoc(`,"remediation":[{"fixed_ref":"abc222","regression_test_ref":"TestFoo","scope_review_ref":"issuecomment-1"}]`), true},
		{"not-reproduced WITHOUT soak_ref is rejected", anchorDoc(`,"remediation":[{` + base + `,"fault_state":"not-reproduced"}]`), true},
		{"not-reproduced WITH soak_ref is valid", anchorDoc(`,"remediation":[{` + base + `,"fault_state":"not-reproduced","soak_ref":"boxrun-log-1"}]`), false},
		{"confirmed WITHOUT soak_ref is fine (cap only bites not-reproduced)", anchorDoc(`,"remediation":[{` + base + `,"fault_state":"confirmed"}]`), false},
		{"missing required scope_review_ref", anchorDoc(`,"remediation":[{"unfixed_ref":"abc111","fixed_ref":"abc222","regression_test_ref":"TestFoo"}]`), true},
	}
	runValidateCases(t, tests)
}
