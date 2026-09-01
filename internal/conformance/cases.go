package conformance

import (
	"fmt"
	"strings"

	"github.com/kimjooyoon/gooo-semantic-regression-bisector/internal/bisector"
)

type CanonicalCase struct {
	Name          string
	Manifest      bisector.Manifest
	Observations  []bisector.Observation
	Decision      string
	Class         string
}

func CanonicalCases() []CanonicalCase {
	return []CanonicalCase{
		cleanBoundary(), replayBoundary(), duplicateEquivalentReceipt(),
		missingMidpoint(), staleReceipt(), ambiguousOrdering(),
		nonMonotonicContradiction(), digestMismatch(), falseMinimalBoundary(),
	}
}

func baseManifest() bisector.Manifest {
	manifest := bisector.Manifest{
		SchemaVersion:  bisector.SchemaVersion,
		InventoryExact: 5,
		KnownGood:      "c0",
		KnownRefuted:   "c4",
		Source: bisector.SourcePin{
			Repository: "https://github.com/example/semantic-source",
			Tag:        bisector.TagPin{Name: "v0.1.0", Object: objectDigest(1), Target: objectDigest(2)},
			Release:    bisector.ReleasePin{ID: 9001, TagName: "v0.1.0", Immutable: true, Digest: digest(3)},
			Asset:      bisector.AssetPin{Name: "manifest.json", Size: 17, Digest: digest(4)},
		},
	}
	for i := 0; i < 5; i++ {
		manifest.Candidates = append(manifest.Candidates, bisector.Candidate{
			ID: iD(i), Ordinal: i, ReleaseID: int64(1000 + i),
			Tag: bisector.TagPin{Name: fmt.Sprintf("candidate-%d", i), Object: objectDigest(10 + i), Target: objectDigest(20 + i)},
			Asset: bisector.AssetPin{Name: "semantic.json", Size: int64(30 + i), Digest: digest(40 + i)},
			StateDigest: digest(50 + i), ClaimDigest: digest(60 + i),
		})
	}
	return manifest
}

func receipt(manifest bisector.Manifest, candidateID, result, key string) bisector.Observation {
	for _, candidate := range manifest.Candidates {
		if candidate.ID == candidateID {
			return bisector.Observation{
				ReceiptID: "r-" + candidateID + "-" + key, CandidateID: candidateID, SchemaVersion: bisector.SchemaVersion,
				ReleaseID: candidate.ReleaseID, TagName: candidate.Tag.Name, TagObject: candidate.Tag.Object, TagTarget: candidate.Tag.Target,
				AssetName: candidate.Asset.Name, AssetSize: candidate.Asset.Size, AssetDigest: candidate.Asset.Digest,
				StateDigest: candidate.StateDigest, ClaimDigest: candidate.ClaimDigest, Result: result,
				EquivalenceKey: key, ObservedAt: "2026-09-01T00:00:00Z",
			}
		}
	}
	panic("unknown candidate " + candidateID)
}

func cleanBoundary() CanonicalCase {
	manifest := baseManifest()
	return CanonicalCase{Name: "clean_boundary", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), receipt(manifest, "c2", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionClosed, Class: "clean_boundary"}
}

func replayBoundary() CanonicalCase {
	manifest := baseManifest()
	replay := receipt(manifest, "c1", bisector.ResultPass, "replay")
	replay.ReplayOf = "r-c1-pass"
	return CanonicalCase{Name: "replay_boundary", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), replay, receipt(manifest, "c2", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionClosed, Class: "replay_boundary"}
}

func duplicateEquivalentReceipt() CanonicalCase {
	manifest := baseManifest()
	duplicate := receipt(manifest, "c1", bisector.ResultPass, "same-proof")
	duplicate.ReceiptID = "r-c1-duplicate"
	return CanonicalCase{Name: "duplicate_equivalent_receipt", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "same-proof"), duplicate, receipt(manifest, "c2", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionClosed, Class: "duplicate_equivalent_receipt"}
}

func missingMidpoint() CanonicalCase {
	manifest := baseManifest()
	return CanonicalCase{Name: "missing_midpoint", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), receipt(manifest, "c3", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionUnknown, Class: "missing_midpoint"}
}

func staleReceipt() CanonicalCase {
	manifest := baseManifest()
	stale := receipt(manifest, "c2", bisector.ResultRefuted, "stale")
	stale.TagObject = objectDigest(99)
	return CanonicalCase{Name: "stale_receipt", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), stale, receipt(manifest, "c3", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionUnknown, Class: "stale_receipt"}
}

func ambiguousOrdering() CanonicalCase {
	manifest := baseManifest()
	manifest.Candidates[2].Ordinal = 3
	return CanonicalCase{Name: "ambiguous_ordering", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), receipt(manifest, "c2", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionUnknown, Class: "ambiguous_ordering"}
}

func nonMonotonicContradiction() CanonicalCase {
	manifest := baseManifest()
	return CanonicalCase{Name: "non_monotonic_contradiction", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), receipt(manifest, "c2", bisector.ResultRefuted, "refuted"), receipt(manifest, "c3", bisector.ResultPass, "pass"),
	}, Decision: bisector.DecisionRefuted, Class: "non_monotonic_contradiction"}
}

func digestMismatch() CanonicalCase {
	manifest := baseManifest()
	mismatched := receipt(manifest, "c2", bisector.ResultRefuted, "bad-digest")
	mismatched.StateDigest = digest(199)
	return CanonicalCase{Name: "digest_mismatch", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c1", bisector.ResultPass, "pass"), mismatched,
	}, Decision: bisector.DecisionRefuted, Class: "digest_mismatch"}
}

func falseMinimalBoundary() CanonicalCase {
	manifest := baseManifest()
	return CanonicalCase{Name: "false_minimal_boundary", Manifest: manifest, Observations: []bisector.Observation{
		receipt(manifest, "c0", bisector.ResultRefuted, "bad-anchor"), receipt(manifest, "c1", bisector.ResultPass, "pass"),
		receipt(manifest, "c2", bisector.ResultRefuted, "refuted"),
	}, Decision: bisector.DecisionRefuted, Class: "false_minimal_boundary"}
}

func iD(index int) string {
	return fmt.Sprintf("c%d", index)
}

func digest(index int) string {
	return "sha256:" + strings.Repeat(fmt.Sprintf("%x", index%16), 64)
}

func objectDigest(index int) string {
	return strings.Repeat(fmt.Sprintf("%x", index%16), 40)
}
