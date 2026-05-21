package store

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JamesPagetButler/confluent-trust/compute"
	"github.com/JamesPagetButler/confluent-trust/model"
)

// ---- test fixture helpers ----

// qbpQuantumFixturePath is the relative path to the QBP-Q quantum fixture.
const qbpQuantumFixturePath = "../testdata/qbp_quantum_v0_2.json"

// liveTestProofAnchorID and liveTestUpdatedNotes are repeated across tests;
// hoisted to satisfy CI goconst.
const (
	liveTestProofAnchorID = "PROOF-new-1"
	liveTestUpdatedNotes  = "updated notes"
)

// minimalInventoryJSON is a small hand-written inventory for tests that
// need a clean, write-to-disk fixture. It is a subset of minimal.json
// with exactly one anchor, one chain, and one input.
const minimalInventoryJSON = `{
  "schema_version": "v0.2",
  "programme": "TEST",
  "version": "0.1",
  "axioms": [
    {
      "id": "AXIOM-1",
      "name": "Test axiom",
      "statement": "Axiom for testing.",
      "derivable": false
    }
  ],
  "anchors": [
    {
      "id": "PROOF-seed",
      "name": "Seed anchor",
      "tier": 1,
      "provenance": "T",
      "status": "coherent",
      "description": "Seed anchor for live tests.",
      "prediction_chain": ["AXIOM-1"]
    }
  ],
  "inputs": [],
  "chains": [
    {
      "id": "CHAIN-seed",
      "name": "Seed chain",
      "source_ids": ["AXIOM-1"],
      "target_id": "PROOF-seed",
      "steps": 1,
      "status": "coherent"
    }
  ],
  "confluence_points": []
}`

// openTempLive writes minimalInventoryJSON to a temp file and opens a
// LiveInventory from it. The caller must Close() the returned handle.
func openTempLive(t *testing.T, hooks *Hooks) (*LiveInventory, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	if err := writeRaw(path, []byte(minimalInventoryJSON)); err != nil {
		t.Fatalf("openTempLive: write fixture: %v", err)
	}
	li, err := OpenLiveInventory(path, hooks)
	if err != nil {
		t.Fatalf("openTempLive: open: %v", err)
	}
	return li, path
}

// freshAnchor returns a valid Anchor with the given ID that is safe to append
// to the minimal TEST inventory.
func freshAnchor(id string) model.Anchor {
	return model.Anchor{
		ID:              id,
		Name:            "Fresh anchor " + id,
		Description:     "Test anchor for " + id,
		Tier:            model.TierProof,
		Provenance:      model.ProvenanceTheoretical,
		Status:          model.StatusUntested,
		PredictionChain: []string{"AXIOM-1"},
	}
}

// freshChain returns a valid Chain with the given ID referencing existing records.
func freshChain(id string) model.Chain {
	return model.Chain{
		ID:        id,
		Name:      "Fresh chain " + id,
		SourceIDs: []string{"AXIOM-1"},
		TargetID:  "PROOF-seed",
		Steps:     1,
		Status:    model.StatusCoherent,
	}
}

// freshConfluence returns a valid ConfluencePoint with the given ID.
func freshConfluence(id string) model.ConfluencePoint {
	return model.ConfluencePoint{
		ID:       id,
		AnchorID: "PROOF-seed",
		Status:   model.StatusCoherent,
	}
}

// freshInput returns a valid Input with the given ID.
func freshInput(id string) model.Input {
	return model.Input{
		ID:   id,
		Name: "Input " + id,
		Type: "input",
	}
}

// ---- tests ----

// TestOpenLiveInventory_LoadsExistingFixture verifies that an existing on-disk
// fixture loads correctly and that nil hooks are accepted.
func TestOpenLiveInventory_LoadsExistingFixture(t *testing.T) {
	li, err := OpenLiveInventory(qbpQuantumFixturePath, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer li.Close() //nolint:errcheck // test defer, error not meaningful

	snap := li.Snapshot()
	if snap.Programme != "QBP-Q" {
		t.Errorf("programme: got %q want %q", snap.Programme, "QBP-Q")
	}
}

// TestAppendAnchor_HappyPath verifies the full happy-path: Snapshot reflects
// the addition; disk reflects the addition; Health is nil; OnAnchorChange
// fires with before==nil and the correct after.ID.
func TestAppendAnchor_HappyPath(t *testing.T) {
	hookFired := false
	var hookBefore, hookAfter *model.Anchor

	hooks := &Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			hookFired = true
			hookBefore = before
			hookAfter = after
		},
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	anchor := freshAnchor(liveTestProofAnchorID)
	if err := li.AppendAnchor(anchor); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}

	// (a) Snapshot reflects addition.
	snap := li.Snapshot()
	found := false
	for _, a := range snap.Anchors {
		if a.ID == liveTestProofAnchorID {
			found = true
		}
	}
	if !found {
		t.Error("Snapshot does not contain the appended anchor")
	}

	// (b) Disk reflects addition.
	reloaded, err := LoadInventory(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	diskFound := false
	for _, a := range reloaded.Anchors {
		if a.ID == liveTestProofAnchorID {
			diskFound = true
		}
	}
	if !diskFound {
		t.Error("reloaded inventory does not contain the appended anchor")
	}

	// (c) Health is nil.
	if snap.Health != nil {
		t.Error("Health should be nil after mutation")
	}

	// (d) Hook fired with before==nil + correct after.ID.
	if !hookFired {
		t.Error("OnAnchorChange did not fire")
	}
	if hookBefore != nil {
		t.Errorf("hook before: got %v want nil", hookBefore)
	}
	if hookAfter == nil || hookAfter.ID != liveTestProofAnchorID {
		t.Errorf("hook after.ID: got %v want PROOF-new-1", hookAfter)
	}
}

// TestAppendAnchor_IDCollision verifies that appending a duplicate ID returns
// wrapped model.ErrIDCollision and does not mutate the file or fire the hook.
func TestAppendAnchor_IDCollision(t *testing.T) {
	hookFiredCount := 0
	hooks := &Hooks{
		OnAnchorChange: func(_, _ *model.Anchor) { hookFiredCount++ },
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	a := freshAnchor("PROOF-dup")
	if err := li.AppendAnchor(a); err != nil {
		t.Fatalf("first append: %v", err)
	}
	hookFiredCount = 0 // reset after the first successful append

	// Second append with same ID.
	err := li.AppendAnchor(a)
	if err == nil {
		t.Fatal("expected error on duplicate ID")
	}
	if !errors.Is(err, model.ErrIDCollision) {
		t.Errorf("expected ErrIDCollision, got %v", err)
	}

	// File on disk unchanged (anchor count == count after first append).
	reloaded, reloadErr := LoadInventory(path)
	if reloadErr != nil {
		t.Fatalf("reload: %v", reloadErr)
	}
	countAfterFirst := 2 // seed + dup
	if got := len(reloaded.Anchors); got != countAfterFirst {
		t.Errorf("reloaded anchor count: got %d want %d", got, countAfterFirst)
	}

	// Hook did not fire for the failed call.
	if hookFiredCount != 0 {
		t.Errorf("hook fired %d times for failed append, want 0", hookFiredCount)
	}
}

// TestAppendAnchor_InvalidRecord verifies that an invalid anchor (empty ID)
// is rejected: error returned; file unchanged; hook does not fire; Snapshot
// unchanged.
func TestAppendAnchor_InvalidRecord(t *testing.T) {
	hookFiredCount := 0
	hooks := &Hooks{
		OnAnchorChange: func(_, _ *model.Anchor) { hookFiredCount++ },
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	snapBefore := li.Snapshot()

	// Empty ID fails Anchor.Validate().
	bad := model.Anchor{
		ID:          "",
		Name:        "bad anchor",
		Description: "bad",
		Tier:        model.TierProof,
		Provenance:  model.ProvenanceTheoretical,
	}
	if err := li.AppendAnchor(bad); err == nil {
		t.Fatal("expected error for invalid anchor")
	}

	// Snapshot unchanged.
	snapAfter := li.Snapshot()
	if len(snapAfter.Anchors) != len(snapBefore.Anchors) {
		t.Errorf("anchor count changed: before %d after %d",
			len(snapBefore.Anchors), len(snapAfter.Anchors))
	}

	// Disk unchanged.
	reloaded, err := LoadInventory(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Anchors) != len(snapBefore.Anchors) {
		t.Errorf("disk anchor count changed after failed append")
	}

	// Hook did not fire.
	if hookFiredCount != 0 {
		t.Errorf("hook fired %d times, want 0", hookFiredCount)
	}
}

// TestAppendInput_NoHookFires verifies that AppendInput lands in Snapshot and
// that none of the three hooks fire.
func TestAppendInput_NoHookFires(t *testing.T) {
	anchorCount, chainCount, confluenceCount := 0, 0, 0
	hooks := &Hooks{
		OnAnchorChange:     func(_, _ *model.Anchor) { anchorCount++ },
		OnChainChange:      func(_, _ *model.Chain) { chainCount++ },
		OnConfluenceChange: func(_, _ *model.ConfluencePoint) { confluenceCount++ },
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	in := freshInput("INPUT-test-1")
	if err := li.AppendInput(in); err != nil {
		t.Fatalf("AppendInput: %v", err)
	}

	// Input is in Snapshot.
	snap := li.Snapshot()
	found := false
	for _, i := range snap.Inputs {
		if i.ID == "INPUT-test-1" {
			found = true
		}
	}
	if !found {
		t.Error("Snapshot does not contain the appended input")
	}

	// None of the three hooks fired.
	if anchorCount != 0 {
		t.Errorf("OnAnchorChange fired %d times, want 0", anchorCount)
	}
	if chainCount != 0 {
		t.Errorf("OnChainChange fired %d times, want 0", chainCount)
	}
	if confluenceCount != 0 {
		t.Errorf("OnConfluenceChange fired %d times, want 0", confluenceCount)
	}
}

// TestUpdateAnchor_HappyPath_StatusChange verifies that a status change fires
// OnAnchorChange with the correct before/after snapshot.
func TestUpdateAnchor_HappyPath_StatusChange(t *testing.T) {
	var capturedBefore, capturedAfter *model.Anchor
	hookCount := 0
	hooks := &Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			hookCount++
			capturedBefore = before
			capturedAfter = after
		},
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	// Append an untested anchor.
	a := freshAnchor("PROOF-status-test")
	a.Status = model.StatusUntested
	if err := li.AppendAnchor(a); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}
	hookCount = 0 // reset after append hook

	// Update status to coherent.
	err := li.UpdateAnchor("PROOF-status-test", func(a *model.Anchor) error {
		a.Status = model.StatusCoherent
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateAnchor: %v", err)
	}

	// (a) Snapshot reflects new status.
	snap := li.Snapshot()
	for _, a := range snap.Anchors {
		if a.ID == "PROOF-status-test" {
			if a.Status != model.StatusCoherent {
				t.Errorf("status: got %v want coherent", a.Status)
			}
		}
	}

	// (b) Hook fired with correct before/after.
	if hookCount != 1 {
		t.Errorf("hook fired %d times, want 1", hookCount)
	}
	if capturedBefore == nil || capturedBefore.Status != model.StatusUntested {
		t.Errorf("before.Status: got %v want untested", capturedBefore)
	}
	if capturedAfter == nil || capturedAfter.Status != model.StatusCoherent {
		t.Errorf("after.Status: got %v want coherent", capturedAfter)
	}
}

// TestUpdateAnchor_HookFiltering_NotesChangeDoesNotFire verifies that updating
// only the Notes field does NOT fire OnAnchorChange (whitelist filter).
func TestUpdateAnchor_HookFiltering_NotesChangeDoesNotFire(t *testing.T) {
	// Track hooks: the append hook fires once (before==nil); update should not fire.
	hookCalls := 0
	var updateHookCalls int // separate counter reset after append
	hooks := &Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			hookCalls++
			if before != nil {
				// This is an update hook (append fires with before==nil).
				updateHookCalls++
			}
		},
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	a := freshAnchor("PROOF-notes-test")
	if err := li.AppendAnchor(a); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}

	// Update only Notes.
	err := li.UpdateAnchor("PROOF-notes-test", func(a *model.Anchor) error {
		a.Notes = liveTestUpdatedNotes
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateAnchor: %v", err)
	}

	// (a) Snapshot reflects the new Notes.
	snap := li.Snapshot()
	for _, a := range snap.Anchors {
		if a.ID == "PROOF-notes-test" {
			if a.Notes != liveTestUpdatedNotes {
				t.Errorf("Notes: got %q want %q", a.Notes, liveTestUpdatedNotes)
			}
		}
	}

	// (b) No update hook fired (only the append hook).
	if updateHookCalls != 0 {
		t.Errorf("OnAnchorChange fired %d update calls, want 0", updateHookCalls)
	}
	if hookCalls != 1 {
		t.Errorf("total hook calls: got %d want 1 (append only)", hookCalls)
	}
}

// TestUpdateAnchor_MutatorError_RollsBack verifies that when the mutator
// returns an error, the state is rolled back and no hook fires.
func TestUpdateAnchor_MutatorError_RollsBack(t *testing.T) {
	hookCount := 0
	hooks := &Hooks{
		OnAnchorChange: func(_, _ *model.Anchor) { hookCount++ },
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	a := freshAnchor("PROOF-rollback-test")
	if err := li.AppendAnchor(a); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}
	hookCount = 0

	snapBefore := li.Snapshot()

	err := li.UpdateAnchor("PROOF-rollback-test", func(a *model.Anchor) error {
		a.Status = model.StatusCoherent
		return fmt.Errorf("simulated mutator error")
	})
	if err == nil {
		t.Fatal("expected error from mutator")
	}

	// (a) In-memory state unchanged.
	snapAfter := li.Snapshot()
	for _, a := range snapAfter.Anchors {
		if a.ID == "PROOF-rollback-test" {
			if a.Status != snapBefore.Anchors[len(snapBefore.Anchors)-1].Status {
				t.Error("status changed despite mutator error — rollback failed")
			}
		}
	}

	// (b) Disk unchanged.
	reloaded, reloadErr := LoadInventory(path)
	if reloadErr != nil {
		t.Fatalf("reload: %v", reloadErr)
	}
	for _, a := range reloaded.Anchors {
		if a.ID == "PROOF-rollback-test" && a.Status == model.StatusCoherent {
			t.Error("disk reflects mutated state despite mutator error")
		}
	}

	// (c) Hook did not fire.
	if hookCount != 0 {
		t.Errorf("hook fired %d times, want 0", hookCount)
	}
}

// TestUpdateAnchor_ValidationFailure_RollsBack verifies that a mutation that
// passes the mutator but fails Inventory.Validate() is rolled back.
func TestUpdateAnchor_ValidationFailure_RollsBack(t *testing.T) {
	hookCount := 0
	hooks := &Hooks{
		OnAnchorChange: func(_, _ *model.Anchor) { hookCount++ },
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	a := freshAnchor("PROOF-valroll-test")
	if err := li.AppendAnchor(a); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}
	hookCount = 0

	// Zeroing the ID will cause anchor.Validate() to fail inside inv.Validate().
	err := li.UpdateAnchor("PROOF-valroll-test", func(a *model.Anchor) error {
		a.ID = "" // invalid — triggers "anchor: empty id"
		return nil
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	// In-memory state unchanged — the anchor still has its original ID.
	snap := li.Snapshot()
	found := false
	for _, a := range snap.Anchors {
		if a.ID == "PROOF-valroll-test" {
			found = true
		}
	}
	if !found {
		t.Error("anchor lost from Snapshot after rollback (rollback overwrote wrong record)")
	}

	// Disk unchanged.
	reloaded, reloadErr := LoadInventory(path)
	if reloadErr != nil {
		t.Fatalf("reload: %v", reloadErr)
	}
	diskFound := false
	for _, a := range reloaded.Anchors {
		if a.ID == "PROOF-valroll-test" {
			diskFound = true
		}
	}
	if !diskFound {
		t.Error("disk does not contain original anchor after rollback")
	}

	// Hook did not fire.
	if hookCount != 0 {
		t.Errorf("hook fired %d times after validation failure, want 0", hookCount)
	}
}

// TestUpdateAnchor_NotFound verifies that a non-existent ID returns wrapped
// model.ErrNotFound.
func TestUpdateAnchor_NotFound(t *testing.T) {
	li, _ := openTempLive(t, nil)
	defer li.Close() //nolint:errcheck // test defer

	err := li.UpdateAnchor("DOES-NOT-EXIST", func(_ *model.Anchor) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestUpdateChain_AllFieldChangesFire verifies that:
//   - AppendChain fires OnChainChange with before==nil.
//   - UpdateChain fires OnChainChange for a Notes-only change (no whitelist).
//   - AppendConfluence fires OnConfluenceChange with before==nil.
func TestUpdateChain_AllFieldChangesFire(t *testing.T) {
	chainHookCount := 0
	var capturedChainBefore, capturedChainAfter *model.Chain
	confluenceHookCount := 0
	hooks := &Hooks{
		OnChainChange: func(before, after *model.Chain) {
			chainHookCount++
			capturedChainBefore = before
			capturedChainAfter = after
		},
		OnConfluenceChange: func(_, _ *model.ConfluencePoint) {
			confluenceHookCount++
		},
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	// AppendChain fires OnChainChange (before==nil).
	newChain := freshChain("CHAIN-new-1")
	if err := li.AppendChain(newChain); err != nil {
		t.Fatalf("AppendChain: %v", err)
	}
	if chainHookCount != 1 {
		t.Errorf("OnChainChange fires on AppendChain: got %d want 1", chainHookCount)
	}
	if capturedChainBefore != nil {
		t.Errorf("OnChainChange before on append: got %v want nil", capturedChainBefore)
	}
	chainHookCount = 0

	// AppendConfluence fires OnConfluenceChange (before==nil).
	newConfl := freshConfluence("CONFL-new-1")
	if err := li.AppendConfluence(newConfl); err != nil {
		t.Fatalf("AppendConfluence: %v", err)
	}
	if confluenceHookCount != 1 {
		t.Errorf("OnConfluenceChange fires on AppendConfluence: got %d want 1", confluenceHookCount)
	}

	// UpdateChain fires for a Notes-only change (no whitelist on chain hooks).
	err := li.UpdateChain("CHAIN-seed", func(c *model.Chain) error {
		c.Notes = liveTestUpdatedNotes
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateChain: %v", err)
	}

	if chainHookCount != 1 {
		t.Errorf("OnChainChange fired %d times on update, want 1", chainHookCount)
	}
	if capturedChainBefore == nil || capturedChainBefore.Notes != "" {
		t.Errorf("before.Notes: got %v want empty string", capturedChainBefore)
	}
	if capturedChainAfter == nil || capturedChainAfter.Notes != liveTestUpdatedNotes {
		t.Errorf("after.Notes: got %v want 'updated notes'", capturedChainAfter)
	}
}

// TestSnapshot_DeepCopy verifies that mutations to the returned snapshot do
// not affect the LiveInventory's internal state.
func TestSnapshot_DeepCopy(t *testing.T) {
	li, _ := openTempLive(t, nil)
	defer li.Close() //nolint:errcheck // test defer

	snap1 := li.Snapshot()

	// Mutate the snapshot: change an anchor field and append to a slice.
	if len(snap1.Anchors) == 0 {
		t.Fatal("fixture has no anchors to mutate")
	}
	snap1.Anchors[0].Notes = "mutated in snapshot"
	snap1.Anchors[0].PredictionChain = append(snap1.Anchors[0].PredictionChain, "EXTRA")
	snap1.Anchors = append(snap1.Anchors, freshAnchor("PROOF-extra-in-snap"))

	// Re-snapshot from the live inventory.
	snap2 := li.Snapshot()

	// Original state is unaffected.
	if snap2.Anchors[0].Notes == "mutated in snapshot" {
		t.Error("Notes mutation leaked from snapshot into LiveInventory")
	}
	for _, a := range snap2.Anchors {
		if a.ID == "PROOF-extra-in-snap" {
			t.Error("extra anchor appended to snapshot leaked into LiveInventory")
		}
	}
	for _, s := range snap2.Anchors[0].PredictionChain {
		if s == "EXTRA" {
			t.Error("PredictionChain append leaked from snapshot into LiveInventory")
		}
	}
}

// TestClose_BlocksFurtherCalls verifies Close semantics: after Close, all
// methods return wrapped ErrClosed; double-Close returns nil.
func TestClose_BlocksFurtherCalls(t *testing.T) {
	li, _ := openTempLive(t, nil)

	if err := li.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// AppendAnchor returns ErrClosed.
	err := li.AppendAnchor(freshAnchor("PROOF-after-close"))
	if !errors.Is(err, model.ErrClosed) {
		t.Errorf("AppendAnchor after Close: expected ErrClosed, got %v", err)
	}

	// UpdateAnchor returns ErrClosed.
	err = li.UpdateAnchor("PROOF-seed", func(_ *model.Anchor) error { return nil })
	if !errors.Is(err, model.ErrClosed) {
		t.Errorf("UpdateAnchor after Close: expected ErrClosed, got %v", err)
	}

	// Snapshot after Close: the design doc says all method calls return ErrClosed
	// wrapped; Snapshot returns a model.Inventory (no error). We verify closed
	// state by checking AppendAnchor returns ErrClosed (already done above).
	// Per the design, Snapshot itself does not return an error so we can only
	// test the mutating methods.

	// Double-Close returns nil.
	if err := li.Close(); err != nil {
		t.Errorf("double-Close: expected nil, got %v", err)
	}
}

// TestConcurrent_AppendsRaceClean spawns 50 goroutines each appending one
// unique-ID anchor, verifies all 50 are present on disk and in Snapshot,
// and that the hook fired exactly 50 times.
func TestConcurrent_AppendsRaceClean(t *testing.T) {
	const n = 50
	var hookCount atomic.Int64

	hooks := &Hooks{
		OnAnchorChange: func(_, _ *model.Anchor) {
			hookCount.Add(1)
		},
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		id := fmt.Sprintf("PROOF-concurrent-%03d", i)
		go func(anchorID string) {
			defer wg.Done()
			if err := li.AppendAnchor(freshAnchor(anchorID)); err != nil {
				t.Errorf("AppendAnchor %s: %v", anchorID, err)
			}
		}(id)
	}
	wg.Wait()

	// (a) All 50 are in Snapshot (plus the seed anchor = 51 total).
	snap := li.Snapshot()
	if got := len(snap.Anchors); got != n+1 {
		t.Errorf("Snapshot anchor count: got %d want %d", got, n+1)
	}

	// (b) Disk reflects all 50.
	reloaded, err := LoadInventory(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := len(reloaded.Anchors); got != n+1 {
		t.Errorf("disk anchor count: got %d want %d", got, n+1)
	}

	// (c) Hook fired exactly n times.
	if got := hookCount.Load(); got != n {
		t.Errorf("hook fired %d times, want %d", got, n)
	}
}

// TestConcurrent_AppendAndUpdate spawns 25 goroutines appending new anchors
// and 25 goroutines updating the seed anchor's Notes. Runs under -race.
// Verifies: no data race; hooks count is correct; final state consistent.
func TestConcurrent_AppendAndUpdate(t *testing.T) {
	const (
		appendN = 25
		updateN = 25
	)
	var anchorHookCount atomic.Int64

	hooks := &Hooks{
		OnAnchorChange: func(_, _ *model.Anchor) {
			anchorHookCount.Add(1)
		},
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	var wg sync.WaitGroup

	// 25 appending goroutines.
	wg.Add(appendN)
	for i := range appendN {
		id := fmt.Sprintf("PROOF-append-%03d", i)
		go func(anchorID string) {
			defer wg.Done()
			if err := li.AppendAnchor(freshAnchor(anchorID)); err != nil {
				t.Errorf("AppendAnchor %s: %v", anchorID, err)
			}
		}(id)
	}

	// 25 updating goroutines — alternate between untested and coherent to
	// trigger the whitelist fields changed check.
	wg.Add(updateN)
	for i := range updateN {
		target := model.StatusCoherent
		if i%2 == 0 {
			target = model.StatusUntested
		}
		go func(newStatus model.Status) {
			defer wg.Done()
			// Ignore not-found / other errors — concurrent appends may not have
			// landed the anchor yet; seed anchor always exists.
			_ = li.UpdateAnchor("PROOF-seed", func(a *model.Anchor) error {
				a.Status = newStatus
				return nil
			})
		}(target)
	}

	wg.Wait()

	// Final state: at least appendN+1 anchors (seed + all appended).
	snap := li.Snapshot()
	if got := len(snap.Anchors); got < appendN+1 {
		t.Errorf("Snapshot anchor count: got %d want >= %d", got, appendN+1)
	}

	// Hook count: appendN fires (before==nil) + however many status changes
	// fired for the seed anchor updates. We can only check a lower bound.
	if got := anchorHookCount.Load(); got < appendN {
		t.Errorf("hook count: got %d want >= %d (append fires)", got, appendN)
	}
}

// ---- PRED-* → Tier 3→2 transition invariant tests ----
// Sprint-1 Notary-bootstrap target #3 (sprint-1-closeout-2026-05-17 seq=12).
// See doc/design/live-inventory-api.md §2.2 for the documented contract.
//
// These tests operate against the main-branch whitelist:
//   {Status, MeasuredValue, MeasuredError, DiscrepancyPct, LastTestedAt}
// per doc/design/live-inventory-api.md §2.1.

// predTestAnchorID is the canonical PRED-* anchor ID used across the
// transition tests; hoisted to satisfy CI goconst.
// predLastTestedAt is the RFC3339 timestamp used in the transition tests.
const (
	predTestAnchorID = "PRED-test-01"
	predLastTestedAt = "2026-05-21T00:00:00Z"
)

// freshPredictionAnchor returns a valid TierPrediction anchor with the given
// ID, ready to be appended to the minimal TEST inventory.
func freshPredictionAnchor(id string) model.Anchor {
	predicted := 42.0
	return model.Anchor{
		ID:              id,
		Name:            "Prediction anchor " + id,
		Description:     "Test prediction anchor for " + id,
		Tier:            model.TierPrediction,
		Provenance:      model.ProvenanceTheoretical,
		Status:          model.StatusUntested,
		PredictedValue:  &predicted,
		PredictionChain: []string{"AXIOM-1"},
	}
}

// regimeToStatus maps a compute.ScoreRegime to the expected model.Status per
// the transition contract in doc/design/live-inventory-api.md §2.2.
// Laminar and LowSediment both map to Coherent; Moderate → Contested; Heavy → Refuted.
func regimeToStatus(regime compute.ScoreRegime) model.Status {
	switch regime {
	case compute.ScoreRegimeLaminar, compute.ScoreRegimeLowSediment:
		return model.StatusCoherent
	case compute.ScoreRegimeModerate:
		return model.StatusContested
	default: // ScoreRegimeHeavy
		return model.StatusRefuted
	}
}

// TestLiveInventory_PredictionToMeasurementTransition_HappyPath verifies the
// full end-to-end Tier 3→2 transition via UpdateAnchor:
//
//   - Tier changes from TierPrediction (3) to TierMeasurement (2)
//   - MeasuredValue, MeasuredError, MeasuredSource, LastTestedAt are populated
//   - DiscrepancyPct is computed via compute.ScorePrediction and stored
//   - Status transitions from Untested → Coherent (laminar regime: delta ~0.71%)
//   - OnAnchorChange fires (MeasuredValue + Status + DiscrepancyPct + LastTestedAt
//     all in the whitelist; before.Tier==3 / after.Tier==2 observable)
//   - Disk (reload via LoadInventory) reflects all 6 field changes
//
// Contract per doc/design/live-inventory-api.md §2.2.
func TestLiveInventory_PredictionToMeasurementTransition_HappyPath(t *testing.T) {
	hookCount := 0
	var capturedBefore, capturedAfter *model.Anchor

	hooks := &Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			hookCount++
			capturedBefore = before
			capturedAfter = after
		},
	}

	li, path := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	// --- Append phase ---
	pred := freshPredictionAnchor(predTestAnchorID)
	if err := li.AppendAnchor(pred); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}

	// AppendAnchor fires OnAnchorChange with before==nil (§2.1 contract).
	if hookCount != 1 {
		t.Errorf("hook count after append: got %d want 1", hookCount)
	}
	if capturedBefore != nil {
		t.Errorf("hook before on append: got %v want nil", capturedBefore)
	}
	if capturedAfter == nil || capturedAfter.Tier != model.TierPrediction {
		t.Errorf("hook after.Tier on append: got %v want TierPrediction", capturedAfter)
	}
	hookCount = 0 // reset for transition assertion

	// --- Transition phase (the load-bearing assertion) ---
	predicted := 42.0
	observed := 41.7
	score, err := compute.ScorePrediction(compute.KindScalar, predicted, observed)
	if err != nil {
		t.Fatalf("ScorePrediction: %v", err)
	}

	measuredErr := 0.1
	measuredSrc := "test"
	lastTested := predLastTestedAt
	wantDiscrepancyPct := score.DiscrepancyPct
	wantStatus := regimeToStatus(score.Regime)

	updateErr := li.UpdateAnchor(predTestAnchorID, func(a *model.Anchor) error {
		a.Tier = model.TierMeasurement
		a.MeasuredValue = &observed
		a.MeasuredError = &measuredErr
		a.MeasuredSource = measuredSrc
		a.LastTestedAt = &lastTested
		a.DiscrepancyPct = &wantDiscrepancyPct
		a.Status = wantStatus
		return nil
	})
	if updateErr != nil {
		t.Fatalf("UpdateAnchor: %v", updateErr)
	}

	// --- Snapshot assertions: all 6 fields set ---
	snap := li.Snapshot()
	var got *model.Anchor
	for i := range snap.Anchors {
		if snap.Anchors[i].ID == predTestAnchorID {
			got = &snap.Anchors[i]
			break
		}
	}
	if got == nil {
		t.Fatal("anchor not found in snapshot after transition")
	}
	if got.Tier != model.TierMeasurement {
		t.Errorf("Tier: got %d want %d (TierMeasurement)", got.Tier, model.TierMeasurement)
	}
	if got.MeasuredValue == nil || *got.MeasuredValue != observed {
		t.Errorf("MeasuredValue: got %v want %v", got.MeasuredValue, observed)
	}
	if got.MeasuredError == nil || *got.MeasuredError != measuredErr {
		t.Errorf("MeasuredError: got %v want %v", got.MeasuredError, measuredErr)
	}
	if got.MeasuredSource != measuredSrc {
		t.Errorf("MeasuredSource: got %q want %q", got.MeasuredSource, measuredSrc)
	}
	if got.LastTestedAt == nil || *got.LastTestedAt != lastTested {
		t.Errorf("LastTestedAt: got %v want %q", got.LastTestedAt, lastTested)
	}
	if got.DiscrepancyPct == nil || math.Abs(*got.DiscrepancyPct-wantDiscrepancyPct) > 1e-9 {
		t.Errorf("DiscrepancyPct: got %v want %v", got.DiscrepancyPct, wantDiscrepancyPct)
	}
	if got.Status != wantStatus {
		t.Errorf("Status: got %v want %v", got.Status, wantStatus)
	}

	// --- Hook assertions: transition observable in before/after ---
	if hookCount != 1 {
		t.Errorf("hook count after transition: got %d want 1", hookCount)
	}
	if capturedBefore == nil || capturedBefore.Tier != model.TierPrediction {
		t.Errorf("hook before.Tier: got %v want TierPrediction (3)", capturedBefore)
	}
	if capturedAfter == nil || capturedAfter.Tier != model.TierMeasurement {
		t.Errorf("hook after.Tier: got %v want TierMeasurement (2)", capturedAfter)
	}
	if capturedBefore == nil || capturedBefore.Status != model.StatusUntested {
		t.Errorf("hook before.Status: got %v want StatusUntested", capturedBefore)
	}
	if capturedAfter == nil || capturedAfter.Status != wantStatus {
		t.Errorf("hook after.Status: got %v want %v", capturedAfter, wantStatus)
	}
	if capturedBefore != nil && capturedBefore.MeasuredValue != nil {
		t.Errorf("hook before.MeasuredValue: got %v want nil", capturedBefore.MeasuredValue)
	}
	if capturedAfter == nil || capturedAfter.MeasuredValue == nil {
		t.Errorf("hook after.MeasuredValue: got nil want non-nil")
	}

	// --- Disk persistence: reload confirms all 6 field changes ---
	reloaded, reloadErr := LoadInventory(path)
	if reloadErr != nil {
		t.Fatalf("reload: %v", reloadErr)
	}
	var disk *model.Anchor
	for i := range reloaded.Anchors {
		if reloaded.Anchors[i].ID == predTestAnchorID {
			disk = &reloaded.Anchors[i]
			break
		}
	}
	if disk == nil {
		t.Fatal("anchor not found on disk after transition")
	}
	if disk.Tier != model.TierMeasurement {
		t.Errorf("disk Tier: got %d want %d", disk.Tier, model.TierMeasurement)
	}
	if disk.Status != wantStatus {
		t.Errorf("disk Status: got %v want %v", disk.Status, wantStatus)
	}
	if disk.MeasuredValue == nil || *disk.MeasuredValue != observed {
		t.Errorf("disk MeasuredValue: got %v want %v", disk.MeasuredValue, observed)
	}
}

// TestLiveInventory_PredictionToMeasurementTransition_RegimeMatches verifies
// that the three regime transitions (Coherent / Contested / Refuted) all round-
// trip correctly through UpdateAnchor. compute.ScorePrediction drives
// DiscrepancyPct + Regime → Status in each case.
//
// Contract per doc/design/live-inventory-api.md §2.2.
func TestLiveInventory_PredictionToMeasurementTransition_RegimeMatches(t *testing.T) {
	cases := []struct {
		name           string
		predicted      float64
		observed       float64
		expectedStatus model.Status
		expectedRegime compute.ScoreRegime
	}{
		{
			name:           "laminar_coherent",
			predicted:      42.0,
			observed:       41.97, // delta ~0.07% — laminar
			expectedStatus: model.StatusCoherent,
			expectedRegime: compute.ScoreRegimeLaminar,
		},
		{
			name:           "moderate_contested",
			predicted:      42.0,
			observed:       35.0, // delta ~16.67% — moderate
			expectedStatus: model.StatusContested,
			expectedRegime: compute.ScoreRegimeModerate,
		},
		{
			name:           "heavy_refuted",
			predicted:      42.0,
			observed:       70.0, // delta ~66.67% — heavy
			expectedStatus: model.StatusRefuted,
			expectedRegime: compute.ScoreRegimeHeavy,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			li, path := openTempLive(t, nil)
			defer li.Close() //nolint:errcheck // test defer

			anchorID := "PRED-regime-" + tc.name
			pred := freshPredictionAnchor(anchorID)
			pred.PredictedValue = &tc.predicted
			if err := li.AppendAnchor(pred); err != nil {
				t.Fatalf("AppendAnchor: %v", err)
			}

			score, err := compute.ScorePrediction(compute.KindScalar, tc.predicted, tc.observed)
			if err != nil {
				t.Fatalf("ScorePrediction: %v", err)
			}
			if score.Regime != tc.expectedRegime {
				t.Errorf("ScorePrediction regime: got %v want %v", score.Regime, tc.expectedRegime)
			}

			discrepancyPct := score.DiscrepancyPct
			lastTested := predLastTestedAt
			wantStatus := regimeToStatus(score.Regime)

			if err := li.UpdateAnchor(anchorID, func(a *model.Anchor) error {
				a.Tier = model.TierMeasurement
				a.MeasuredValue = &tc.observed
				a.DiscrepancyPct = &discrepancyPct
				a.LastTestedAt = &lastTested
				a.Status = wantStatus
				return nil
			}); err != nil {
				t.Fatalf("UpdateAnchor: %v", err)
			}

			// Snapshot confirms transition.
			snap := li.Snapshot()
			var got *model.Anchor
			for i := range snap.Anchors {
				if snap.Anchors[i].ID == anchorID {
					got = &snap.Anchors[i]
					break
				}
			}
			if got == nil {
				t.Fatal("anchor not found in snapshot")
			}
			if got.Tier != model.TierMeasurement {
				t.Errorf("Tier: got %d want TierMeasurement", got.Tier)
			}
			if got.Status != tc.expectedStatus {
				t.Errorf("Status: got %v want %v", got.Status, tc.expectedStatus)
			}
			if got.DiscrepancyPct == nil || math.Abs(*got.DiscrepancyPct-score.DiscrepancyPct) > 1e-9 {
				t.Errorf("DiscrepancyPct: got %v want %v", got.DiscrepancyPct, score.DiscrepancyPct)
			}

			// Disk persistence.
			reloaded, reloadErr := LoadInventory(path)
			if reloadErr != nil {
				t.Fatalf("reload: %v", reloadErr)
			}
			var disk *model.Anchor
			for i := range reloaded.Anchors {
				if reloaded.Anchors[i].ID == anchorID {
					disk = &reloaded.Anchors[i]
					break
				}
			}
			if disk == nil {
				t.Fatal("anchor not found on disk")
			}
			if disk.Tier != model.TierMeasurement {
				t.Errorf("disk Tier: got %d want TierMeasurement", disk.Tier)
			}
			if disk.Status != tc.expectedStatus {
				t.Errorf("disk Status: got %v want %v", disk.Status, tc.expectedStatus)
			}
		})
	}
}

// TestLiveInventory_PredictionToMeasurementTransition_HookWhitelistAllFire
// verifies that a single UpdateAnchor touching all 5 whitelist fields fires
// OnAnchorChange exactly once (not 5x — hook fires per-call, not per-field)
// and that the before/after pair reflects all 5 field changes.
//
// Whitelist (main branch): {Status, MeasuredValue, MeasuredError, DiscrepancyPct, LastTestedAt}.
// Contract per doc/design/live-inventory-api.md §2.1 + §2.2.
func TestLiveInventory_PredictionToMeasurementTransition_HookWhitelistAllFire(t *testing.T) {
	hookCount := 0
	var capturedBefore, capturedAfter *model.Anchor

	hooks := &Hooks{
		OnAnchorChange: func(before, after *model.Anchor) {
			hookCount++
			capturedBefore = before
			capturedAfter = after
		},
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	anchorID := "PRED-whitelist-all"
	pred := freshPredictionAnchor(anchorID)
	if err := li.AppendAnchor(pred); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}
	hookCount = 0 // reset; count only the update hook

	predicted := 42.0
	observed := 41.7
	score, err := compute.ScorePrediction(compute.KindScalar, predicted, observed)
	if err != nil {
		t.Fatalf("ScorePrediction: %v", err)
	}

	measuredErr := 0.1
	lastTested := predLastTestedAt
	discrepancyPct := score.DiscrepancyPct

	if err := li.UpdateAnchor(anchorID, func(a *model.Anchor) error {
		// Touch all 5 whitelist fields in one mutator call.
		a.MeasuredValue = &observed
		a.MeasuredError = &measuredErr
		a.DiscrepancyPct = &discrepancyPct
		a.LastTestedAt = &lastTested
		a.Status = model.StatusCoherent
		// Tier transition also set as part of the full transition shape.
		a.Tier = model.TierMeasurement
		return nil
	}); err != nil {
		t.Fatalf("UpdateAnchor: %v", err)
	}

	// Hook fires exactly once per UpdateAnchor call, not once per field.
	if hookCount != 1 {
		t.Errorf("hook count: got %d want 1 (fires per-call, not per-field)", hookCount)
	}
	if capturedBefore == nil || capturedAfter == nil {
		t.Fatal("hook before/after must not be nil")
	}

	// All 5 whitelist fields differ between before and after.
	if capturedBefore.Status == capturedAfter.Status {
		t.Errorf("Status unchanged: before=%v after=%v", capturedBefore.Status, capturedAfter.Status)
	}
	if float64PtrEqual(capturedBefore.MeasuredValue, capturedAfter.MeasuredValue) {
		t.Errorf("MeasuredValue unchanged: before=%v after=%v", capturedBefore.MeasuredValue, capturedAfter.MeasuredValue)
	}
	if float64PtrEqual(capturedBefore.MeasuredError, capturedAfter.MeasuredError) {
		t.Errorf("MeasuredError unchanged: before=%v after=%v", capturedBefore.MeasuredError, capturedAfter.MeasuredError)
	}
	if float64PtrEqual(capturedBefore.DiscrepancyPct, capturedAfter.DiscrepancyPct) {
		t.Errorf("DiscrepancyPct unchanged: before=%v after=%v", capturedBefore.DiscrepancyPct, capturedAfter.DiscrepancyPct)
	}
	// LastTestedAt: before is nil; after is set — they differ (nil-to-non-nil).
	if capturedAfter.LastTestedAt == nil {
		t.Errorf("LastTestedAt after: got nil want non-nil")
	}
	if capturedBefore.LastTestedAt != nil {
		t.Errorf("LastTestedAt before: got non-nil want nil")
	}
}

// TestLiveInventory_PartialTransition_HookFiresWhenWhitelistTouched is a
// negative-discipline test that documents the known gap in the v0.3 API.
//
// The current API allows a mutator to set MeasuredValue without changing Tier
// (no auto-Tier-transition; the caller is responsible). This test asserts the
// current observed behavior:
//   - UpdateAnchor succeeds even when only MeasuredValue changes (Tier stays at 3)
//   - OnAnchorChange still fires (MeasuredValue is in the whitelist)
//   - Tier in the snapshot remains TierPrediction
//
// Known gap (v0.3 deferred): Anchor.Validate() does not enforce the joint-
// population shape (all 6 fields moving together). This is the documented gap
// from sprint-1-closeout-2026-05-17 seq=12 — future Invariant 6 candidate.
// See doc/design/live-inventory-api.md §2.2.
func TestLiveInventory_PartialTransition_HookFiresWhenWhitelistTouched(t *testing.T) {
	hookCount := 0
	var capturedAfter *model.Anchor

	hooks := &Hooks{
		OnAnchorChange: func(_, after *model.Anchor) {
			hookCount++
			capturedAfter = after
		},
	}

	li, _ := openTempLive(t, hooks)
	defer li.Close() //nolint:errcheck // test defer

	anchorID := "PRED-partial-01"
	pred := freshPredictionAnchor(anchorID)
	if err := li.AppendAnchor(pred); err != nil {
		t.Fatalf("AppendAnchor: %v", err)
	}
	hookCount = 0 // reset

	// Mutator sets MeasuredValue ONLY — Tier stays at TierPrediction.
	// This is the documented-gap case: the API allows this even though it
	// creates a semantically incomplete measurement record.
	observed := 41.7
	if err := li.UpdateAnchor(anchorID, func(a *model.Anchor) error {
		a.MeasuredValue = &observed
		return nil
	}); err != nil {
		t.Fatalf("UpdateAnchor: %v", err)
	}

	// OnAnchorChange fires because MeasuredValue is in the whitelist.
	if hookCount != 1 {
		t.Errorf("hook count: got %d want 1 (MeasuredValue is whitelist)", hookCount)
	}

	// Tier in snapshot is still TierPrediction — the API did NOT auto-promote.
	snap := li.Snapshot()
	var got *model.Anchor
	for i := range snap.Anchors {
		if snap.Anchors[i].ID == anchorID {
			got = &snap.Anchors[i]
			break
		}
	}
	if got == nil {
		t.Fatal("anchor not found in snapshot")
	}
	if got.Tier != model.TierPrediction {
		t.Errorf("Tier: got %d want %d (TierPrediction — no auto-promotion)", got.Tier, model.TierPrediction)
	}
	if got.MeasuredValue == nil || *got.MeasuredValue != observed {
		t.Errorf("MeasuredValue: got %v want %v", got.MeasuredValue, observed)
	}

	// Confirm hook captured the partial state.
	if capturedAfter == nil || capturedAfter.Tier != model.TierPrediction {
		t.Errorf("hook after.Tier: got %v want TierPrediction", capturedAfter)
	}

	// t.Log documents the semantic gap for future Invariant 6 candidate.
	t.Log("KNOWN GAP (v0.3 deferred): MeasuredValue set without Tier transition. " +
		"Anchor.Validate() does not enforce joint-population of all 6 transition fields. " +
		"Future Invariant 6 candidate: enforce Tier==TierMeasurement when MeasuredValue!=nil. " +
		"See doc/design/live-inventory-api.md §2.2.")
}
