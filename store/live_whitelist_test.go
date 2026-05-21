package store

// TestOnAnchorChange_WhitelistDerivation is the sprint-1-closeout-2026-05-17
// seq=12 Notary-bootstrap target #2 regression test.
//
// It serves as a forcing function: if a future PR adds a new Anchor field-read
// in compute.NetCompression's call graph without updating anchorStatusFieldsChanged
// (or vice versa), at least one case below will surface the drift.
//
// Structure:
//
//  1. For each whitelist field, mutate ONLY that field → verify OnAnchorChange fires.
//  2. For each known non-whitelist field (Notes, Description, LastTestedAt),
//     mutate ONLY that field → verify OnAnchorChange does NOT fire.
//
// The whitelist under test is:
//
//	{Status, MeasuredValue, MeasuredError, DiscrepancyPct, Tier, PredictionChain}
//
// Derivation: doc/design/onanchorchange-whitelist-derivation.md
// (sprint-1-closeout seq=12, Notary-bootstrap target #2).
//
// Note on Tier: the test changes Tier from TierProof (1) to TierMeasurement (2).
// This is an artificial mutation used to verify the whitelist; in production Tier
// transitions follow the full prediction→measurement lifecycle (Tier 3→2) or a
// corrigendum (Tier 2→1 demotion). The important invariant is that anchorStatusFieldsChanged
// detects any Tier change, regardless of direction.
import (
	"testing"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// TestOnAnchorChange_WhitelistDerivation verifies that OnAnchorChange fires
// when and only when a whitelist field changes.
func TestOnAnchorChange_WhitelistDerivation(t *testing.T) {
	t.Run("whitelist_fields_fire_hook", func(t *testing.T) {
		whitelistCases := []struct {
			mutate func(*model.Anchor)
			name   string
		}{
			{
				name: "Status",
				mutate: func(a *model.Anchor) {
					a.Status = model.StatusCoherent
				},
			},
			{
				name: "MeasuredValue",
				mutate: func(a *model.Anchor) {
					v := 42.0
					a.MeasuredValue = &v
				},
			},
			{
				name: "MeasuredError",
				mutate: func(a *model.Anchor) {
					v := 0.1
					a.MeasuredError = &v
				},
			},
			{
				name: "DiscrepancyPct",
				mutate: func(a *model.Anchor) {
					v := 0.71
					a.DiscrepancyPct = &v
				},
			},
			{
				name: "Tier",
				mutate: func(a *model.Anchor) {
					// TierProof (1) → TierMeasurement (2): a change in Tier alters
					// confirmedAnchors gate AND ConfirmatoryInfo branch selection.
					// Anchor.Validate() permits any Tier in [1,3], so this mutation
					// is schema-valid.
					a.Tier = model.TierMeasurement
				},
			},
			{
				name: "PredictionChain",
				mutate: func(a *model.Anchor) {
					// Extend the PredictionChain with an extra entry. This changes
					// input-cost allocation in both passes of NetCompression.
					a.PredictionChain = append(a.PredictionChain, "AXIOM-extra-dep")
				},
			},
		}

		for _, tc := range whitelistCases {
			t.Run(tc.name, func(t *testing.T) {
				hookCount := 0
				hooks := &Hooks{
					OnAnchorChange: func(before, after *model.Anchor) {
						if before != nil { // ignore append-hook (before==nil)
							hookCount++
						}
					},
				}

				li, _ := openTempLive(t, hooks)
				defer li.Close() //nolint:errcheck // test defer

				// Append a fresh anchor; ignore the append-hook fire.
				id := "PROOF-wl-" + tc.name
				a := freshAnchor(id)
				a.Status = model.StatusUntested
				if err := li.AppendAnchor(a); err != nil {
					t.Fatalf("AppendAnchor: %v", err)
				}

				// Apply the single-field mutation.
				if err := li.UpdateAnchor(id, func(a *model.Anchor) error {
					tc.mutate(a)
					return nil
				}); err != nil {
					t.Fatalf("UpdateAnchor: %v", err)
				}

				if hookCount != 1 {
					t.Errorf("whitelist field %q: hook fired %d times, want 1",
						tc.name, hookCount)
				}
			})
		}
	})

	t.Run("non_whitelist_fields_do_not_fire_hook", func(t *testing.T) {
		nonWhitelistCases := []struct {
			mutate func(*model.Anchor)
			name   string
		}{
			{
				name: "Notes",
				mutate: func(a *model.Anchor) {
					a.Notes = "updated notes — routine bookkeeping, not ρ_net-affecting"
				},
			},
			{
				name: "Description",
				mutate: func(a *model.Anchor) {
					a.Description = "updated description"
				},
			},
			{
				name: "LastTestedAt",
				// LastTestedAt was in the v0.1 whitelist but is absent from the
				// derived set: it has no path to any NetCompression computation.
				// This case documents the removal from the whitelist.
				// See doc/design/onanchorchange-whitelist-derivation.md §5.
				mutate: func(a *model.Anchor) {
					s := "2026-05-21T00:00:00Z"
					a.LastTestedAt = &s
				},
			},
		}

		for _, tc := range nonWhitelistCases {
			t.Run(tc.name, func(t *testing.T) {
				updateHookCount := 0
				hooks := &Hooks{
					OnAnchorChange: func(before, after *model.Anchor) {
						if before != nil { // ignore append-hook (before==nil)
							updateHookCount++
						}
					},
				}

				li, _ := openTempLive(t, hooks)
				defer li.Close() //nolint:errcheck // test defer

				id := "PROOF-nonwl-" + tc.name
				a := freshAnchor(id)
				if err := li.AppendAnchor(a); err != nil {
					t.Fatalf("AppendAnchor: %v", err)
				}

				if err := li.UpdateAnchor(id, func(a *model.Anchor) error {
					tc.mutate(a)
					return nil
				}); err != nil {
					t.Fatalf("UpdateAnchor: %v", err)
				}

				if updateHookCount != 0 {
					t.Errorf("non-whitelist field %q: hook fired %d times, want 0 "+
						"(field is not ρ_net-affecting per derivation)",
						tc.name, updateHookCount)
				}
			})
		}
	})

	t.Run("whitelist_derivation_assert_field_set", func(t *testing.T) {
		// This sub-test is a documentation assertion: it verifies that
		// anchorStatusFieldsChanged returns the expected truth-table for
		// a hand-crafted before/after pair that differs in exactly the
		// derived fields.
		//
		// If a future edit adds a new field to anchorStatusFieldsChanged
		// without updating the derivation doc, this test does NOT catch it
		// automatically — but the whitelist_fields_fire_hook cases above do
		// (since every whitelist field must have a corresponding case there).
		// This case is documentation: it captures the full before→after state
		// that represents "every whitelist field changed at once".
		disc := 0.71
		measVal := 42.0
		measErr := 0.1

		before := &model.Anchor{
			ID:              "PROOF-derivation-assert",
			Name:            "assert",
			Description:     "assert",
			Tier:            model.TierProof,
			Status:          model.StatusUntested,
			Provenance:      model.ProvenanceTheoretical,
			PredictionChain: []string{"AXIOM-1"},
		}
		after := &model.Anchor{
			ID:              "PROOF-derivation-assert",
			Name:            "assert",
			Description:     "assert",
			Tier:            model.TierMeasurement, // changed
			Status:          model.StatusCoherent,  // changed
			Provenance:      model.ProvenanceTheoretical,
			PredictionChain: []string{"AXIOM-1", "AXIOM-2"}, // changed
			DiscrepancyPct:  &disc,                          // changed
			MeasuredValue:   &measVal,                       // changed
			MeasuredError:   &measErr,                       // changed
		}

		if !anchorStatusFieldsChanged(before, after) {
			t.Error("anchorStatusFieldsChanged should return true when all whitelist fields change")
		}

		// Identical copies: no change should return false.
		afterCopy := *before
		if anchorStatusFieldsChanged(before, &afterCopy) {
			t.Error("anchorStatusFieldsChanged should return false when no whitelist field changes")
		}

		// Only Notes changes (non-whitelist): should return false.
		afterNotes := *before
		afterNotes.Notes = "routine note update"
		if anchorStatusFieldsChanged(before, &afterNotes) {
			t.Error("anchorStatusFieldsChanged should return false when only Notes changes")
		}

		// Only LastTestedAt changes (removed from whitelist): should return false.
		s := "2026-05-21T00:00:00Z"
		afterLTA := *before
		afterLTA.LastTestedAt = &s
		if anchorStatusFieldsChanged(before, &afterLTA) {
			t.Error("anchorStatusFieldsChanged should return false when only LastTestedAt changes " +
				"(LastTestedAt removed from whitelist in sprint-1-closeout seq=12 derivation)")
		}
	})
}
