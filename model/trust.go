package model

import (
	"encoding/json"
	"fmt"
)

// AxisTrust holds per-axis trust scores for an NT_CTH_ANCHOR.
// Each field is in [0.0, 1.0]; omitempty allows partial population
// (axes not yet scored are absent in JSON). Zero values are valid
// (un-scored is expressed by omitting the enclosing AxisTrust).
//
// Axis semantics per inter#41 Decision 2:
//   - Reproducibility: how reproducible is the supporting evidence?
//   - Theory: how strong is the theoretical grounding?
//   - Stats: how adequate is the statistical treatment?
//   - Method: how sound is the methodology?
//   - Independence: how independent are the contributing labs/perspectives?
type AxisTrust struct {
	Reproducibility *float64 `json:"reproducibility,omitempty"`
	Theory          *float64 `json:"theory,omitempty"`
	Stats           *float64 `json:"stats,omitempty"`
	Method          *float64 `json:"method,omitempty"`
	Independence    *float64 `json:"independence,omitempty"`
}

// Validate enforces that all present axis values are in [0.0, 1.0].
func (a AxisTrust) Validate() error {
	type namedAxis struct {
		val  *float64
		name string
	}
	axes := []namedAxis{
		{val: a.Reproducibility, name: "reproducibility"},
		{val: a.Theory, name: "theory"},
		{val: a.Stats, name: "stats"},
		{val: a.Method, name: "method"},
		{val: a.Independence, name: "independence"},
	}
	for _, ax := range axes {
		if ax.val != nil && (*ax.val < 0.0 || *ax.val > 1.0) {
			return fmt.Errorf("axis_trust.%s: value %g out of [0.0, 1.0]", ax.name, *ax.val)
		}
	}
	return nil
}

// ---- ClusterState ----

// ClusterState classifies the coverage state of a trust cluster.
// States are coverage-based (sheaf coverage), not count-based per inter#41 Decision 2.
type ClusterState uint8

// ClusterState values.
//
//   - ClusterStateNascent: anchors from one lab or one perspective; independence section weak.
//   - ClusterStateDeveloping: anchors from multiple independent labs; independence section improving.
//   - ClusterStateConfluent: global section exists across all axes above threshold.
const (
	ClusterStateUnknown    ClusterState = iota
	ClusterStateNascent                 // one lab / one perspective; independence weak
	ClusterStateDeveloping              // multiple independent labs; independence improving
	ClusterStateConfluent               // global section across all axes above threshold
)

// String returns the canonical JSON string form of a ClusterState.
func (c ClusterState) String() string {
	switch c {
	case ClusterStateNascent:
		return "nascent"
	case ClusterStateDeveloping:
		return "developing"
	case ClusterStateConfluent:
		return "confluent"
	default:
		return ""
	}
}

// MarshalJSON encodes a ClusterState as its canonical string, or null when unknown.
func (c ClusterState) MarshalJSON() ([]byte, error) {
	v := c.String()
	if v == "" {
		return []byte(`null`), nil
	}
	return json.Marshal(v)
}

// UnmarshalJSON decodes a ClusterState from its canonical string.
// Unknown values produce a wrapped error; null and empty string become ClusterStateUnknown.
func (c *ClusterState) UnmarshalJSON(b []byte) error {
	if string(b) == jsonNull {
		*c = ClusterStateUnknown
		return nil
	}
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("cluster_state: %w", err)
	}
	switch raw {
	case "":
		*c = ClusterStateUnknown
	case "nascent":
		*c = ClusterStateNascent
	case "developing":
		*c = ClusterStateDeveloping
	case "confluent":
		*c = ClusterStateConfluent
	default:
		return fmt.Errorf("cluster_state: unknown value %q", raw)
	}
	return nil
}
