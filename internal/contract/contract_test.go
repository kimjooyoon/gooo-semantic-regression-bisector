package contract

import (
	"strings"
	"testing"
)

func TestParseSourceRejectsDuplicateDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "program",
			source: "program semantic_regression_bisector version \"0.1.0\"\nprogram semantic_regression_bisector version \"0.1.0\"\n",
			want:   "duplicates the program declaration",
		},
		{
			name:   "owns",
			source: "owns candidate_ordering = gooo\nowns candidate_ordering = gooo\n",
			want:   "duplicates owns declaration",
		},
		{
			name:   "rule",
			source: "rule candidate_ordering = ordinal\nrule candidate_ordering = ordinal\n",
			want:   "duplicates rule declaration",
		},
		{
			name:   "canonical_case",
			source: "canonical_case clean_boundary = pass\ncanonical_case clean_boundary = pass\n",
			want:   "duplicates canonical_case declaration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSource(tt.source)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseSource() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
