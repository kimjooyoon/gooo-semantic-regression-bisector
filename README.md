# Gooo semantic regression bisector

This repository finds the smallest semantic state/claim boundary between a
pinned known-good candidate and a pinned known-refuted candidate. It consumes
only the supplied manifest and observation receipts. It does not fetch,
rewrite, retag, or mutate a source repository, release, tag, or asset.

Candidate ordering, receipt admissibility, interval narrowing, stop conditions,
and decision precedence are owned by the versioned Gooo contract at
[`internal/contract/semantic_bisect.gooo`](internal/contract/semantic_bisect.gooo).
Go provides the evaluator, generator, and runtime. Every invocation records the
contract's nine activities exactly once.

## Decisions

`CLOSED` is emitted only when adjacent PASS and REFUTED candidates are proven.
`UNKNOWN` is a causal frontier: the midpoint is not inferred, and the receipt
contains `stage`, `step`, `reason`, `class`, `next_operation`, and `blocked_by`.
`REFUTED` has precedence when observations contradict each other or a pinned
digest is wrong. Operational receipt failures remain distinguishable from
semantic contradictions in the ledger and report.

An actual external regression cannot be reduced by inference alone; without a
supplied admissible observation it remains `UNKNOWN` and is reported as a
blocked causal frontier.

The canonical conformance set contains exactly nine cases: three CLOSED, three
UNKNOWN, and three REFUTED. There is no aggregate score or percentage.

## Run

```text
go run ./cmd/gooo-semantic-regression-bisector \
  --manifest path/to/bisect-manifest.json \
  --observations path/to/observation-ledger.ndjson \
  --out output
```

The caller-owned output directory must contain exactly these six files:

* `bisect-manifest.json`
* `candidate-events.ndjson`
* `observation-ledger.ndjson`
* `boundary-receipt.json`
* `replay-receipt.json`
* `report.md`

The manifest pins immutable release metadata, annotated tag object and target,
asset size and digest, candidate state and claim digests, and an exact integer
inventory. Observation receipts must repeat the candidate pins exactly to be
admissible. Equivalent duplicates are retained in the ledger but counted once.

## Verification authority

The repository intentionally does not use local Go test, build, vet, format,
generate, conformance, or integration commands as evidence. GitHub Actions is
the verification authority. Pull requests run compile, build, test,
conformance, and integration checks and record `wall_ms` and `peak_rss_kib` for
each stage, together with integer test counts for total, selected, executed,
reused, failed, and unknown tests. A successful main-branch verification
creates the annotated `v0.1.0` tag and follows draft release → asset upload →
publish; the public API verification checks the tag object/target, immutable
flag, and exact asset sizes and digests.

The one-time repository immutable-release operator procedure is documented in
[`docs/operator-enable-immutable-releases.md`](docs/operator-enable-immutable-releases.md).
