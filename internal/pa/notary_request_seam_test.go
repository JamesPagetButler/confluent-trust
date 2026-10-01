package pa_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/JamesPagetButler/confluent-trust/internal/pa"
)

const notaryRequestSchemaURI = "https://github.com/JamesPagetButler/confluent-trust/testdata/pa/notary-request.schema.json"

func compileNotaryRequestSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pa", "notary-request.schema.json")) // #nosec G304 -- test reads a fixed-prefix testdata path
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(notaryRequestSchemaURI, doc); err != nil {
		t.Fatalf("register schema: %v", err)
	}
	s, err := c.Compile(notaryRequestSchemaURI)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return s
}

func validateInstance(t *testing.T, s *jsonschema.Schema, raw []byte) error {
	t.Helper()
	var inst any
	if err := json.Unmarshal(raw, &inst); err != nil {
		t.Fatalf("parse instance: %v", err)
	}
	return s.Validate(inst)
}

// TestNotaryRequestSeam_SampleValidates is the seam contract: every line of the
// shared sample jsonl validates against the canonical schema. notary#1 and the
// other emitters vendor this file byte-for-byte; a change on any side that breaks
// this fails the others' suites too.
func TestNotaryRequestSeam_SampleValidates(t *testing.T) {
	s := compileNotaryRequestSchema(t)
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pa", "notary-request.sample.jsonl")) // #nosec G304 -- test reads a fixed-prefix testdata path
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	n := 0
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		n++
		if err := validateInstance(t, s, line); err != nil {
			t.Errorf("sample line %d invalid: %v", n, err)
		}
	}
	if n < 3 {
		t.Errorf("expected at least 3 sample lines (one per reason), got %d", n)
	}
}

// TestNotaryRequestSeam_ConditionalBites proves the schema's reason-conditional is
// real: an artifact_tie_mismatch with an integer current_pa (should be null) and a
// pa_shortfall with a non-null artifact_digest (should be null) must both be
// rejected. Without the conditional these would pass and the seam would be hollow.
func TestNotaryRequestSeam_ConditionalBites(t *testing.T) {
	s := compileNotaryRequestSchema(t)
	bad := map[string]string{
		"tie with non-null current_pa": `{"schema_version":"1","ledger_version":"6.13.0","emitter":"qbp-cu-silicon-gate","emitted_at":"2026-10-01T00:00:00Z","reason":"artifact_tie_mismatch","claim_id":"PROOF-x","source_ref":"o/r@p","source_sha":"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0","artifact_digest":"sha256:ab","required_pa":2,"current_pa":0,"pinning_consumer":[],"stale":false}`,
		"pa_shortfall with digest":     `{"schema_version":"1","ledger_version":"6.13.0","emitter":"cth-pa-engine","emitted_at":"2026-10-01T00:00:00Z","reason":"pa_shortfall","claim_id":"PROOF-x","source_ref":"o/r@p","source_sha":"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0","artifact_digest":"sha256:ab","required_pa":2,"current_pa":0,"pinning_consumer":[],"stale":false}`,
		"unknown reason":               `{"schema_version":"1","ledger_version":"6.13.0","emitter":"cth-pa-engine","emitted_at":"2026-10-01T00:00:00Z","reason":"bogus","claim_id":"PROOF-x","source_ref":"o/r@p","source_sha":"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0","artifact_digest":null,"required_pa":2,"current_pa":0,"pinning_consumer":[],"stale":false}`,
		"short source_sha":             `{"schema_version":"1","ledger_version":"6.13.0","emitter":"cth-pa-engine","emitted_at":"2026-10-01T00:00:00Z","reason":"pa_shortfall","claim_id":"PROOF-x","source_ref":"o/r@p","source_sha":"abc","artifact_digest":null,"required_pa":2,"current_pa":0,"pinning_consumer":[],"stale":false}`,
	}
	for name, rec := range bad {
		if err := validateInstance(t, s, []byte(rec)); err == nil {
			t.Errorf("%s: schema accepted an invalid record", name)
		}
	}
}

// TestNotaryRequestSeam_DetailAccepted (R5): the optional detail field of locked
// v1.3 must validate — the schema was blocking it via additionalProperties:false.
func TestNotaryRequestSeam_DetailAccepted(t *testing.T) {
	s := compileNotaryRequestSchema(t)
	withDetail := `{"schema_version":"1","ledger_version":"6.13.0","emitter":"cth-pa-engine","emitted_at":"2026-10-01T00:00:00Z","reason":"pa_shortfall","claim_id":"PROOF-x","source_ref":"o/r@p","source_sha":"a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0","detail":"short human note","artifact_digest":null,"required_pa":2,"current_pa":0,"pinning_consumer":[],"stale":false}`
	if err := validateInstance(t, s, []byte(withDetail)); err != nil {
		t.Errorf("record with detail rejected: %v", err)
	}
}

// TestNotaryRequestSeam_EngineEmitsConforming proves the engine's own returned
// record conforms to the shared schema: a pa_shortfall and a staleness request,
// once stamped with emit metadata, both validate.
func TestNotaryRequestSeam_EngineEmitsConforming(t *testing.T) {
	s := compileNotaryRequestSchema(t)
	c := pa.Claim{ID: "PROOF-cd-structure-constant-tables"}
	meta := pa.RequestMeta{
		LedgerVersion:   "6.13.0",
		SourceRef:       "JamesPagetButler/QBP@proofs/QBP/Foundations/CDAlg.lean",
		SourceSHA:       "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
		EmittedAt:       "2026-10-01T00:00:00Z",
		PinningConsumer: []string{"roms/octonion_idx.hex"},
	}
	for _, tc := range []struct {
		name  string
		stale bool
	}{{"pa_shortfall", false}, {"staleness", true}} {
		res := pa.Result{ClaimID: c.ID}
		if tc.stale {
			res.Flags = []string{"lean4:" + pa.FlagStale}
		}
		r := pa.RequestFor(c, res, pa.PA0, pa.PA2, meta)
		if r == nil {
			t.Fatalf("%s: RequestFor returned nil", tc.name)
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("%s: marshal: %v", tc.name, err)
		}
		if err := validateInstance(t, s, raw); err != nil {
			t.Errorf("%s: engine-emitted record fails the shared schema: %v\n%s", tc.name, err, raw)
		}
	}
}
