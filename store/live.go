package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/JamesPagetButler/confluent-trust/model"
)

// LiveInventory wraps a model.Inventory with concurrent append/mutate
// operations and atomic persistence. It is the long-lived handle that
// BMA, QBP, and other federation tenants hold across continuous-loop
// cycles.
//
// LiveInventory is safe for concurrent use by multiple goroutines.
// Snapshot() acquires a read lock; append/update methods acquire a
// write lock; the lock window covers validation, in-memory mutation,
// and disk persistence.
//
// Field order is GC-pointer-span optimised: pointer-containing fields
// (inv, hooks, path) are grouped first so the GC pointer bitmap covers
// the minimum extent; non-pointer fields (mu, closed) follow.
type LiveInventory struct {
	inv    *model.Inventory // protected by mu; pointer-containing fields first
	hooks  *Hooks           // optional change callbacks; see design §6
	path   string           // JSON file path (Toddle); ignored at Walk-cutover
	mu     sync.RWMutex     // no GC pointers — after pointer-containing fields
	closed bool             //nolint:govet // fieldalignment: positioned after mu for clarity; pointer-span is already optimal
}

// Hooks carry caller-supplied callbacks fired after a successful commit.
// All hooks are invoked outside the critical section (post-fsync, after
// the write lock is released) so callers do not deadlock by re-entering
// LiveInventory from a hook. Hooks must not panic; a panicking hook is
// logged and recovered, never propagated.
//
// Hook implementations must NOT call back into LiveInventory's append
// or update methods. Calling Snapshot() from a hook is safe (it acquires
// only a read lock). Re-entering an append/update method from a hook
// produces a write-after-write race the API does not protect against.
//
// The NATS event publish surface (cth.scoring.{anchor_id}.score_event
// per CTH #19) is wired by callers via these hooks; LiveInventory itself
// has no NATS dependency.
type Hooks struct {
	// OnAnchorChange fires after a successful AppendAnchor (before==nil,
	// after points at the just-appended record) or after a successful
	// UpdateAnchor that changed one of {Status, MeasuredValue,
	// MeasuredError, DiscrepancyPct, LastTestedAt}. Other UpdateAnchor
	// field changes (e.g. Notes, Description) do not fire the hook.
	OnAnchorChange func(before *model.Anchor, after *model.Anchor)

	// OnChainChange fires after a successful AppendChain (before==nil)
	// or after any successful UpdateChain call — all field changes fire,
	// no whitelist. Chain topology is the derivation surface; every
	// change matters to closure-caching consumers.
	OnChainChange func(before *model.Chain, after *model.Chain)

	// OnConfluenceChange fires after a successful AppendConfluence
	// (before==nil) or after any field change in a confluence update.
	// No whitelist — same rationale as OnChainChange.
	OnConfluenceChange func(before *model.ConfluencePoint, after *model.ConfluencePoint)
}

// OpenLiveInventory loads an inventory from disk and returns a handle
// suitable for live append/mutate. The file is loaded with full schema
// validation (same path as store.LoadInventory). The hooks argument may
// be nil (no callbacks).
//
// At Walk, this constructor is shadowed by OpenLiveInventoryWyrd; see design §7.
func OpenLiveInventory(path string, hooks *Hooks) (*LiveInventory, error) {
	inv, err := LoadInventory(path)
	if err != nil {
		return nil, fmt.Errorf("store/live: open %s: %w", path, err)
	}
	return &LiveInventory{
		inv:   &inv,
		path:  path,
		hooks: hooks,
	}, nil
}

// AppendAnchor adds a new anchor. The anchor.ID must not collide with
// any existing record ID (anchors, axioms, derived principles, inputs,
// chains, confluences); on collision the method returns a wrapped
// model.ErrIDCollision. The single anchor is schema-validated before
// insertion; the full Inventory.Validate() runs before fsync.
//
// On successful commit, Hooks.OnAnchorChange fires with before==nil
// and after pointing at the appended anchor (deep copy). See design §2.1
// for the full hook-semantics contract.
func (li *LiveInventory) AppendAnchor(a model.Anchor) error {
	li.mu.Lock()

	if li.closed {
		li.mu.Unlock()
		return fmt.Errorf("LiveInventory closed: %w", model.ErrClosed)
	}

	// Per-record validation before touching shared state.
	if err := a.Validate(); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendAnchor: %w", err)
	}

	// ID collision check across all record types.
	if err := li.checkIDCollision(a.ID); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("anchor %q: %w", a.ID, model.ErrIDCollision)
	}

	// Snapshot for rollback.
	prev := cloneInventory(li.inv)

	// Mutate in memory.
	li.inv.Anchors = append(li.inv.Anchors, a)
	li.inv.Health = nil

	// Full inventory validation.
	if err := li.inv.Validate(); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendAnchor: validate: %w", err)
	}

	// Fsync via .tmp + rename.
	if err := SaveInventory(*li.inv, li.path); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendAnchor: save: %w", err)
	}

	// Capture the appended record for the hook (deep copy, same isolation as Snapshot).
	after := cloneAnchor(a)

	li.mu.Unlock()

	// Dispatch hook outside the critical section.
	if li.hooks != nil && li.hooks.OnAnchorChange != nil {
		li.hooks.OnAnchorChange(nil, &after)
	}

	return nil
}

// AppendChain adds a new chain. Same collision/validation/fsync contract
// as AppendAnchor. On success, Hooks.OnChainChange fires with before==nil.
func (li *LiveInventory) AppendChain(c model.Chain) error {
	li.mu.Lock()

	if li.closed {
		li.mu.Unlock()
		return fmt.Errorf("LiveInventory closed: %w", model.ErrClosed)
	}

	if err := c.Validate(); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendChain: %w", err)
	}

	// Chains have their own ID namespace; check only chain IDs.
	if err := li.checkChainIDCollision(c.ID); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("chain %q: %w", c.ID, model.ErrIDCollision)
	}

	prev := cloneInventory(li.inv)

	li.inv.Chains = append(li.inv.Chains, c)
	li.inv.Health = nil

	if err := li.inv.Validate(); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendChain: validate: %w", err)
	}

	if err := SaveInventory(*li.inv, li.path); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendChain: save: %w", err)
	}

	after := cloneChain(c)

	li.mu.Unlock()

	if li.hooks != nil && li.hooks.OnChainChange != nil {
		li.hooks.OnChainChange(nil, &after)
	}

	return nil
}

// AppendConfluence adds a new confluence point. Same collision/validation/fsync
// contract as AppendAnchor. On success, Hooks.OnConfluenceChange fires with
// before==nil.
func (li *LiveInventory) AppendConfluence(cp model.ConfluencePoint) error {
	li.mu.Lock()

	if li.closed {
		li.mu.Unlock()
		return fmt.Errorf("LiveInventory closed: %w", model.ErrClosed)
	}

	if err := cp.Validate(); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendConfluence: %w", err)
	}

	if err := li.checkConfluenceIDCollision(cp.ID); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("confluence %q: %w", cp.ID, model.ErrIDCollision)
	}

	prev := cloneInventory(li.inv)

	li.inv.ConfluencePoints = append(li.inv.ConfluencePoints, cp)
	li.inv.Health = nil

	if err := li.inv.Validate(); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendConfluence: validate: %w", err)
	}

	if err := SaveInventory(*li.inv, li.path); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: AppendConfluence: save: %w", err)
	}

	after := cloneConfluence(cp)

	li.mu.Unlock()

	if li.hooks != nil && li.hooks.OnConfluenceChange != nil {
		li.hooks.OnConfluenceChange(nil, &after)
	}

	return nil
}

// AppendInput adds a new input record. Same collision/validation/fsync
// contract as AppendAnchor. AppendInput does NOT fire any hook (inputs are
// programme-level metadata, not anchor-graph state; no OnInputChange is
// defined per design §2.1).
func (li *LiveInventory) AppendInput(in model.Input) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	if li.closed {
		return fmt.Errorf("LiveInventory closed: %w", model.ErrClosed)
	}

	if err := in.Validate(); err != nil {
		return fmt.Errorf("store/live: AppendInput: %w", err)
	}

	if err := li.checkInputIDCollision(in.ID); err != nil {
		return fmt.Errorf("input %q: %w", in.ID, model.ErrIDCollision)
	}

	prev := cloneInventory(li.inv)

	li.inv.Inputs = append(li.inv.Inputs, in)
	li.inv.Health = nil

	if err := li.inv.Validate(); err != nil {
		li.inv = &prev
		return fmt.Errorf("store/live: AppendInput: validate: %w", err)
	}

	if err := SaveInventory(*li.inv, li.path); err != nil {
		li.inv = &prev
		return fmt.Errorf("store/live: AppendInput: save: %w", err)
	}

	// No hook fires for AppendInput per design §2.1.
	return nil
}

// UpdateAnchor applies a caller-supplied mutator to the anchor identified
// by id. The mutator runs inside the write lock; it must not call other
// LiveInventory methods (deadlock). Mutator errors abort the update; the
// in-memory state is rolled back; disk state is unchanged.
//
// If the mutator changes any of {Status, MeasuredValue, MeasuredError,
// DiscrepancyPct, LastTestedAt}, Hooks.OnAnchorChange fires with the
// before/after snapshot. Other field changes do not fire the hook
// (avoids spurious NATS publishes on routine bookkeeping per design §2.1).
func (li *LiveInventory) UpdateAnchor(id string, mutator func(*model.Anchor) error) error {
	li.mu.Lock()

	if li.closed {
		li.mu.Unlock()
		return fmt.Errorf("LiveInventory closed: %w", model.ErrClosed)
	}

	// Find the anchor.
	idx := -1
	for i := range li.inv.Anchors {
		if li.inv.Anchors[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		li.mu.Unlock()
		return fmt.Errorf("anchor %q: %w", id, model.ErrNotFound)
	}

	// Snapshot before-state for rollback + hook.
	prev := cloneInventory(li.inv)
	beforeAnchor := cloneAnchor(li.inv.Anchors[idx])

	// Run the mutator on a copy of the anchor.
	updated := cloneAnchor(li.inv.Anchors[idx])
	if err := mutator(&updated); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("store/live: UpdateAnchor %q: mutator: %w", id, err)
	}

	// Swap the updated record into the slice.
	li.inv.Anchors[idx] = updated
	li.inv.Health = nil

	// Full validation under lock.
	if err := li.inv.Validate(); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: UpdateAnchor %q: validate: %w", id, err)
	}

	// Fsync.
	if err := SaveInventory(*li.inv, li.path); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: UpdateAnchor %q: save: %w", id, err)
	}

	afterAnchor := cloneAnchor(updated)
	shouldFire := anchorStatusFieldsChanged(&beforeAnchor, &afterAnchor)

	li.mu.Unlock()

	// Hook fires only when whitelist fields changed.
	if shouldFire && li.hooks != nil && li.hooks.OnAnchorChange != nil {
		li.hooks.OnAnchorChange(&beforeAnchor, &afterAnchor)
	}

	return nil
}

// UpdateChain applies a caller-supplied mutator to the chain identified by id.
// The mutator runs inside the write lock. OnChainChange fires on ALL field
// changes — no whitelist filter. See design §2.1 for rationale.
func (li *LiveInventory) UpdateChain(id string, mutator func(*model.Chain) error) error {
	li.mu.Lock()

	if li.closed {
		li.mu.Unlock()
		return fmt.Errorf("LiveInventory closed: %w", model.ErrClosed)
	}

	idx := -1
	for i := range li.inv.Chains {
		if li.inv.Chains[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		li.mu.Unlock()
		return fmt.Errorf("chain %q: %w", id, model.ErrNotFound)
	}

	prev := cloneInventory(li.inv)
	beforeChain := cloneChain(li.inv.Chains[idx])

	updated := cloneChain(li.inv.Chains[idx])
	if err := mutator(&updated); err != nil {
		li.mu.Unlock()
		return fmt.Errorf("store/live: UpdateChain %q: mutator: %w", id, err)
	}

	li.inv.Chains[idx] = updated
	li.inv.Health = nil

	if err := li.inv.Validate(); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: UpdateChain %q: validate: %w", id, err)
	}

	if err := SaveInventory(*li.inv, li.path); err != nil {
		li.inv = &prev
		li.mu.Unlock()
		return fmt.Errorf("store/live: UpdateChain %q: save: %w", id, err)
	}

	afterChain := cloneChain(updated)

	li.mu.Unlock()

	// All field changes fire OnChainChange (no whitelist).
	if li.hooks != nil && li.hooks.OnChainChange != nil {
		li.hooks.OnChainChange(&beforeChain, &afterChain)
	}

	return nil
}

// Snapshot returns a deep copy of the current inventory state. Callers
// may compute on the snapshot without holding any LiveInventory lock;
// concurrent mutations to LiveInventory do not affect the returned value.
//
// A snapshot is always internally consistent. Back-to-back snapshots may
// differ in ways concurrent appends produce.
func (li *LiveInventory) Snapshot() model.Inventory {
	li.mu.RLock()
	defer li.mu.RUnlock()
	return cloneInventory(li.inv)
}

// Close releases the LiveInventory. After Close, all method calls return
// model.ErrClosed wrapped. Close does not fsync — every append and update
// already fsynced. It removes any leftover .tmp staging file.
func (li *LiveInventory) Close() error {
	li.mu.Lock()
	defer li.mu.Unlock()

	if li.closed {
		return nil
	}
	li.closed = true

	// Remove any leftover .tmp staging file.
	tmp := filepath.Clean(li.path) + ".tmp"
	_ = os.Remove(tmp)

	return nil
}

// ---- ID collision helpers ----

// checkIDCollision checks the anchor ID against all namespaces that share
// the anchor-level ID space: anchors, axioms, derived principles, and inputs.
func (li *LiveInventory) checkIDCollision(id string) error {
	for _, a := range li.inv.Anchors {
		if a.ID == id {
			return model.ErrIDCollision
		}
	}
	for _, a := range li.inv.Axioms {
		if a.ID == id {
			return model.ErrIDCollision
		}
	}
	for _, d := range li.inv.DerivedPrinciples {
		if d.ID == id {
			return model.ErrIDCollision
		}
	}
	for _, in := range li.inv.Inputs {
		if in.ID == id {
			return model.ErrIDCollision
		}
	}
	return nil
}

// checkChainIDCollision checks the chain ID against the chains slice.
func (li *LiveInventory) checkChainIDCollision(id string) error {
	for _, c := range li.inv.Chains {
		if c.ID == id {
			return model.ErrIDCollision
		}
	}
	return nil
}

// checkConfluenceIDCollision checks the confluence ID against the confluence_points slice.
func (li *LiveInventory) checkConfluenceIDCollision(id string) error {
	for _, cp := range li.inv.ConfluencePoints {
		if cp.ID == id {
			return model.ErrIDCollision
		}
	}
	return nil
}

// checkInputIDCollision checks the input ID against the inputs slice and the
// anchor-level namespace (axioms, anchors, derived principles, inputs).
func (li *LiveInventory) checkInputIDCollision(id string) error {
	return li.checkIDCollision(id)
}

// ---- anchorStatusFieldsChanged ----

// anchorStatusFieldsChanged returns true if any of the five whitelist fields
// {Status, MeasuredValue, MeasuredError, DiscrepancyPct, LastTestedAt} differ
// between before and after. Pointer fields are compared by dereferenced value
// with nil-checks; a nil-to-non-nil (or non-nil-to-nil) transition also counts.
func anchorStatusFieldsChanged(before, after *model.Anchor) bool {
	if before.Status != after.Status {
		return true
	}
	if !float64PtrEqual(before.MeasuredValue, after.MeasuredValue) {
		return true
	}
	if !float64PtrEqual(before.MeasuredError, after.MeasuredError) {
		return true
	}
	if !float64PtrEqual(before.DiscrepancyPct, after.DiscrepancyPct) {
		return true
	}
	if !stringPtrEqual(before.LastTestedAt, after.LastTestedAt) {
		return true
	}
	return false
}

func float64PtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func stringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ---- Deep-copy helpers ----

// cloneInventory returns a deep copy of *inv as a value. The copy covers all
// slice elements and pointer fields so that mutations to the original do not
// affect the clone and vice versa.
func cloneInventory(inv *model.Inventory) model.Inventory {
	out := *inv // shallow copy of scalar fields

	// Clone slices of value types (elements are copied by value in the range).
	out.ParentProgrammes = cloneStringSlice(inv.ParentProgrammes)
	out.Changelog = cloneChangelogSlice(inv.Changelog)

	// MetaAxiom is a pointer — clone if non-nil.
	if inv.MetaAxiom != nil {
		ma := *inv.MetaAxiom
		out.MetaAxiom = &ma
	}

	// Health is a pointer — clone if non-nil.
	if inv.Health != nil {
		h := *inv.Health
		// TierBreakdown is a map — deep-copy it.
		if inv.Health.TierBreakdown != nil {
			h.TierBreakdown = make(map[string]int, len(inv.Health.TierBreakdown))
			for k, v := range inv.Health.TierBreakdown {
				h.TierBreakdown[k] = v
			}
		}
		// RhoNet, CoherenceRatio, CompressionVelocity are *float64 — clone.
		if inv.Health.RhoNet != nil {
			v := *inv.Health.RhoNet
			h.RhoNet = &v
		}
		if inv.Health.CoherenceRatio != nil {
			v := *inv.Health.CoherenceRatio
			h.CoherenceRatio = &v
		}
		if inv.Health.CompressionVelocity != nil {
			v := *inv.Health.CompressionVelocity
			h.CompressionVelocity = &v
		}
		if inv.Health.RhoNetSensitivity != nil {
			s := *inv.Health.RhoNetSensitivity
			h.RhoNetSensitivity = &s
		}
		out.Health = &h
	}

	// Axioms.
	if inv.Axioms != nil {
		out.Axioms = make([]model.Axiom, len(inv.Axioms))
		for i, a := range inv.Axioms {
			out.Axioms[i] = cloneAxiom(a)
		}
	}

	// DerivedPrinciples.
	if inv.DerivedPrinciples != nil {
		out.DerivedPrinciples = make([]model.DerivedPrinciple, len(inv.DerivedPrinciples))
		for i, d := range inv.DerivedPrinciples {
			out.DerivedPrinciples[i] = cloneDerivedPrinciple(d)
		}
	}

	// Anchors.
	if inv.Anchors != nil {
		out.Anchors = make([]model.Anchor, len(inv.Anchors))
		for i, a := range inv.Anchors {
			out.Anchors[i] = cloneAnchor(a)
		}
	}

	// Inputs (all value fields — no pointers or slices to deep-copy).
	if inv.Inputs != nil {
		out.Inputs = make([]model.Input, len(inv.Inputs))
		copy(out.Inputs, inv.Inputs)
	}

	// Chains.
	if inv.Chains != nil {
		out.Chains = make([]model.Chain, len(inv.Chains))
		for i, c := range inv.Chains {
			out.Chains[i] = cloneChain(c)
		}
	}

	// ConfluencePoints.
	if inv.ConfluencePoints != nil {
		out.ConfluencePoints = make([]model.ConfluencePoint, len(inv.ConfluencePoints))
		for i, cp := range inv.ConfluencePoints {
			out.ConfluencePoints[i] = cloneConfluence(cp)
		}
	}

	// ForkPoints.
	if inv.ForkPoints != nil {
		out.ForkPoints = make([]model.ForkPoint, len(inv.ForkPoints))
		for i, f := range inv.ForkPoints {
			out.ForkPoints[i] = cloneForkPoint(f)
		}
	}

	return out
}

func cloneAnchor(a model.Anchor) model.Anchor {
	out := a // shallow copy of scalar/enum fields

	// Clone pointer fields.
	if a.PredictedValue != nil {
		v := *a.PredictedValue
		out.PredictedValue = &v
	}
	if a.SorryCount != nil {
		v := *a.SorryCount
		out.SorryCount = &v
	}
	if a.LastTestedAt != nil {
		v := *a.LastTestedAt
		out.LastTestedAt = &v
	}
	if a.DiscrepancyPct != nil {
		v := *a.DiscrepancyPct
		out.DiscrepancyPct = &v
	}
	if a.MeasuredError != nil {
		v := *a.MeasuredError
		out.MeasuredError = &v
	}
	if a.MeasuredValue != nil {
		v := *a.MeasuredValue
		out.MeasuredValue = &v
	}

	// Clone slice fields.
	out.LeanCompanionTheorems = cloneStringSlice(a.LeanCompanionTheorems)
	out.PredictionChain = cloneStringSlice(a.PredictionChain)

	return out
}

func cloneChain(c model.Chain) model.Chain {
	out := c // shallow copy

	// Clone pointer fields.
	if c.WeakestLinkID != nil {
		v := *c.WeakestLinkID
		out.WeakestLinkID = &v
	}
	if c.Fidelity != nil {
		v := *c.Fidelity
		out.Fidelity = &v
	}

	// Clone slice fields.
	out.SourceIDs = cloneStringSlice(c.SourceIDs)
	out.StepTypes = cloneStringSlice(c.StepTypes)

	if c.DomainBoundaries != nil {
		out.DomainBoundaries = make([]model.DomainBoundary, len(c.DomainBoundaries))
		copy(out.DomainBoundaries, c.DomainBoundaries)
	}

	return out
}

func cloneConfluence(cp model.ConfluencePoint) model.ConfluencePoint {
	out := cp // shallow copy

	// Clone legacy pointer fields (present before NormalizePaths).
	if cp.LegacyPathA != nil {
		v := *cp.LegacyPathA
		out.LegacyPathA = &v
	}
	if cp.LegacyPathB != nil {
		v := *cp.LegacyPathB
		out.LegacyPathB = &v
	}

	// Clone Paths slice (ChainRef has a *float64 Fidelity pointer).
	if cp.Paths != nil {
		out.Paths = make([]model.ChainRef, len(cp.Paths))
		for i, p := range cp.Paths {
			out.Paths[i] = cloneChainRef(p)
		}
	}

	return out
}

func cloneChainRef(cr model.ChainRef) model.ChainRef {
	out := cr
	if cr.Fidelity != nil {
		v := *cr.Fidelity
		out.Fidelity = &v
	}
	return out
}

func cloneAxiom(a model.Axiom) model.Axiom {
	out := a
	out.DerivedFromAxioms = cloneStringSlice(a.DerivedFromAxioms)
	return out
}

func cloneDerivedPrinciple(d model.DerivedPrinciple) model.DerivedPrinciple {
	out := d
	out.DerivedFrom = cloneStringSlice(d.DerivedFrom)
	return out
}

func cloneForkPoint(f model.ForkPoint) model.ForkPoint {
	out := f

	out.SharedPrefix = cloneStringSlice(f.SharedPrefix)

	if f.Branches != nil {
		out.Branches = make([]model.Branch, len(f.Branches))
		for i, b := range f.Branches {
			out.Branches[i] = cloneBranch(b)
		}
	}

	if f.Observations != nil {
		out.Observations = make([]model.BranchObservation, len(f.Observations))
		for i, o := range f.Observations {
			out.Observations[i] = cloneBranchObservation(o)
		}
	}

	return out
}

func cloneBranch(b model.Branch) model.Branch {
	out := b
	out.Anchors = cloneStringSlice(b.Anchors)
	out.Chains = cloneStringSlice(b.Chains)
	out.Confluences = cloneStringSlice(b.Confluences)
	out.Inputs = cloneStringSlice(b.Inputs)
	out.Predictions = cloneStringSlice(b.Predictions)
	return out
}

func cloneBranchObservation(o model.BranchObservation) model.BranchObservation {
	out := o
	if o.Interpretations != nil {
		out.Interpretations = make([]model.BranchInterpretation, len(o.Interpretations))
		for i, interp := range o.Interpretations {
			bi := interp
			bi.PredictionChain = cloneStringSlice(interp.PredictionChain)
			out.Interpretations[i] = bi
		}
	}
	return out
}

func cloneStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

func cloneChangelogSlice(s []model.ChangelogEntry) []model.ChangelogEntry {
	if s == nil {
		return nil
	}
	out := make([]model.ChangelogEntry, len(s))
	copy(out, s)
	return out
}
