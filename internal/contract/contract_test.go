package contract

import "testing"

func TestParseRejectsDuplicateGoooDeclarations(t *testing.T) {
	cases := []struct {
		name      string
		duplicate string
	}{
		{name: "program", duplicate: `program semantic_regression_bisector version "0.1.0"`},
		{name: "ownership", duplicate: "owns candidate_ordering = gooo"},
		{name: "rule", duplicate: "rule candidate_ordering = ordinal_then_candidate_id"},
		{name: "activity", duplicate: "activity validate_manifest exactly_once"},
		{name: "canonical_case", duplicate: "canonical_case clean_boundary = CLOSED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parse(source + "\n" + tc.duplicate + "\n"); err == nil {
				t.Fatalf("parse accepted duplicate %s declaration", tc.name)
			}
		})
	}
}
