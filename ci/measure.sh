#!/usr/bin/env bash
set -uo pipefail

output_dir="${1:?output directory is required}"
mkdir -p "$output_dir"
metrics="$output_dir/ci-metrics.ndjson"
: > "$metrics"
overall_rc=0

run_stage() {
  local stage="$1"
  shift
  local log="$output_dir/$stage.log"
  local timing="$output_dir/$stage.time"
  local rc=0
  /usr/bin/time -f '%e %M' -o "$timing" "$@" >"$log" 2>&1 || rc=$?
  local seconds="0"
  local rss="0"
  if read -r seconds rss < "$timing"; then
    :
  fi
  local wall_ms
  wall_ms="$(awk -v value="$seconds" 'BEGIN { printf "%d", value * 1000 + 0.5 }')"
  printf '{"stage":"%s","wall_ms":%d,"peak_rss_kib":%d,"exit_code":%d}\n' "$stage" "$wall_ms" "$rss" "$rc" >> "$metrics"
  if [ "$rc" -ne 0 ]; then
    overall_rc=1
  fi
}

run_stage compile bash -c 'go generate ./... && go test ./... -run "^$" -count=1'
run_stage build bash -c 'go_files="$(rg --files -g "*.go")"; test -z "$(gofmt -l $go_files)"; go vet ./...; go build ./...'
run_stage test go test -json ./... -count=1
run_stage conformance go test -json ./internal/conformance -run TestConformance -count=1
run_stage integration go test -json ./internal/conformance -run TestIntegrationOutputsExactlySixCallerOwnedFiles -count=1

if ! "$(dirname "$0")/test-summary.sh" "$output_dir"; then
  overall_rc=1
fi
if [ ! -s "$output_dir/test-summary.json" ]; then
  overall_rc=1
fi
exit "$overall_rc"
