package pa_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/attestation"
	"github.com/JamesPagetButler/confluent-trust/internal/pa"
)

// fxAssistant mirrors a fixture's proof_assistant entry. It embeds pa.Assistant
// (so the engine-graded fields unmarshal straight through) and adds the test-only
// attestation block, which the harness feeds to a fake verify() — never to the
// engine directly. That is the strongest form of derive-don't-declare: the engine
// has no field in which to read a producer-declared attestation; it must CALL
// verify(), and the harness decides what verify() returns.
type fxAssistant struct {
	Attestation struct {
		Signer string `json:"signer"`
		Role   string `json:"role"`
		Method string `json:"method"`
	} `json:"attestation"`
	pa.Assistant
}

// fxFile is one fixture on disk. expected_pa and the flags are TEST-ONLY oracles;
// they are never handed to the engine (pa.Claim carries no such fields, so the
// "harness strips expected_pa" rule is enforced structurally, not by convention).
type fxFile struct {
	TruePA          *int              `json:"true_pa_do_not_read"`
	Claim           string            `json:"claim"`
	Consumer        string            `json:"consumer"`
	PinnedSHA       string            `json:"pinned_sha"`
	Correspondence  pa.Correspondence `json:"correspondence"`
	ProofAssistants []fxAssistant     `json:"proof_assistants"`
	CorroboratedBy  []string          `json:"corroborated_by"`
	Flags           []string          `json:"flags"`
	ExpectedPA      int               `json:"expected_pa"`
	Policy          struct {
		RequireSignature bool `json:"require_signature"`
	} `json:"policy"`
}

func (f fxFile) claim() pa.Claim {
	as := make([]pa.Assistant, len(f.ProofAssistants))
	for i, a := range f.ProofAssistants {
		as[i] = a.Assistant
	}
	return pa.Claim{
		ID:             f.Claim,
		Consumer:       f.Consumer,
		PinnedSHA:      f.PinnedSHA,
		Assistants:     as,
		Correspondence: f.Correspondence,
		CorroboratedBy: f.CorroboratedBy,
	}
}

func (f fxFile) policy() pa.Policy { return pa.Policy{RequireSignature: f.Policy.RequireSignature} }

// fixtureVerifier is the test oracle for inter#149 verify(): it returns, per
// evidence_ref, the attestation Result the fixture says a real verifier would.
// verified is derived from the method (anything but unsigned/empty is verified),
// which covers every case the fixtures need without adding a field outside the
// converged format.
type fixtureVerifier struct{ byRef map[string]attestation.Result }

func (v fixtureVerifier) Verify(recordBytes []byte, _ string) attestation.Result {
	if r, ok := v.byRef[string(recordBytes)]; ok {
		return r
	}
	return attestation.Result{Signer: attestation.SignerUnknown, Role: attestation.RoleNone, Method: attestation.MethodUnsigned}
}

func (f fxFile) verifier() fixtureVerifier {
	m := make(map[string]attestation.Result, len(f.ProofAssistants))
	for _, a := range f.ProofAssistants {
		signer := a.Attestation.Signer
		if signer == "" {
			signer = attestation.SignerUnknown
		}
		m[a.EvidenceRef] = attestation.Result{
			Verified: a.Attestation.Method != "" && a.Attestation.Method != string(attestation.MethodUnsigned),
			Signer:   signer,
			Role:     attestation.Role(a.Attestation.Role),
			Method:   attestation.Method(a.Attestation.Method),
		}
	}
	return fixtureVerifier{byRef: m}
}

func loadFixture(t *testing.T, name string) fxFile {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pa", name)) // #nosec G304 -- test reads a fixed-prefix testdata path
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var f fxFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return f
}

// flagType strips the "<assistant>:" prefix an engine flag may carry, leaving the
// bare flag type the fixture lists (claim-level flags have no prefix).
func flagType(f string) string {
	if _, after, found := strings.Cut(f, ":"); found {
		return after
	}
	return f
}

func typeSet(flags []string) map[string]bool {
	s := map[string]bool{}
	for _, f := range flags {
		s[flagType(f)] = true
	}
	return s
}

func sortedKeys(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// TestFixtures runs every single-claim fixture through GradeClaim and asserts the
// grade and the exact set of flag types match the fixture's oracle. The chain
// edge file (not a claim) and fixture 12 (expected_pa deliberately wrong) have
// their own tests.
func TestFixtures(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "testdata", "pa"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || name == "10_edges.json" || name == "12_expected_pa_wrong.json" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			f := loadFixture(t, name)
			got := pa.GradeClaim(f.claim(), f.policy(), f.verifier())
			if int(got.PA) != f.ExpectedPA {
				t.Errorf("%s: PA = %d, want %d (flags %v)", name, got.PA, f.ExpectedPA, got.Flags)
			}
			want := typeSet(f.Flags)
			have := typeSet(got.Flags)
			for k := range want {
				if !have[k] {
					t.Errorf("%s: missing expected flag %q (got %v)", name, k, sortedKeys(have))
				}
			}
			for k := range have {
				if !want[k] {
					t.Errorf("%s: unexpected flag %q (want %v)", name, k, sortedKeys(want))
				}
			}
		})
	}
}

// TestEffectivePA_Chain (fixture 10, AC4): effective PA is the min over DERIVATION
// edges only. The head grades PA-2; its derivation dep grades PA-1; a relevance
// edge points at a PA-0 claim that must NOT lower the result.
func TestEffectivePA_Chain(t *testing.T) {
	head := loadFixture(t, "10a_chain_head_pa2.json")
	dep := loadFixture(t, "10b_chain_dep_pa1.json")
	rel := loadFixture(t, "10c_chain_relevance_pa0.json")

	grades := map[string]pa.Grade{}
	for _, f := range []fxFile{head, dep, rel} {
		grades[f.Claim] = pa.GradeClaim(f.claim(), f.policy(), f.verifier()).PA
	}
	if grades[head.Claim] != pa.PA2 || grades[dep.Claim] != pa.PA1 || grades[rel.Claim] != pa.PA0 {
		t.Fatalf("component grades wrong: %v", grades)
	}

	var edgeFile struct {
		Target   string    `json:"target"`
		Edges    []pa.Edge `json:"edges"`
		Expected int       `json:"expected_effective_pa"`
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pa", "10_edges.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &edgeFile); err != nil {
		t.Fatal(err)
	}

	deps := pa.DerivationDeps(edgeFile.Edges)
	got := pa.EffectivePA(edgeFile.Target, grades, deps)
	if int(got) != edgeFile.Expected {
		t.Errorf("EffectivePA(%s) = %d, want %d (deps %v)", edgeFile.Target, got, edgeFile.Expected, deps)
	}
	// The relevance edge's PA-0 target must have been excluded.
	if _, included := deps[edgeFile.Target]; included {
		for _, to := range deps[edgeFile.Target] {
			if to == rel.Claim {
				t.Errorf("relevance edge to %s leaked into derivation deps", rel.Claim)
			}
		}
	}
}

// TestEngineIgnoresExpectedPA (fixture 12): the fixture's expected_pa is planted
// wrong (2) while the true grade is 1. The engine must compute the true grade;
// it structurally cannot read expected_pa (pa.Claim has no such field).
func TestEngineIgnoresExpectedPA(t *testing.T) {
	f := loadFixture(t, "12_expected_pa_wrong.json")
	if f.TruePA == nil {
		t.Fatal("fixture 12 must carry true_pa_do_not_read")
	}
	got := pa.GradeClaim(f.claim(), f.policy(), f.verifier())
	if int(got.PA) != *f.TruePA {
		t.Errorf("PA = %d, want true grade %d", got.PA, *f.TruePA)
	}
	if int(got.PA) == f.ExpectedPA {
		t.Errorf("PA = %d equals the planted-wrong expected_pa; engine must not read it", got.PA)
	}
}

// TestTrustCheckVetoOnly (AC2/C2): trust_check is veto-only. A fail zeroes an
// otherwise-clean assistant; a pass grants nothing on its own.
func TestTrustCheckVetoOnly(t *testing.T) {
	clean := pa.Assistant{
		Assistant:   "lean4",
		EvidenceRef: "proofs/x.lean@sha#t",
		Producer:    "qbp-oppenheimer",
		Declared:    pa.Declared{Mode: "decide", Axioms: []string{"propext"}, OutputHash: "sha256:a"},
		Derived:     pa.Derived{TacticsUsed: []string{"decide"}, Axioms: []string{"propext"}, OutputHash: "sha256:a", KernelClean: true},
		TrustCheck:  "fail",
	}
	c := pa.Claim{ID: "PROOF-veto", Assistants: []pa.Assistant{clean}}
	got := pa.GradeClaim(c, pa.Policy{}, attestation.StubV0{})
	if got.PA != pa.PA0 {
		t.Errorf("trust_check:fail did not veto: PA = %d", got.PA)
	}
	foundVeto := false
	for _, f := range got.Flags {
		if flagType(f) == "trust_check_fail" {
			foundVeto = true
		}
	}
	if !foundVeto {
		t.Errorf("missing trust_check_fail flag, got %v", got.Flags)
	}
}

// TestRequestFor covers the notary#3 v1.3 emit shaping: below required ⇒ a
// pa_shortfall record; met ⇒ nil; a moved sha (stale) ⇒ a staleness record even
// when the grade meets required. The engine returns the record; the caller emits.
func TestRequestFor(t *testing.T) {
	c := pa.Claim{ID: "PROOF-cd-structure-constant-tables"}
	meta := pa.RequestMeta{
		LedgerVersion:   "6.13.0",
		SourceRef:       "JamesPagetButler/QBP@proofs/QBP/Foundations/CDAlg.lean",
		SourceSHA:       "0123456789abcdef0123456789abcdef01234567",
		EmittedAt:       "2026-10-01T00:00:00Z",
		PinningConsumer: []string{"roms/octonion_idx.hex", "roms/octonion_signs.hex"},
	}

	clean := pa.Result{ClaimID: c.ID}
	staleRes := pa.Result{ClaimID: c.ID, Flags: []string{"lean4:" + pa.FlagStale}}

	if r := pa.RequestFor(c, clean, pa.PA1, pa.PA2, meta); r == nil {
		t.Error("below required: want a request, got nil")
	} else {
		if r.Reason != pa.ReasonPAShortfall {
			t.Errorf("reason = %q, want pa_shortfall", r.Reason)
		}
		if r.ClaimID != c.ID || r.Emitter != pa.EmitterCTH || r.SchemaVersion != pa.SchemaVersion {
			t.Errorf("identity wrong: %+v", r)
		}
		if r.CurrentPA == nil || *r.CurrentPA != pa.PA1 || r.RequiredPA != pa.PA2 || r.Stale {
			t.Errorf("fields wrong: %+v", r)
		}
		if r.ArtifactDigest != nil {
			t.Errorf("artifact_digest must be nil for pa_shortfall: %+v", r)
		}
		if len(r.PinningConsumer) != 2 {
			t.Errorf("pinning_consumer not carried as a list: %+v", r.PinningConsumer)
		}
	}

	if r := pa.RequestFor(c, clean, pa.PA2, pa.PA2, meta); r != nil {
		t.Errorf("met required, not stale: want nil, got %+v", r)
	}

	// Staleness is derived from the grade's flags, not a separate argument: a
	// stale grade that still meets required PA must still emit a staleness request.
	if r := pa.RequestFor(c, staleRes, pa.PA2, pa.PA2, meta); r == nil {
		t.Error("stale grade: want a request even when effective PA meets required, got nil")
	} else if r.Reason != pa.ReasonStaleness || !r.Stale {
		t.Errorf("stale request must be reason=staleness + Stale: %+v", r)
	}
}

// TestDedupeKey (notary#3 R1): the key is (claim_id, source_sha, reason,
// artifact_digest). Same claim+sha but different reasons ⇒ different keys (so a
// pa_shortfall and an artifact_tie_mismatch don't merge); identical ⇒ same key.
func TestDedupeKey(t *testing.T) {
	dig := "sha256:beef"
	base := pa.NotaryRequest{ClaimID: "PROOF-x", SourceSHA: "sha1", Reason: pa.ReasonPAShortfall}
	same := base
	tie := pa.NotaryRequest{ClaimID: "PROOF-x", SourceSHA: "sha1", Reason: pa.ReasonArtifactTieMismatch, ArtifactDigest: &dig}

	if base.DedupeKey() != same.DedupeKey() {
		t.Errorf("identical records must share a key: %q vs %q", base.DedupeKey(), same.DedupeKey())
	}
	if base.DedupeKey() == tie.DedupeKey() {
		t.Errorf("different reason/digest must not merge: %q == %q", base.DedupeKey(), tie.DedupeKey())
	}
}

// TestStaleWhilePassing (notary#3 staleness seam): a claim whose assistants were
// produced against a sha that no longer matches the claim's pinned sha grades
// below its old level — a changed proof cannot keep an old pass.
func TestStaleWhilePassing(t *testing.T) {
	f := loadFixture(t, "15_stale_source.json")
	got := pa.GradeClaim(f.claim(), f.policy(), f.verifier())
	if int(got.PA) != f.ExpectedPA {
		t.Errorf("stale fixture: PA = %d, want %d (flags %v)", got.PA, f.ExpectedPA, got.Flags)
	}
	stale := false
	for _, fl := range got.Flags {
		if flagType(fl) == "stale" {
			stale = true
		}
	}
	if !stale {
		t.Errorf("stale fixture: missing stale flag, got %v", got.Flags)
	}
}
