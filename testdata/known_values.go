// Package testdata holds expected outputs for the fixtures in this directory.
// Compute tests in package compute import this map and assert against it as
// each function lands. Empty fields are filled in incrementally as Issues
// #4 through #14 implement the corresponding compute functions.
package testdata

// KnownValues maps a fixture filename to its expected compute outputs.
// Hand-verified or Python-engine-verified. Update when the theory says so;
// never to chase a Go test that disagrees with the canonical values.
type KnownValues struct {
	// Issue #4: residual entropy + confirmatory information
	ResidualEntropy   map[string]float64 // anchor id -> bits
	ConfirmatoryInfo  map[string]float64 // anchor id -> bits

	// Issue #5: chain fidelity + sediment regime
	ChainFidelity     map[string]float64 // chain id -> fidelity
	FidelityRegime    map[string]string  // chain id -> "laminar"|"low_sediment"|"moderate"|"heavy"

	// Issue #7: compression
	RhoGross          float64
	RhoNet            float64

	// Issue #8: sensitivity bracket
	SensitivityHalfH   float64
	SensitivityBaseH   float64
	SensitivityDoubleH float64

	// Issue #9: weighted gap / eddy proximity
	EddyProximity map[string]float64 // input id -> proximity

	// Issue #10: bridge centrality
	TopBridge string

	// Issue #11: sediment partitions
	DirtyOnlyDomains []string
	SharpPartition   bool

	// Issue #14: programme merge
	BridgeFidelity1 float64 // shared Tier 1 anchor bridge fidelity
}

// Fixtures is the indexed table consumed by tests in package compute.
//
// qbp_v3_2.json is intentionally absent: the existing QBP v3.2
// inventory at QBP-Compute-Unit/cth/testdata/qbp_v3_2.json is a
// pre-v0.2 prototype that does not match the v0.2 schema (lowercase
// provenance values, ChainRef.chain_id null entries, missing
// statement/name fields, DerivedPrinciple ids without the DERIV-
// prefix). A follow-up issue will convert it.
var Fixtures = map[string]KnownValues{
	// Filled incrementally as compute lands.
	"minimal.json":          {},
	"qbp_quantum_v0_1.json": {},
	"qbp_quantum_v0_2.json": {},
}
