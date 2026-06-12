package compute

import "github.com/JamesPagetButler/confluent-trust/model"

// GlueAxisTrust applies axis-specific sheaf gluing operations over a slice of
// AxisTrust sections, returning the combined AxisTrust for the cluster.
//
// Gluing semantics per inter#41 Decision 2:
//
//	reproducibility — meet (infimum): one weak joint poisons the claim.
//	theory          — join (supremum): any strong theoretical anchor elevates.
//	stats           — meet: conservative.
//	method          — meet: conservative.
//	independence    — meet: most conservative axis.
//
// An axis present in fewer than all sections still participates; absent
// (nil) values are skipped. An axis with no sections populated returns nil
// in the result (the section is absent/unscored).
//
// The caller is responsible for passing non-empty sections; an empty input
// returns a zero AxisTrust with all axes nil.
func GlueAxisTrust(sections []model.AxisTrust) model.AxisTrust {
	var out model.AxisTrust
	for _, s := range sections {
		out.Reproducibility = glueAxis(out.Reproducibility, s.Reproducibility, false) // meet
		out.Theory = glueAxis(out.Theory, s.Theory, true)                             // join
		out.Stats = glueAxis(out.Stats, s.Stats, false)                               // meet
		out.Method = glueAxis(out.Method, s.Method, false)                            // meet
		out.Independence = glueAxis(out.Independence, s.Independence, false)          // meet
	}
	return out
}

// glueAxis folds a new per-section axis value (next) into the accumulated
// value (acc). join=true computes supremum; join=false computes infimum.
// A nil pointer means the axis is absent in that section and is skipped.
func glueAxis(acc, next *float64, join bool) *float64 {
	if next == nil {
		return acc
	}
	if acc == nil {
		v := *next
		return &v
	}
	var v float64
	if join {
		// supremum
		if *next > *acc {
			v = *next
		} else {
			v = *acc
		}
	} else {
		// infimum
		if *next < *acc {
			v = *next
		} else {
			v = *acc
		}
	}
	return &v
}

// ClusterStateFromAxes derives the coverage-based ClusterState from a
// glued AxisTrust section. The transition rules follow inter#41 Decision 2:
//
//	NASCENT     — independence axis absent or below threshold; or fewer than
//	              two axes are above threshold across the section.
//	DEVELOPING  — independence axis above threshold but at least one of the
//	              four other axes is still below threshold (or absent).
//	CONFLUENT   — global section: independence AND all four other axes are
//	              above model.ClusterStateThreshold.
//
// The threshold is model.clusterStateThreshold (0.7) — package-level constant,
// accessible here through the exported helper AboveClusterThreshold.
func ClusterStateFromAxes(at model.AxisTrust) model.ClusterState {
	// Independence is the sentinel axis: weak independence = NASCENT regardless
	// of the other axes (inter#41 note: "one lab produces weak independence section").
	if !aboveThreshold(at.Independence) {
		return model.ClusterStateNascent
	}

	// Independence is above threshold. Check the remaining four axes.
	allAbove := aboveThreshold(at.Reproducibility) &&
		aboveThreshold(at.Theory) &&
		aboveThreshold(at.Stats) &&
		aboveThreshold(at.Method)
	if allAbove {
		return model.ClusterStateConfluent
	}
	return model.ClusterStateDeveloping
}

// aboveThreshold reports whether val is non-nil and >= clusterStateThreshold.
func aboveThreshold(val *float64) bool {
	return val != nil && *val >= clusterThreshold
}

// clusterThreshold is the per-axis coverage threshold for cluster state
// transitions. Mirrors model.clusterStateThreshold (unexported); kept here
// so compute does not import model constants by name.
const clusterThreshold = 0.7
