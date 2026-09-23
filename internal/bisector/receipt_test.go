package bisector

import "testing"

func TestNormalizeReceiptsRejectsDuplicateReceiptIDs(t *testing.T) {
	candidate := Candidate{
		ID: "candidate", ReleaseID: 1,
		Tag: TagPin{Name: "v1.0.0", Object: "tag-object", Target: "tag-target"},
		Asset: AssetPin{Name: "asset", Size: 1, Digest: "asset-digest"},
		StateDigest: "state-digest", ClaimDigest: "claim-digest",
	}
	observation := func(equivalenceKey string) Observation {
		return Observation{
			ReceiptID: "receipt-1", CandidateID: candidate.ID, SchemaVersion: SchemaVersion,
			ReleaseID: candidate.ReleaseID, TagName: candidate.Tag.Name, TagObject: candidate.Tag.Object, TagTarget: candidate.Tag.Target,
			AssetName: candidate.Asset.Name, AssetSize: candidate.Asset.Size, AssetDigest: candidate.Asset.Digest,
			StateDigest: candidate.StateDigest, ClaimDigest: candidate.ClaimDigest,
			Result: ResultPass, EquivalenceKey: equivalenceKey,
		}
	}
	ledger := normalizeReceipts(Manifest{Candidates: []Candidate{candidate}}, []Candidate{candidate}, []Observation{observation("first"), observation("replay")})
	if len(ledger) != 2 || ledger[0].Admissibility != Admissible || ledger[1].Admissibility != DuplicateReceipt {
		t.Fatalf("ledger = %#v", ledger)
	}
}
