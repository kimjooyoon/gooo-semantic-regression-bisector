package bisector

import (
	"testing"

	"github.com/kimjooyoon/gooo-semantic-regression-bisector/internal/contract"
)

func TestContractActivityTraceIsExactlyOnce(t *testing.T) {
	program, err := contract.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Activities) != 9 {
		t.Fatalf("activity count = %d, want 9", len(program.Activities))
	}
	seen := map[string]int{}
	for _, activity := range program.Activities {
		seen[activity]++
	}
	for activity, count := range seen {
		if count != 1 {
			t.Fatalf("activity %q appears %d times", activity, count)
		}
	}
}

func TestDigestAndObjectPinValidation(t *testing.T) {
	if !isDigest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("expected valid sha256 digest")
	}
	if isDigest("sha256:not-a-digest") {
		t.Fatal("expected invalid digest")
	}
	if !objectID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("expected valid object id")
	}
}

func TestEmptyReceiptIDIsNotAdmissible(t *testing.T) {
	ledger := normalizeReceipts(
		Manifest{Candidates: []Candidate{{ID: "candidate"}}},
		[]Candidate{{ID: "candidate"}},
		[]Observation{{CandidateID: "candidate", Result: ResultPass}},
	)
	if len(ledger) != 1 || ledger[0].Admissibility != DigestMismatch {
		t.Fatalf("empty receipt id was admitted: %#v", ledger)
	}
}
