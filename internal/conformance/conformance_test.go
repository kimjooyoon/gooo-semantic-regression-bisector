package conformance

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kimjooyoon/gooo-semantic-regression-bisector/internal/bisector"
	"github.com/kimjooyoon/gooo-semantic-regression-bisector/internal/contract"
)

func TestConformance(t *testing.T) {
	program, err := contract.Load()
	if err != nil {
		t.Fatal(err)
	}
	cases := CanonicalCases()
	if len(cases) != 9 {
		t.Fatalf("canonical case count = %d, want 9", len(cases))
	}
	for _, canonical := range cases {
		canonical := canonical
		t.Run(canonical.Name, func(t *testing.T) {
			result, err := bisector.Evaluate(canonical.Manifest, canonical.Observations, program)
			if err != nil {
				t.Fatal(err)
			}
			if result.Boundary.Decision != canonical.Decision || result.Boundary.Class != canonical.Class {
				t.Fatalf("decision = %s/%s, want %s/%s", result.Boundary.Decision, result.Boundary.Class, canonical.Decision, canonical.Class)
			}
			if len(result.Events) != 9 {
				t.Fatalf("activity events = %d, want 9", len(result.Events))
			}
			seen := map[string]bool{}
			for _, event := range result.Events {
				if seen[event.Activity] {
					t.Fatalf("activity %q was executed more than once", event.Activity)
				}
				seen[event.Activity] = true
			}
			if canonical.Name == "duplicate_equivalent_receipt" {
				found := false
				for _, entry := range result.Ledger {
					if entry.Admissibility == bisector.DuplicateReceipt {
						found = true
					}
				}
				if !found {
					t.Fatal("equivalent duplicate was not retained in the ledger")
				}
			}
		})
	}
}

func TestIntegrationOutputsExactlySixCallerOwnedFiles(t *testing.T) {
	program, err := contract.Load()
	if err != nil {
		t.Fatal(err)
	}
	result, err := bisector.Evaluate(CanonicalCases()[0].Manifest, CanonicalCases()[0].Observations, program)
	if err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	if err := bisector.WriteOutputs(outputDir, result); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	sort.Strings(got)
	want := []string{"bisect-manifest.json", "boundary-receipt.json", "candidate-events.ndjson", "observation-ledger.ndjson", "replay-receipt.json", "report.md"}
	if len(got) != len(want) {
		t.Fatalf("output file count = %d, want %d (%v)", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("output files = %v, want %v", got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(outputDir, "output.json")); !os.IsNotExist(err) {
		t.Fatal("unexpected aggregate output file")
	}
}
