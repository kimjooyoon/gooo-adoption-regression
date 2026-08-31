#!/usr/bin/env bash
set -euo pipefail

repo_root="$(pwd)"
ci_dir="$(mktemp -d)"
out_a="$(mktemp -d)"
out_b="$(mktemp -d)"
trap 'rm -rf "$ci_dir" "$out_b"' EXIT

build_time="$ci_dir/build.time"
test_time="$ci_dir/test.time"
test_json="$ci_dir/test.json"
runtime_json="$ci_dir/runtime.json"
binary="$ci_dir/gooo-adoption-regression"
format_output="$ci_dir/gofmt.txt"

find . -type f -name '*.go' -not -path './.git/*' -exec gofmt -d {} + > "$format_output"
if [ -s "$format_output" ]; then
  echo 'gofmt diagnostic diff' >&2
  cat "$format_output" >&2
fi

/usr/bin/time -f '%e %M' -o "$build_time" go build -o "$binary" ./cmd/gooo-adoption-regression
/usr/bin/time -f '%e %M' -o "$test_time" go test -json -count=1 ./... > "$test_json"
go vet ./...

read -r build_seconds build_rss < "$build_time"
read -r test_seconds test_rss < "$test_time"
build_wall_ms="$(awk -v value="$build_seconds" 'BEGIN { printf "%d", (value * 1000) + 0.5 }')"
test_wall_ms="$(awk -v value="$test_seconds" 'BEGIN { printf "%d", (value * 1000) + 0.5 }')"
peak_rss_kib="$(awk -v build="$build_rss" -v test="$test_rss" 'BEGIN { if (build > test) print build; else print test }')"
tests_discovered="$(jq -s '[.[] | select(.Action == "run" and .Test != null)] | length' "$test_json")"
tests_executed="$(jq -s '[.[] | select((.Action == "pass" or .Action == "fail") and .Test != null)] | length' "$test_json")"
tests_skipped="$(jq -s '[.[] | select(.Action == "skip" and .Test != null)] | length' "$test_json")"
tests_cached="$(jq -s '[.[] | select(.Action == "output" and .Test != null and ((.Output // "") | contains("(cached)")))] | length' "$test_json")"

ci_job_id="$(gh api "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/jobs" --paginate --jq '.jobs[] | select(.name == "conformance") | .id' | head -n 1 || true)"
if [ -z "$ci_job_id" ]; then ci_job_id="$GITHUB_JOB"; fi

file_count="$(find . -type f -not -path './.git/*' -not -path './README.md' | wc -l | tr -d ' ')"
directory_count="$(find . -type d -not -path './.git' -not -path './.git/*' | wc -l | tr -d ' ')"
physical_lines="$(find . -type f -not -path './.git/*' -not -path './README.md' -print0 | xargs -0 -r awk '{ total++ } END { print total + 0 }')"
go_files="$(find . -type f -name '*.go' -not -path './.git/*' | wc -l | tr -d ' ')"
go_lines="$(find . -type f -name '*.go' -not -path './.git/*' -print0 | xargs -0 -r awk '{ total++ } END { print total + 0 }')"
gooo_lines="$(find . -type f -name '*.gooo' -not -path './.git/*' -print0 | xargs -0 -r awk '{ total++ } END { print total + 0 }')"

jq -n \
  --arg schema 'gooo/adoption-regression/ci-runtime/v1' \
  --arg run_id "$GITHUB_RUN_ID" \
  --arg job_id "$ci_job_id" \
  --argjson build_wall_ms "$build_wall_ms" \
  --argjson test_wall_ms "$test_wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  --argjson tests_discovered "$tests_discovered" \
  --argjson tests_executed "$tests_executed" \
  --argjson tests_skipped "$tests_skipped" \
  --argjson tests_cached "$tests_cached" \
  --argjson directories "$directory_count" \
  --argjson files "$file_count" \
  --argjson physical_lines "$physical_lines" \
  --argjson go_lines "$go_lines" \
  --argjson gooo_lines "$gooo_lines" \
  '{schema:$schema,ci_run_id:$run_id,ci_job_id:$job_id,build_wall_ms:$build_wall_ms,test_wall_ms:$test_wall_ms,peak_rss_kib:$peak_rss_kib,tests_discovered:$tests_discovered,tests_executed:$tests_executed,tests_skipped:$tests_skipped,tests_cached:$tests_cached,cache_hit:false,directories:$directories,files:$files,physical_lines:$physical_lines,go_lines:$go_lines,gooo_lines:$gooo_lines,output_artifact_count:7,repository_writes:0,local_test_executions:0,cross_project_required_gates:0}' > "$runtime_json"

"$binary" run --source "$repo_root/examples/adoption-regression.gooo" --contract "$repo_root/contracts/adoption-regression-denominator-v1.json" --corpus "$repo_root/examples/canonical-corpus.json" --ci-runtime "$runtime_json" --output-dir "$out_a"
"$binary" run --source "$repo_root/examples/adoption-regression.gooo" --contract "$repo_root/contracts/adoption-regression-denominator-v1.json" --corpus "$repo_root/examples/canonical-corpus.json" --ci-runtime "$runtime_json" --output-dir "$out_b"

for artifact in comparison-manifest.json before-receipt.json after-receipt.json test-manifest.json semantic-diff.json replay-receipt.json regression-report.md; do
  cmp "$out_a/$artifact" "$out_b/$artifact"
done

if [ "$(find "$out_a" -type f | wc -l | tr -d ' ')" -ne 7 ]; then
  echo 'unexpected output artifact count' >&2
  exit 1
fi
if [ -n "$(git status --porcelain --untracked-files=all)" ]; then
  echo 'repository changed during conformance' >&2
  git status --porcelain --untracked-files=all >&2
  exit 1
fi

printf 'GOOO_OUTPUT_DIR=%s\n' "$out_a" >> "$GITHUB_ENV"
printf 'GOOO_RUNTIME_JSON=%s\n' "$runtime_json" >> "$GITHUB_ENV"
