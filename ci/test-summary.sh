#!/usr/bin/env bash
set -euo pipefail

output_dir="${1:?output directory is required}"
summary="$output_dir/test-summary.json"
json_logs=("$output_dir/test.log" "$output_dir/conformance.log" "$output_dir/integration.log")
total_functions="$(find . -type f -name '*_test.go' -not -path './.git/*' -exec grep -h -E '^func Test[[:alnum:]_]+\(' {} + | wc -l | tr -d ' ')"
canonical_cases=9
total=$((total_functions + canonical_cases))
selected="$total"
executed="0"
reused="0"
failed="0"
if command -v jq >/dev/null 2>&1; then
  executed="$(jq -s '[.[] | select(.Action == "pass" and .Test != null) | .Test] | unique | length' "${json_logs[@]}" 2>/dev/null || echo 0)"
  reused="$(jq -s '[.[] | select(.Action == "cached" and .Test != null) | .Test] | unique | length' "${json_logs[@]}" 2>/dev/null || echo 0)"
  failed="$(jq -s '[.[] | select(.Action == "fail" and .Test != null) | .Test] | unique | length' "${json_logs[@]}" 2>/dev/null || echo 0)"
fi
unknown="$(jq -s '[.[] | select(.Test != null) | .Test | select(test("TestConformance/(missing_midpoint|stale_receipt|ambiguous_ordering)$"))] | unique | length' "$output_dir/conformance.log" 2>/dev/null || echo 0)"
jq -n \
  --argjson total "$total" \
  --argjson selected "$selected" \
  --argjson executed "$executed" \
  --argjson reused "$reused" \
  --argjson failed "$failed" \
  --argjson unknown "$unknown" \
  '{total:$total,selected:$selected,executed:$executed,reused:$reused,failed:$failed,unknown:$unknown}' \
  > "$summary"
