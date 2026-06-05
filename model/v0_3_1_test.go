package model

import (
	"encoding/json"
	"testing"
)

// ---- v0.3.1 additive anchor fields (#93 disposition trio + #96 foundation_batch) ----

func TestAnchorV031Fields_RoundTrip(t *testing.T) {
	a := baseAnchor("PROOF-v031-roundtrip")
	a.KilledBy = "QBP/analysis/HQM-MMI-derive-or-die-report.md"
	a.KilledNote = "tested and failed; witnessed negative result"
	a.ReviewFlag = "dependency went incoherent; re-evaluate basis"
	a.FoundationBatch = "474-row-sedenion-zero-divisors"

	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Anchor
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.KilledBy != a.KilledBy || back.KilledNote != a.KilledNote ||
		back.ReviewFlag != a.ReviewFlag || back.FoundationBatch != a.FoundationBatch {
		t.Errorf("v0.3.1 fields did not round-trip: got %+v", back)
	}
	if err := back.Validate(); err != nil {
		t.Errorf("anchor with v0.3.1 fields should validate: %v", err)
	}
}

func TestAnchorV031Fields_OmittedWhenEmpty(t *testing.T) {
	raw, err := json.Marshal(baseAnchor("PROOF-v031-omit"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{"killed_by", "killed_note", "review_flag", "foundation_batch"} {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, present := doc[key]; present {
			t.Errorf("empty %s should be omitted from JSON (additive-optional contract)", key)
		}
	}
}

// ---- v0.3.1 class_floors (contextus PR #32 Q1 ruling) ----

// baseInventoryV031 returns a minimal valid inventory for class_floors tests.
func baseInventoryV031() Inventory {
	return Inventory{
		Programme: "TEST",
		Version:   "0.0.1",
		Axioms:    []Axiom{},
		Anchors:   []Anchor{},
		Chains:    []Chain{},
	}
}

func TestClassFloors_RoundTripAndValidate(t *testing.T) {
	inv := baseInventoryV031()
	inv.ClassFloors = map[string]ClassFloor{
		"PRED": {Floor: 0.85},
		"MEAS": {Floor: 0.7},
	}
	if err := inv.Validate(); err != nil {
		t.Fatalf("in-range class floors should validate: %v", err)
	}

	raw, err := json.Marshal(&inv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Inventory
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := back.ClassFloors["PRED"].Floor; got != 0.85 {
		t.Errorf("class_floors[PRED].floor = %v, want 0.85", got)
	}
	if got := back.ClassFloors["MEAS"].Floor; got != 0.7 {
		t.Errorf("class_floors[MEAS].floor = %v, want 0.7", got)
	}
}

func TestClassFloors_OutOfRange_Negative(t *testing.T) {
	for name, floor := range map[string]float64{"above-one": 1.5, "below-zero": -0.1} {
		t.Run(name, func(t *testing.T) {
			inv := baseInventoryV031()
			inv.ClassFloors = map[string]ClassFloor{"PRED": {Floor: floor}}
			if err := inv.Validate(); err == nil {
				t.Errorf("floor %v should fail validation", floor)
			}
		})
	}
}

// TestClassFloors_AbsentIsValid pins the fail-closed contract's schema half:
// an inventory with no class_floors at all is valid — absence means "no
// automated status change permitted", an evaluator-side rule, not a
// document-validity rule.
func TestClassFloors_AbsentIsValid(t *testing.T) {
	inv := baseInventoryV031()
	if err := inv.Validate(); err != nil {
		t.Fatalf("inventory without class_floors should validate: %v", err)
	}
	raw, err := json.Marshal(&inv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, present := doc["class_floors"]; present {
		t.Error("empty class_floors should be omitted from JSON")
	}
}
