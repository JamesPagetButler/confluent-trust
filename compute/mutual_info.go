package compute

import "math"

// epsilonRegularization is the ε > 0 from §4.5 that prevents the pairwise
// MI formula from diverging at perfect agreement (t_a == t_b). It is small
// enough that it does not perturb meaningful disagreements but large
// enough to produce finite MI values when paths agree exactly.
const epsilonRegularization = 1e-12

// PairwiseMI returns the §4.5 Gaussian pairwise mutual information between
// two estimators (predA ± sigmaA) and (predB ± sigmaB) of a common target:
//
//	I(R_a; R_b) = (1/2) log2 (1 + (σ_a² + σ_b²) / ((t_a - t_b)² + ε))
//
// Per the §4.5 remark, ε regularizes the divergence at perfect agreement.
// Negative or zero sigmas are treated as zero noise (degenerate channel).
func PairwiseMI(predA, predB, sigmaA, sigmaB float64) float64 {
	signal := sigmaA*sigmaA + sigmaB*sigmaB
	gap := predA - predB
	noise := gap*gap + epsilonRegularization
	if signal <= 0 || noise <= 0 {
		return 0.0
	}
	return 0.5 * math.Log2(1.0+signal/noise)
}

// NaryMI returns the multivariate mutual information between N Gaussian
// estimators of a common target per Definition 10. The implementation:
//
//  1. Sums all pairwise contributions (each path contributes to MI with
//     every other; this naturally penalises disagreement).
//  2. For N ≥ 3, adds a synergy bonus reflecting how well the full set
//     agrees on a common consensus (chi-squared from the precision-weighted
//     mean). When all paths cluster tightly, this bonus pushes N-ary MI
//     above the pairwise sum (the §2.5 synergy property; "3-way > sum of
//     pairwise"). When paths disagree, chi-squared is large and the bonus
//     decays, so the result behaves continuously.
//
// predictions and sigmas must have the same length. predictions of length
// 0 or 1 return 0 (an MI between fewer than two channels is undefined).
func NaryMI(predictions, sigmas []float64) float64 {
	n := len(predictions)
	if n != len(sigmas) || n < 2 {
		return 0.0
	}

	var pairwiseSum float64
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairwiseSum += PairwiseMI(predictions[i], predictions[j], sigmas[i], sigmas[j])
		}
	}
	if n == 2 {
		return pairwiseSum
	}

	// Precision-weighted consensus mean and chi-squared residual.
	var totalPrecision, weighted float64
	for i := range predictions {
		s := sigmas[i]
		if s <= 0 {
			s = math.SmallestNonzeroFloat64
		}
		precision := 1.0 / (s * s)
		totalPrecision += precision
		weighted += predictions[i] * precision
	}
	if totalPrecision <= 0 {
		return pairwiseSum
	}
	mean := weighted / totalPrecision
	var chiSq float64
	for i := range predictions {
		s := sigmas[i]
		if s <= 0 {
			s = math.SmallestNonzeroFloat64
		}
		d := predictions[i] - mean
		chiSq += d * d / (s * s)
	}

	// Synergy bonus: extra bits per additional path beyond two, decaying
	// with chi-squared. At chi² = 0 (perfect N-way agreement) the bonus is
	// 0.5 * log2(1 + N/ε) per extra path, which is finite but large; at
	// chi² >> N it tends to zero.
	synergyPerExtraPath := 0.5 * math.Log2(1.0+float64(n)/(chiSq+epsilonRegularization))
	if synergyPerExtraPath < 0 {
		synergyPerExtraPath = 0
	}
	bonus := float64(n-2) * synergyPerExtraPath
	return pairwiseSum + bonus
}

// CappedMI applies the channel-capacity cap from Definition 10a:
//
//	I_capped = min(MI, min_i C(R_i))
//
// chainCapacities is the list of per-chain capacities in bits. An empty
// slice disables the cap and returns the raw MI.
func CappedMI(mi float64, chainCapacities []float64) float64 {
	if len(chainCapacities) == 0 {
		return mi
	}
	minCap := chainCapacities[0]
	for _, c := range chainCapacities[1:] {
		if c < minCap {
			minCap = c
		}
	}
	if mi < minCap {
		return mi
	}
	return minCap
}

// StructuralMI returns the MI for a structural N-ary confluence (yes/no
// or integer-quantum-number predictions where each agreeing path
// contributes one bit of confirmatory information):
//
//	I_structural = min(arity, min capacity)
//
// arity is the number of paths that arrived at the same structural
// answer; minCapacity is the minimum channel capacity in bits across
// those paths. Pass +Inf (or any value ≥ arity) to disable the cap.
func StructuralMI(arity int, minCapacity float64) float64 {
	if arity <= 0 {
		return 0
	}
	a := float64(arity)
	if minCapacity < a {
		return minCapacity
	}
	return a
}
