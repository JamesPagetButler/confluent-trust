package compute

// Shared test-scope constants. CI's golangci-lint goconst pass flags any
// string literal that appears 3+ times across the package's tests, so
// any value that more than two test files want is hoisted here.
const (
	testInputType   = "input"
	testInputStatus = "measurable"

	testAxiomID  = "AX-1"
	testAnchorM1 = "M-1"
	testAnchorM2 = "M-2"

	testInputShared = "INST-shared"
	testInputA      = "INST-a"
	testInputB      = "INST-b"

	testChainCheap  = "C-cheap"
	testChainDirect = "C-direct"
	testChainC1     = "C-1"
	testChainC2     = "C-2"
	testAxiomShadow = "AXIOM-shadow"

	testProgramme = "synthetic"
	testVersion   = "0.0.1"

	// Categorical label pair used by scoring tests.
	testCatA = "A"
	testCatB = "B"

	// Canonical regime name strings used by scoring tests.
	scoreRegimeStrLaminar     = "laminar"
	scoreRegimeStrLowSediment = "low_sediment"
	scoreRegimeStrModerate    = "moderate"
	scoreRegimeStrHeavy       = "heavy"
)
