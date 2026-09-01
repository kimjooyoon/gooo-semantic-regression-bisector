package bisector

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-semantic-regression-bisector/internal/contract"
)

const SchemaVersion = "gooo.semantic.bisect/v1"

const (
	DecisionClosed   = "CLOSED"
	DecisionUnknown  = "UNKNOWN"
	DecisionRefuted  = "REFUTED"
	ResultPass       = "PASS"
	ResultRefuted    = "REFUTED"
	ResultUnknown    = "UNKNOWN"
	Admissible       = "ADMISSIBLE"
	DuplicateReceipt = "DUPLICATE_EQUIVALENT"
	StaleReceipt     = "STALE"
	DigestMismatch   = "DIGEST_MISMATCH"
)

type TagPin struct {
	Name   string `json:"name"`
	Object string `json:"object"`
	Target string `json:"target"`
}

type ReleasePin struct {
	ID        int64  `json:"id"`
	TagName   string `json:"tag_name"`
	Immutable bool   `json:"immutable"`
	Digest    string `json:"digest"`
}

type AssetPin struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type SourcePin struct {
	Repository string      `json:"repository"`
	Tag        TagPin      `json:"tag"`
	Release    ReleasePin  `json:"release"`
	Asset      AssetPin    `json:"asset"`
}

type Candidate struct {
	ID          string   `json:"id"`
	Ordinal     int      `json:"ordinal"`
	ReleaseID   int64    `json:"release_id"`
	Tag         TagPin   `json:"tag"`
	Asset       AssetPin `json:"asset"`
	StateDigest string   `json:"state_digest"`
	ClaimDigest string   `json:"claim_digest"`
}

type Manifest struct {
	SchemaVersion string      `json:"schema_version"`
	Source        SourcePin   `json:"source"`
	InventoryExact int         `json:"inventory_exact"`
	Candidates    []Candidate `json:"candidates"`
	KnownGood     string      `json:"known_good"`
	KnownRefuted  string      `json:"known_refuted"`
}

type Observation struct {
	ReceiptID       string `json:"receipt_id"`
	CandidateID     string `json:"candidate_id"`
	SchemaVersion   string `json:"schema_version"`
	ReleaseID       int64  `json:"release_id"`
	TagName         string `json:"tag_name"`
	TagObject       string `json:"tag_object"`
	TagTarget       string `json:"tag_target"`
	AssetName       string `json:"asset_name"`
	AssetSize       int64  `json:"asset_size"`
	AssetDigest     string `json:"asset_digest"`
	StateDigest     string `json:"state_digest"`
	ClaimDigest     string `json:"claim_digest"`
	Result          string `json:"result"`
	EquivalenceKey  string `json:"equivalence_key"`
	ReplayOf        string `json:"replay_of,omitempty"`
	ObservedAt      string `json:"observed_at"`
}

type LedgerEntry struct {
	Sequence        int    `json:"sequence"`
	ReceiptID       string `json:"receipt_id"`
	CandidateID     string `json:"candidate_id"`
	EquivalenceKey  string `json:"equivalence_key"`
	SuppliedResult  string `json:"supplied_result"`
	Admissibility   string `json:"admissibility"`
	EffectiveResult string `json:"effective_result,omitempty"`
	FailureKind     string `json:"failure_kind,omitempty"`
	Reason          string `json:"reason,omitempty"`
	ReplayOf        string `json:"replay_of,omitempty"`
}

type CandidateEvent struct {
	Sequence     int      `json:"sequence"`
	Activity     string   `json:"activity"`
	Stage        string   `json:"stage"`
	Step         string   `json:"step"`
	Status       string   `json:"status"`
	Class        string   `json:"class"`
	Reason       string   `json:"reason"`
	NextOperation string   `json:"next_operation"`
	BlockedBy    []string `json:"blocked_by"`
	CandidateIDs []string `json:"candidate_ids,omitempty"`
	ReceiptIDs   []string `json:"receipt_ids,omitempty"`
}

type CandidateSummary struct {
	ID          string `json:"id"`
	Ordinal     int    `json:"ordinal"`
	Status      string `json:"status"`
	Class       string `json:"class"`
	ObservationCount int `json:"observation_count"`
}

type ManifestOutput struct {
	SchemaVersion       string             `json:"schema_version"`
	ContractVersion     string             `json:"contract_version"`
	ContractSourceDigest string            `json:"contract_source_digest"`
	Repository          string             `json:"repository"`
	SourceTag           TagPin             `json:"source_tag"`
	SourceRelease       ReleasePin         `json:"source_release"`
	SourceAsset         AssetPin           `json:"source_asset"`
	InventoryExact      int                `json:"inventory_exact"`
	KnownGood           string             `json:"known_good"`
	KnownRefuted        string             `json:"known_refuted"`
	OrderedCandidates   []CandidateSummary `json:"ordered_candidates"`
	Policy              map[string]string  `json:"policy"`
}

type BoundaryReceipt struct {
	SchemaVersion   string      `json:"schema_version"`
	Status          string      `json:"status"`
	Decision        string      `json:"decision"`
	Class           string      `json:"class"`
	Stage           string      `json:"stage"`
	Step            string      `json:"step"`
	Reason          string      `json:"reason"`
	NextOperation   string      `json:"next_operation"`
	BlockedBy       []string    `json:"blocked_by"`
	LowerCandidate  string      `json:"lower_candidate,omitempty"`
	UpperCandidate  string      `json:"upper_candidate,omitempty"`
	LowerStatus     string      `json:"lower_status,omitempty"`
	UpperStatus     string      `json:"upper_status,omitempty"`
	FailureKind     string      `json:"failure_kind,omitempty"`
	Repository      string      `json:"repository"`
	SourceTag       TagPin      `json:"source_tag"`
	SourceRelease   ReleasePin  `json:"source_release"`
	SourceAsset     AssetPin    `json:"source_asset"`
	ReceiptDigest   string      `json:"receipt_digest,omitempty"`
}

type ReplayReceipt struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Deterministic bool   `json:"deterministic"`
	ActivityCount int    `json:"activity_count"`
	InputDigest   string `json:"input_digest"`
	BoundaryDigest string `json:"boundary_digest"`
	Decision      string `json:"decision"`
	Class         string `json:"class"`
}

type Result struct {
	ManifestOutput ManifestOutput
	Events         []CandidateEvent
	Ledger         []LedgerEntry
	Boundary       BoundaryReceipt
	Replay         ReplayReceipt
	Report         string
}

type validationOutcome struct {
	Valid       bool
	Class       string
	Reason      string
	BlockedBy   []string
}

type candidateState struct {
	Status          string
	Class           string
	Reason          string
	ObservationCount int
	HasStale        bool
	StaleReceipts   []string
}

type evaluation struct {
	manifest       Manifest
	ordered        []Candidate
	program        contract.Program
	validation     validationOutcome
	ledger         []LedgerEntry
	states         map[string]candidateState
	events         []CandidateEvent
	globalDecision string
	globalClass    string
	globalStage    string
	globalStep     string
	globalReason   string
	globalNext     string
	globalBlocked  []string
	globalFailure  string
}

func Evaluate(manifest Manifest, observations []Observation, program contract.Program) (Result, error) {
	if err := program.Validate(); err != nil {
		return Result{}, err
	}
	e := &evaluation{
		manifest: manifest,
		program:  program,
		states:   map[string]candidateState{},
	}

	e.validation = validateManifest(manifest)
	e.addActivity("validate_manifest", e.validation.Class, e.validation.Reason, "validate_manifest", e.validation.BlockedBy, nil, nil)

	e.ordered = append([]Candidate(nil), manifest.Candidates...)
	sort.SliceStable(e.ordered, func(i, j int) bool {
		if e.ordered[i].Ordinal != e.ordered[j].Ordinal {
			return e.ordered[i].Ordinal < e.ordered[j].Ordinal
		}
		return e.ordered[i].ID < e.ordered[j].ID
	})
	orderedIDs := candidateIDs(e.ordered)
	e.addActivity("order_candidates", "ordered", "candidate order is ordinal then candidate id", "order_candidates", nil, orderedIDs, nil)

	orderedObservations := append([]Observation(nil), observations...)
	sort.SliceStable(orderedObservations, func(i, j int) bool {
		return orderedObservations[i].ReceiptID < orderedObservations[j].ReceiptID
	})
	receiptIDs := make([]string, 0, len(orderedObservations))
	for _, observation := range orderedObservations {
		receiptIDs = append(receiptIDs, observation.ReceiptID)
	}
	e.addActivity("ingest_observations", "ingest_observations", "supplied observations were read without repository access", "ingest_observations", nil, nil, receiptIDs)

	e.ledger = normalizeReceipts(manifest, e.ordered, orderedObservations)
	e.addActivity("normalize_receipts", "normalize_receipts", "receipt admissibility was checked against pinned fields", "normalize_receipts", nil, nil, ledgerReceiptIDs(e.ledger))

	e.states = classifyCandidates(e.ordered, e.ledger, manifest.KnownGood, manifest.KnownRefuted)
	e.addActivity("classify_candidates", "classify_candidates", "candidate states use REFUTED before UNKNOWN before PASS", "classify_candidates", nil, orderedIDs, nil)

	e.detectContradiction()
	e.addActivity("detect_contradiction", e.globalClassOr("contradiction_scan"), e.globalReasonOr("ordered candidate claims were checked for contradiction"), e.globalStepOr("candidate_ordering"), e.globalBlocked, nil, nil)

	decision := e.narrowInterval()
	e.addActivity("narrow_interval", decision.Class, decision.Reason, decision.Step, decision.BlockedBy, nonEmptyIDs(decision.LowerCandidate, decision.UpperCandidate), nil)

	e.addActivity("materialize_frontier", decision.Class, decision.Reason, decision.Step, decision.BlockedBy, nonEmptyIDs(decision.LowerCandidate, decision.UpperCandidate), nil)
	e.addActivity("emit_receipts", decision.Class, "six caller-owned outputs are ready", "emit_receipts", decision.BlockedBy, nil, nil)

	if len(e.events) != len(program.Activities) {
		return Result{}, fmt.Errorf("executed %d activities, contract requires %d", len(e.events), len(program.Activities))
	}
	manifestOutput := makeManifestOutput(manifest, e.ordered, e.states, program)
	boundary := makeBoundaryReceipt(manifest, decision)
	boundary.ReceiptDigest = digestJSON(boundary)
	inputDigest := digestJSON(struct {
		Manifest     Manifest       `json:"manifest"`
		Observations []Observation  `json:"observations"`
	}{manifest, orderedObservations})
	replay := ReplayReceipt{
		SchemaVersion:  SchemaVersion,
		Status:         boundary.Decision,
		Deterministic:  true,
		ActivityCount:  len(e.events),
		InputDigest:    inputDigest,
		BoundaryDigest: boundary.ReceiptDigest,
		Decision:       boundary.Decision,
		Class:          boundary.Class,
	}
	return Result{
		ManifestOutput: manifestOutput,
		Events:         e.events,
		Ledger:         e.ledger,
		Boundary:       boundary,
		Replay:         replay,
		Report:         makeReport(manifest, e.events, e.ledger, boundary),
	}, nil
}

func validateManifest(manifest Manifest) validationOutcome {
	if manifest.SchemaVersion != SchemaVersion {
		return validationOutcome{Class: "invalid_manifest", Reason: "manifest schema is not supported", BlockedBy: []string{"schema_version"}}
	}
	if manifest.Source.Repository == "" || !manifest.Source.Release.Immutable || manifest.Source.Release.ID <= 0 {
		return validationOutcome{Class: "invalid_manifest", Reason: "repository, release id, and immutable release pin are required", BlockedBy: []string{"source"}}
	}
	if manifest.Source.Release.TagName != manifest.Source.Tag.Name || manifest.Source.Tag.Name == "" || !objectID(manifest.Source.Tag.Object) || !objectID(manifest.Source.Tag.Target) {
		return validationOutcome{Class: "invalid_manifest", Reason: "source release and source tag pins are incomplete", BlockedBy: []string{"source_tag"}}
	}
	if !assetPin(manifest.Source.Asset) || !isDigest(manifest.Source.Release.Digest) {
		return validationOutcome{Class: "invalid_manifest", Reason: "source release and asset digests must be pinned", BlockedBy: []string{"source_asset"}}
	}
	if manifest.InventoryExact != len(manifest.Candidates) || manifest.InventoryExact <= 0 {
		return validationOutcome{Class: "invalid_manifest", Reason: "inventory_exact does not equal the candidate inventory", BlockedBy: []string{"inventory_exact"}}
	}
	ids := map[string]bool{}
	ordinals := map[int]string{}
	for _, candidate := range manifest.Candidates {
		if candidate.ID == "" || ids[candidate.ID] || candidate.ReleaseID <= 0 || !objectID(candidate.Tag.Object) || !objectID(candidate.Tag.Target) || candidate.Tag.Name == "" || !assetPin(candidate.Asset) || !isDigest(candidate.StateDigest) || !isDigest(candidate.ClaimDigest) {
			return validationOutcome{Class: "invalid_manifest", Reason: "candidate pins and digests are incomplete", BlockedBy: []string{candidate.ID}}
		}
		ids[candidate.ID] = true
		if previous, exists := ordinals[candidate.Ordinal]; exists {
			return validationOutcome{Class: "ambiguous_ordering", Reason: "two candidates share an ordinal; no midpoint was inferred", BlockedBy: []string{previous, candidate.ID}}
		}
		ordinals[candidate.Ordinal] = candidate.ID
	}
	if !ids[manifest.KnownGood] || !ids[manifest.KnownRefuted] {
		return validationOutcome{Class: "ambiguous_ordering", Reason: "known-good and known-refuted anchors must be present", BlockedBy: []string{"known_good", "known_refuted"}}
	}
	for ordinal := 0; ordinal < len(manifest.Candidates); ordinal++ {
		if _, exists := ordinals[ordinal]; !exists {
			return validationOutcome{Class: "ambiguous_ordering", Reason: "candidate ordinals are not contiguous; no midpoint was inferred", BlockedBy: []string{"candidate_ordering"}}
		}
	}
	goodOrdinal := -1
	refutedOrdinal := -1
	for _, candidate := range manifest.Candidates {
		if candidate.ID == manifest.KnownGood {
			goodOrdinal = candidate.Ordinal
		}
		if candidate.ID == manifest.KnownRefuted {
			refutedOrdinal = candidate.Ordinal
		}
	}
	if goodOrdinal < 0 || refutedOrdinal < 0 || goodOrdinal >= refutedOrdinal {
		return validationOutcome{Class: "ambiguous_ordering", Reason: "known-good must precede known-refuted", BlockedBy: []string{"known_good", "known_refuted"}}
	}
	return validationOutcome{Valid: true, Class: "valid_manifest", Reason: "manifest inventory and immutable pins are valid", BlockedBy: []string{}}
}

func normalizeReceipts(manifest Manifest, ordered []Candidate, observations []Observation) []LedgerEntry {
	candidates := map[string]Candidate{}
	for _, candidate := range ordered {
		candidates[candidate.ID] = candidate
	}
	seen := map[string]bool{}
	ledger := make([]LedgerEntry, 0, len(observations))
	for index, observation := range observations {
		entry := LedgerEntry{
			Sequence:       index + 1,
			ReceiptID:      observation.ReceiptID,
			CandidateID:    observation.CandidateID,
			EquivalenceKey: observation.EquivalenceKey,
			SuppliedResult: observation.Result,
			ReplayOf:       observation.ReplayOf,
		}
		candidate, exists := candidates[observation.CandidateID]
		if !exists {
			entry.Admissibility = DigestMismatch
			entry.FailureKind = "operational"
			entry.Reason = "receipt references a candidate outside the pinned inventory"
			ledger = append(ledger, entry)
			continue
		}
		if observation.SchemaVersion != SchemaVersion {
			entry.Admissibility = StaleReceipt
			entry.FailureKind = "operational"
			entry.Reason = "receipt schema is stale"
			ledger = append(ledger, entry)
			continue
		}
		if observation.TagName != candidate.Tag.Name || observation.TagObject != candidate.Tag.Object || observation.TagTarget != candidate.Tag.Target || observation.ReleaseID != candidate.ReleaseID {
			entry.Admissibility = StaleReceipt
			entry.FailureKind = "operational"
			entry.Reason = "receipt does not match the candidate's pinned release or tag"
			ledger = append(ledger, entry)
			continue
		}
		if observation.AssetName != candidate.Asset.Name || observation.AssetSize != candidate.Asset.Size || observation.AssetDigest != candidate.Asset.Digest || observation.StateDigest != candidate.StateDigest || observation.ClaimDigest != candidate.ClaimDigest {
			entry.Admissibility = DigestMismatch
			entry.FailureKind = "operational"
			entry.Reason = "receipt asset, state, or claim digest differs from the pinned candidate"
			ledger = append(ledger, entry)
			continue
		}
		if observation.Result != ResultPass && observation.Result != ResultRefuted && observation.Result != ResultUnknown {
			entry.Admissibility = DigestMismatch
			entry.FailureKind = "operational"
			entry.Reason = "receipt result is outside PASS, REFUTED, UNKNOWN"
			ledger = append(ledger, entry)
			continue
		}
		if observation.EquivalenceKey != "" {
			key := observation.CandidateID + "\x00" + observation.EquivalenceKey
			if seen[key] {
				entry.Admissibility = DuplicateReceipt
				entry.Reason = "equivalent receipt retained as replay evidence and not counted twice"
				ledger = append(ledger, entry)
				continue
			}
			seen[key] = true
		}
		entry.Admissibility = Admissible
		entry.EffectiveResult = observation.Result
		if observation.Result == ResultRefuted {
			entry.FailureKind = "semantic"
		}
		ledger = append(ledger, entry)
	}
	return ledger
}

func classifyCandidates(ordered []Candidate, ledger []LedgerEntry, knownGood, knownRefuted string) map[string]candidateState {
	states := map[string]candidateState{}
	for _, candidate := range ordered {
		states[candidate.ID] = candidateState{Status: ResultUnknown, Class: "missing_observation", Reason: "no admissible observation was supplied", StaleReceipts: []string{}}
	}
	for _, entry := range ledger {
		state, exists := states[entry.CandidateID]
		if !exists {
			continue
		}
		if entry.Admissibility == Admissible {
			state.ObservationCount++
			if entry.EffectiveResult == ResultRefuted {
				state.Status = ResultRefuted
				state.Class = "refuted_observation"
				state.Reason = "an admissible REFUTED observation is present"
			} else if entry.EffectiveResult == ResultPass && state.Status != ResultRefuted {
				state.Status = ResultPass
				state.Class = "passed_observation"
				state.Reason = "an admissible PASS observation is present"
			} else if entry.EffectiveResult == ResultUnknown && state.Status == ResultUnknown {
				state.Class = "unknown_observation"
				state.Reason = "an admissible UNKNOWN observation does not narrow the interval"
			}
		}
		if entry.Admissibility == StaleReceipt {
			state.HasStale = true
			state.StaleReceipts = append(state.StaleReceipts, entry.ReceiptID)
		}
		states[entry.CandidateID] = state
	}
	for _, candidate := range ordered {
		state := states[candidate.ID]
		if state.Status != ResultUnknown {
			continue
		}
		if candidate.ID == knownGood {
			state.Status = ResultPass
			state.Class = "trusted_good_anchor"
			state.Reason = "known-good is a supplied trusted anchor"
		}
		if candidate.ID == knownRefuted {
			state.Status = ResultRefuted
			state.Class = "trusted_refuted_anchor"
			state.Reason = "known-refuted is a supplied trusted anchor"
		}
		states[candidate.ID] = state
	}
	return states
}

func (e *evaluation) detectContradiction() {
	if !e.validation.Valid {
		if e.validation.Class == "ambiguous_ordering" {
			e.globalDecision = DecisionUnknown
			e.globalClass = e.validation.Class
			e.globalStage = "validate_manifest"
			e.globalStep = "candidate_ordering"
			e.globalReason = e.validation.Reason
			e.globalNext = "repair_candidate_ordering"
			e.globalBlocked = append([]string{}, e.validation.BlockedBy...)
			e.globalFailure = "operational"
		}
		return
	}
	if hasDigestMismatch(e.ledger) {
		e.globalDecision = DecisionRefuted
		e.globalClass = "digest_mismatch"
		e.globalStage = "normalize_receipts"
		e.globalStep = firstLedgerReason(e.ledger, DigestMismatch)
		e.globalReason = "a supplied receipt has a digest mismatch and cannot support a semantic boundary"
		e.globalNext = "supply_a_receipt_with_matching_pinned_digests"
		e.globalBlocked = []string{e.globalStep}
		e.globalFailure = "operational"
		return
	}
	good := e.states[e.manifest.KnownGood]
	refuted := e.states[e.manifest.KnownRefuted]
	if good.Status == ResultRefuted || refuted.Status == ResultPass || hasAdmissibleResult(e.ledger, e.manifest.KnownGood, ResultRefuted) || hasAdmissibleResult(e.ledger, e.manifest.KnownRefuted, ResultPass) {
		e.globalDecision = DecisionRefuted
		e.globalClass = "false_minimal_boundary"
		e.globalStage = "detect_contradiction"
		e.globalStep = e.manifest.KnownGood + "->" + e.manifest.KnownRefuted
		e.globalReason = "trusted anchors contradict their declared good/refuted roles; a minimal boundary would be false"
	e.globalNext = "revalidate_trusted_anchors"
	e.globalBlocked = []string{e.manifest.KnownGood, e.manifest.KnownRefuted}
	e.globalFailure = "semantic"
		return
	}
	seenRefuted := false
	for _, candidate := range e.ordered {
		state := e.states[candidate.ID]
		if candidate.ID == e.manifest.KnownGood || candidate.ID == e.manifest.KnownRefuted || candidate.Ordinal > candidateOrdinal(e.ordered, e.manifest.KnownGood) && candidate.Ordinal < candidateOrdinal(e.ordered, e.manifest.KnownRefuted) {
			if state.Status == ResultRefuted {
				seenRefuted = true
			}
			if state.Status == ResultPass && seenRefuted {
				e.globalDecision = DecisionRefuted
				e.globalClass = "non_monotonic_contradiction"
				e.globalStage = "detect_contradiction"
				e.globalStep = candidate.ID
				e.globalReason = "a PASS appears after a REFUTED claim in candidate order"
				e.globalNext = "inspect_claim_graph_and_reconcile_observations"
				e.globalBlocked = []string{e.manifest.KnownGood, candidate.ID}
				e.globalFailure = "semantic"
				return
			}
		}
	}
}

type intervalDecision struct {
	Decision       string
	Class          string
	Stage          string
	Step           string
	Reason         string
	NextOperation  string
	BlockedBy      []string
	LowerCandidate string
	UpperCandidate string
	LowerStatus    string
	UpperStatus    string
	FailureKind    string
}

func (e *evaluation) narrowInterval() intervalDecision {
	if e.globalDecision != "" {
		return intervalDecision{
			Decision: e.globalDecision, Class: e.globalClass, Stage: e.globalStage, Step: e.globalStep,
			Reason: e.globalReason, NextOperation: e.globalNext, BlockedBy: append([]string{}, e.globalBlocked...),
			FailureKind: e.globalFailure,
		}
	}
	if !e.validation.Valid {
		return intervalDecision{Decision: DecisionRefuted, Class: "invalid_manifest", Stage: "validate_manifest", Step: "manifest", Reason: e.validation.Reason, NextOperation: "repair_manifest", BlockedBy: e.validation.BlockedBy, FailureKind: "operational"}
	}
	lower := indexOf(e.ordered, e.manifest.KnownGood)
	upper := indexOf(e.ordered, e.manifest.KnownRefuted)
	for upper-lower > 1 {
		mid := lower + (upper-lower)/2
		candidate := e.ordered[mid]
		state := e.states[candidate.ID]
		if state.Status == ResultUnknown {
			class := "missing_midpoint"
			reason := "midpoint has no admissible observation; no semantic inference was made"
			next := "supply_admissible_midpoint_observation"
			blocked := []string{candidate.ID}
			if state.HasStale {
				class = "stale_receipt"
				reason = "midpoint is represented only by a stale receipt; no semantic inference was made"
				next = "replay_midpoint_against_pinned_release_tag_asset"
				blocked = append([]string{}, state.StaleReceipts...)
			}
			return intervalDecision{Decision: DecisionUnknown, Class: class, Stage: "narrow_interval", Step: candidate.ID, Reason: reason, NextOperation: next, BlockedBy: blocked, LowerCandidate: e.ordered[lower].ID, UpperCandidate: e.ordered[upper].ID, LowerStatus: e.states[e.ordered[lower].ID].Status, UpperStatus: e.states[e.ordered[upper].ID].Status, FailureKind: "operational"}
		}
		if state.Status == ResultPass {
			lower = mid
			continue
		}
		if state.Status == ResultRefuted {
			upper = mid
			continue
		}
		return intervalDecision{Decision: DecisionUnknown, Class: "unknown_midpoint", Stage: "narrow_interval", Step: candidate.ID, Reason: "midpoint result is not decision-bearing", NextOperation: "supply_a_decision_bearing_observation", BlockedBy: []string{candidate.ID}, LowerCandidate: e.ordered[lower].ID, UpperCandidate: e.ordered[upper].ID, FailureKind: "operational"}
	}
	class := "clean_boundary"
	if hasReplay(e.ledger) {
		class = "replay_boundary"
	} else if hasDuplicate(e.ledger) {
		class = "duplicate_equivalent_receipt"
	}
	return intervalDecision{
		Decision: DecisionClosed, Class: class, Stage: "narrow_interval", Step: e.ordered[upper].ID,
		Reason: "adjacent PASS and REFUTED candidates form a proven semantic state/claim boundary",
		NextOperation: "publish_boundary_receipt", BlockedBy: []string{}, LowerCandidate: e.ordered[lower].ID,
		UpperCandidate: e.ordered[upper].ID, LowerStatus: e.states[e.ordered[lower].ID].Status,
		UpperStatus: e.states[e.ordered[upper].ID].Status,
	}
}

func makeManifestOutput(manifest Manifest, ordered []Candidate, states map[string]candidateState, program contract.Program) ManifestOutput {
	summaries := make([]CandidateSummary, 0, len(ordered))
	for _, candidate := range ordered {
		state := states[candidate.ID]
		summaries = append(summaries, CandidateSummary{ID: candidate.ID, Ordinal: candidate.Ordinal, Status: state.Status, Class: state.Class, ObservationCount: state.ObservationCount})
	}
	policy := map[string]string{}
	for key, value := range program.Rules {
		policy[key] = value
	}
	return ManifestOutput{
		SchemaVersion: SchemaVersion, ContractVersion: program.Version, ContractSourceDigest: program.SourceDigest,
		Repository: manifest.Source.Repository, SourceTag: manifest.Source.Tag, SourceRelease: manifest.Source.Release,
		SourceAsset: manifest.Source.Asset, InventoryExact: len(ordered), KnownGood: manifest.KnownGood,
		KnownRefuted: manifest.KnownRefuted, OrderedCandidates: summaries, Policy: policy,
	}
}

func makeBoundaryReceipt(manifest Manifest, decision intervalDecision) BoundaryReceipt {
	return BoundaryReceipt{
		SchemaVersion: SchemaVersion, Status: decision.Decision, Decision: decision.Decision, Class: decision.Class,
		Stage: decision.Stage, Step: decision.Step, Reason: decision.Reason,
		NextOperation: decision.NextOperation, BlockedBy: nonNilStrings(decision.BlockedBy),
		LowerCandidate: decision.LowerCandidate, UpperCandidate: decision.UpperCandidate,
		LowerStatus: decision.LowerStatus, UpperStatus: decision.UpperStatus,
		FailureKind: decision.FailureKind, Repository: manifest.Source.Repository,
		SourceTag: manifest.Source.Tag, SourceRelease: manifest.Source.Release, SourceAsset: manifest.Source.Asset,
	}
}

func makeReport(manifest Manifest, events []CandidateEvent, ledger []LedgerEntry, boundary BoundaryReceipt) string {
	admissible, duplicates, stale, mismatches := 0, 0, 0, 0
	for _, entry := range ledger {
		switch entry.Admissibility {
		case Admissible:
			admissible++
		case DuplicateReceipt:
			duplicates++
		case StaleReceipt:
			stale++
		case DigestMismatch:
			mismatches++
		}
	}
	var b strings.Builder
	b.WriteString("# Semantic regression bisect report\n\n")
	fmt.Fprintf(&b, "Decision: `%s`\n\n", boundary.Decision)
	fmt.Fprintf(&b, "Class: `%s`\n\n", boundary.Class)
	fmt.Fprintf(&b, "Inventory exact: `%d` candidates\n\n", manifest.InventoryExact)
	fmt.Fprintf(&b, "Source: `%s`\n\n", manifest.Source.Repository)
	fmt.Fprintf(&b, "Boundary stage: `%s`\n\n", boundary.Stage)
	fmt.Fprintf(&b, "Boundary step: `%s`\n\n", boundary.Step)
	fmt.Fprintf(&b, "Reason: %s\n\n", boundary.Reason)
	fmt.Fprintf(&b, "Next operation: `%s`\n\n", boundary.NextOperation)
	fmt.Fprintf(&b, "Blocked by: `%s`\n\n", strings.Join(boundary.BlockedBy, ", "))
	fmt.Fprintf(&b, "Admissible observations: `%d`\n\n", admissible)
	fmt.Fprintf(&b, "Duplicate equivalent receipts: `%d`\n\n", duplicates)
	fmt.Fprintf(&b, "Stale receipts: `%d`\n\n", stale)
	fmt.Fprintf(&b, "Digest mismatches: `%d`\n\n", mismatches)
	fmt.Fprintf(&b, "Gooo activities executed exactly once: `%d`\n\n", len(events))
	b.WriteString("Operational and semantic evidence remain separate in the observation ledger. The evaluator consumed supplied pins and observations only; it did not inspect or mutate a source repository, tag, release, or asset.\n")
	return b.String()
}

func RunFiles(manifestPath, observationsPath string) (Result, error) {
	manifest, err := readJSONFile[Manifest](manifestPath)
	if err != nil {
		return Result{}, fmt.Errorf("read manifest: %w", err)
	}
	observations, err := readObservations(observationsPath)
	if err != nil {
		return Result{}, fmt.Errorf("read observations: %w", err)
	}
	program, err := contract.Load()
	if err != nil {
		return Result{}, err
	}
	return Evaluate(manifest, observations, program)
}

func WriteOutputs(outputDir string, result Result) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	allowed := map[string]bool{
		"bisect-manifest.json": true, "candidate-events.ndjson": true, "observation-ledger.ndjson": true,
		"boundary-receipt.json": true, "replay-receipt.json": true, "report.md": true,
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			return fmt.Errorf("output directory contains non-caller-owned file %q", entry.Name())
		}
	}
	if err := writeJSONFile(filepath.Join(outputDir, "bisect-manifest.json"), result.ManifestOutput); err != nil {
		return err
	}
	if err := writeNDJSONFile(filepath.Join(outputDir, "candidate-events.ndjson"), result.Events); err != nil {
		return err
	}
	if err := writeNDJSONFile(filepath.Join(outputDir, "observation-ledger.ndjson"), result.Ledger); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outputDir, "boundary-receipt.json"), result.Boundary); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(outputDir, "replay-receipt.json"), result.Replay); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "report.md"), []byte(result.Report), 0o644)
}

func readObservations(path string) ([]Observation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var observations []Observation
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var observation Observation
		if err := json.Unmarshal([]byte(text), &observation); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		observations = append(observations, observation)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return observations, nil
}

func readJSONFile[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, err
	}
	return value, nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeNDJSONFile[T any](path string, values []T) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			return err
		}
	}
	return nil
}

func (e *evaluation) addActivity(activity, class, reason, step string, blocked, candidates, receipts []string) {
	sequence := len(e.events)
	if sequence >= len(e.program.Activities) || e.program.Activities[sequence] != activity {
		panic(fmt.Sprintf("Gooo activity order violation at %d: %s", sequence+1, activity))
	}
	status := DecisionClosed
	if class == "missing_midpoint" || class == "stale_receipt" || class == "ambiguous_ordering" || class == "invalid_manifest" {
		status = DecisionUnknown
	}
	if class == "digest_mismatch" || class == "non_monotonic_contradiction" || class == "false_minimal_boundary" {
		status = DecisionRefuted
	}
	e.events = append(e.events, CandidateEvent{
		Sequence: sequence + 1, Activity: activity, Stage: activity, Step: step,
		Status: status, Class: class, Reason: reason, NextOperation: nextOperationForClass(class),
		BlockedBy: nonNilStrings(blocked), CandidateIDs: candidates, ReceiptIDs: receipts,
	})
}

func nextOperationForClass(class string) string {
	switch class {
	case "missing_midpoint":
		return "supply_admissible_midpoint_observation"
	case "stale_receipt":
		return "replay_midpoint_against_pinned_release_tag_asset"
	case "ambiguous_ordering":
		return "repair_candidate_ordering"
	case "digest_mismatch":
		return "supply_a_receipt_with_matching_pinned_digests"
	case "non_monotonic_contradiction":
		return "inspect_claim_graph_and_reconcile_observations"
	case "false_minimal_boundary":
		return "revalidate_trusted_anchors"
	case "clean_boundary", "replay_boundary", "duplicate_equivalent_receipt":
		return "publish_boundary_receipt"
	default:
		return "continue_evaluation"
	}
}

func (e *evaluation) globalClassOr(fallback string) string {
	if e.globalClass != "" {
		return e.globalClass
	}
	return fallback
}

func (e *evaluation) globalReasonOr(fallback string) string {
	if e.globalReason != "" {
		return e.globalReason
	}
	return fallback
}

func (e *evaluation) globalStepOr(fallback string) string {
	if e.globalStep != "" {
		return e.globalStep
	}
	return fallback
}

func candidateIDs(candidates []Candidate) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	return ids
}

func ledgerReceiptIDs(ledger []LedgerEntry) []string {
	ids := make([]string, 0, len(ledger))
	for _, entry := range ledger {
		ids = append(ids, entry.ReceiptID)
	}
	return ids
}

func nonEmptyIDs(first, second string) []string {
	ids := []string{}
	if first != "" {
		ids = append(ids, first)
	}
	if second != "" {
		ids = append(ids, second)
	}
	return ids
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func indexOf(candidates []Candidate, id string) int {
	for index, candidate := range candidates {
		if candidate.ID == id {
			return index
		}
	}
	return -1
}

func candidateOrdinal(candidates []Candidate, id string) int {
	for _, candidate := range candidates {
		if candidate.ID == id {
			return candidate.Ordinal
		}
	}
	return -1
}

func hasDigestMismatch(ledger []LedgerEntry) bool {
	for _, entry := range ledger {
		if entry.Admissibility == DigestMismatch {
			return true
		}
	}
	return false
}

func hasAdmissibleResult(ledger []LedgerEntry, candidateID, result string) bool {
	for _, entry := range ledger {
		if entry.CandidateID == candidateID && entry.Admissibility == Admissible && entry.EffectiveResult == result {
			return true
		}
	}
	return false
}

func firstLedgerReason(ledger []LedgerEntry, kind string) string {
	for _, entry := range ledger {
		if entry.Admissibility == kind {
			return entry.ReceiptID
		}
	}
	return kind
}

func hasReplay(ledger []LedgerEntry) bool {
	for _, entry := range ledger {
		if entry.ReplayOf != "" {
			return true
		}
	}
	return false
}

func hasDuplicate(ledger []LedgerEntry) bool {
	for _, entry := range ledger {
		if entry.Admissibility == DuplicateReceipt {
			return true
		}
	}
	return false
}

func assetPin(asset AssetPin) bool {
	return asset.Name != "" && asset.Size > 0 && isDigest(asset.Digest)
}

func isDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func objectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}
