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

	testProgramme = "synthetic"
	testVersion   = "0.0.1"
)
