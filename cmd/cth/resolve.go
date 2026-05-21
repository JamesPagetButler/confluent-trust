package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JamesPagetButler/confluent-trust/model"
	"github.com/JamesPagetButler/confluent-trust/store"
)

// ResolveAnchor looks up the record by ID across an inventory's anchors[],
// axioms[], and derived_principles[] arrays. Stable lookup order: anchors[]
// first, then axioms[], then derived_principles[]. If an ID collision exists
// (which Inventory.Validate() rejects), the first match wins.
//
// For axioms + derived_principles, the function synthesizes a model.Anchor
// with Tier set per the workspace convention (axioms = TierAxiom = 0;
// derived_principles = TierProof = 1) so the returned shape is uniform.
//
// Returns ok=true if found.
func ResolveAnchor(inv model.Inventory, id string) (model.Anchor, bool) {
	for i := range inv.Anchors {
		if inv.Anchors[i].ID == id {
			return inv.Anchors[i], true
		}
	}
	for i := range inv.Axioms {
		if inv.Axioms[i].ID == id {
			// Synthesize Anchor shape from Axiom.
			return model.Anchor{
				ID:          inv.Axioms[i].ID,
				Name:        inv.Axioms[i].Name,
				Description: inv.Axioms[i].Statement,
				Tier:        model.TierAxiom,
				// Status / Provenance / PredictionChain left as zero-values;
				// axioms don't carry these in the v0.3 schema.
			}, true
		}
	}
	for i := range inv.DerivedPrinciples {
		if inv.DerivedPrinciples[i].ID == id {
			return model.Anchor{
				ID:              inv.DerivedPrinciples[i].ID,
				Name:            inv.DerivedPrinciples[i].Name,
				Description:     inv.DerivedPrinciples[i].Statement,
				Tier:            model.TierProof,
				PredictionChain: inv.DerivedPrinciples[i].DerivedFrom,
			}, true
		}
	}
	return model.Anchor{}, false
}

func runResolve(args []string) error {
	out, pos, err := parseFlags("resolve", args)
	if err != nil {
		return err
	}
	if len(pos) != 2 {
		return errors.New("resolve: expects two arguments: <inventory.json> <anchor-id>")
	}
	inv, err := store.LoadInventory(pos[0])
	if err != nil {
		return err
	}
	anchor, found := ResolveAnchor(inv, pos[1])
	if !found {
		return fmt.Errorf("resolve: anchor %q not found in %s", pos[1], pos[0])
	}
	data, err := json.MarshalIndent(anchor, "", "  ")
	if err != nil {
		return fmt.Errorf("resolve: marshal: %w", err)
	}
	return writeOutput(out, string(data))
}
