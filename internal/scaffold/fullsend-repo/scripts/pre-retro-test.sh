#!/usr/bin/env bash
# pre-retro-test.sh — Test pre-retro.sh validation and PR context prefetch.
#
# Uses a mock gh command to capture calls without hitting GitHub.
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PRE_SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Helpers ---

# build_mock creates a mock gh binary that returns preconfigured responses.
# Arguments:
#   $1 — JSON for "gh pr view" (empty = fail with exit 1)
#   $2 — JSON for "gh api .../comments" (empty = fail with exit 1)
#   $3 — JSON for "gh api .../reviews" (empty = fail with exit 1)
#   $4 — JSON for "gh api .../actions/runs" (empty = fail with exit 1)
build_mock() {
  local pr_json="${1:-}"
  local comments_json="${2:-}"
  local reviews_json="${3:-}"
  local runs_json="${4:-}"
  local mock_bin="${TMPDIR}/bin"
  local gh_log="${TMPDIR}/gh-calls.log"

  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"
  : > "${gh_log}"

  # Write mock data to temp files for the mock script to read.
  printf '%s' "${pr_json}" > "${TMPDIR}/pr.json"
  printf '%s' "${comments_json}" > "${TMPDIR}/comments.json"
  printf '%s' "${reviews_json}" > "${TMPDIR}/reviews.json"
  printf '%s' "${runs_json}" > "${TMPDIR}/runs.json"

  cat > "${mock_bin}/gh" <<MOCKEOF
#!/usr/bin/env bash
LOG_FILE="${gh_log}"
DATA_DIR="${TMPDIR}"

echo "gh \$*" >> "\${LOG_FILE}"

if [[ "\$1" == "pr" && "\$2" == "view" ]]; then
  DATA=\$(cat "\${DATA_DIR}/pr.json")
  if [[ -z "\${DATA}" ]]; then
    echo "mock: pr view failed" >&2
    exit 1
  fi
  echo "\${DATA}"
  exit 0
fi

if [[ "\$1" == "api" ]]; then
  URL="\$2"
  if [[ "\${URL}" == *"/issues/"*"/comments" ]]; then
    DATA=\$(cat "\${DATA_DIR}/comments.json")
    if [[ -z "\${DATA}" ]]; then
      echo "mock: comments fetch failed" >&2
      exit 1
    fi
    echo "\${DATA}"
    exit 0
  fi
  if [[ "\${URL}" == *"/pulls/"*"/reviews" ]]; then
    DATA=\$(cat "\${DATA_DIR}/reviews.json")
    if [[ -z "\${DATA}" ]]; then
      echo "mock: reviews fetch failed" >&2
      exit 1
    fi
    echo "\${DATA}"
    exit 0
  fi
  if [[ "\${URL}" == *"/actions/runs" ]]; then
    DATA=\$(cat "\${DATA_DIR}/runs.json")
    if [[ -z "\${DATA}" ]]; then
      echo "mock: runs fetch failed" >&2
      exit 1
    fi
    echo "\${DATA}"
    exit 0
  fi
fi

exit 0
MOCKEOF

  chmod +x "${mock_bin}/gh"
  echo "${mock_bin}"
}

# Standard mock data for successful prefetch.
PR_JSON='{"number":42,"title":"Fix widget","body":"Fixes the widget","state":"MERGED","author":{"login":"dev"},"labels":[],"baseRefName":"main","headRefName":"fix-widget","additions":10,"deletions":5,"changedFiles":2,"commits":{"totalCount":1},"createdAt":"2026-01-01T00:00:00Z","closedAt":"2026-01-02T00:00:00Z","mergedAt":"2026-01-02T00:00:00Z"}'
COMMENTS_JSON='[{"id":1,"body":"LGTM","user":{"login":"reviewer"}}]'
REVIEWS_JSON='[{"id":1,"state":"APPROVED","user":{"login":"reviewer"}}]'
RUNS_JSON='{"workflow_runs":[{"id":100,"name":"CI","status":"completed","conclusion":"success","html_url":"https://github.com/test-org/test-repo/actions/runs/100","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:01:00Z","head_sha":"abc123","head_branch":"main","event":"push"}]}'

run_test() {
  local test_name="$1"
  local url="$2"
  local expect_exit="$3"
  local check_fn="$4"  # function name to call for assertions
  local gh_token="${5-fake-token}"
  local pr_json="${6-${PR_JSON}}"
  local comments_json="${7-${COMMENTS_JSON}}"
  local reviews_json="${8-${REVIEWS_JSON}}"
  local runs_json="${9-${RUNS_JSON}}"

  local mock_bin
  mock_bin="$(build_mock "${pr_json}" "${comments_json}" "${reviews_json}" "${runs_json}")"

  local runner_temp="${TMPDIR}/runner-temp-${test_name}"
  local github_output="${TMPDIR}/github-output-${test_name}"
  mkdir -p "${runner_temp}"
  : > "${github_output}"

  local exit_code=0
  env \
    PATH="${mock_bin}:${PATH}" \
    ORIGINATING_URL="${url}" \
    GH_TOKEN="${gh_token}" \
    RUNNER_TEMP="${runner_temp}" \
    GITHUB_OUTPUT="${github_output}" \
    RETRO_COMMENT="" \
    bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout-${test_name}.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    echo "--- stdout ---"
    cat "${TMPDIR}/stdout-${test_name}.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Run check function if provided.
  if [[ -n "${check_fn}" ]]; then
    if ! "${check_fn}" "${test_name}" "${runner_temp}" "${github_output}"; then
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# --- Check functions ---

check_context_file_exists() {
  local test_name="$1"
  local runner_temp="$2"
  local context_file="${runner_temp}/pr-context.json"

  if [[ ! -f "${context_file}" ]]; then
    echo "FAIL: ${test_name} — pr-context.json not created"
    return 1
  fi

  if ! jq empty "${context_file}" 2>/dev/null; then
    echo "FAIL: ${test_name} — pr-context.json is not valid JSON"
    cat "${context_file}"
    return 1
  fi

  return 0
}

check_prefetch_ok() {
  local test_name="$1"
  local runner_temp="$2"
  local github_output="$3"

  check_context_file_exists "${test_name}" "${runner_temp}" || return 1

  local context_file="${runner_temp}/pr-context.json"

  # Verify prefetch_status is "ok".
  local status
  status=$(jq -r '.prefetch_status' "${context_file}")
  if [[ "${status}" != "ok" ]]; then
    echo "FAIL: ${test_name} — prefetch_status='${status}', expected 'ok'"
    return 1
  fi

  # Verify PR metadata is present.
  local pr_number
  pr_number=$(jq -r '.pr.number' "${context_file}")
  if [[ "${pr_number}" != "42" ]]; then
    echo "FAIL: ${test_name} — pr.number='${pr_number}', expected '42'"
    return 1
  fi

  # Verify comments array is present.
  local comment_count
  comment_count=$(jq '.comments | length' "${context_file}")
  if [[ "${comment_count}" -lt 1 ]]; then
    echo "FAIL: ${test_name} — expected at least 1 comment"
    return 1
  fi

  # Verify reviews array is present.
  local review_count
  review_count=$(jq '.reviews | length' "${context_file}")
  if [[ "${review_count}" -lt 1 ]]; then
    echo "FAIL: ${test_name} — expected at least 1 review"
    return 1
  fi

  # Verify workflow_runs array is present.
  local run_count
  run_count=$(jq '.workflow_runs | length' "${context_file}")
  if [[ "${run_count}" -lt 1 ]]; then
    echo "FAIL: ${test_name} — expected at least 1 workflow run"
    return 1
  fi

  # Verify GITHUB_OUTPUT was written.
  if ! grep -q "^pr_context_file=" "${github_output}" 2>/dev/null; then
    echo "FAIL: ${test_name} — pr_context_file not written to GITHUB_OUTPUT"
    return 1
  fi

  return 0
}

check_no_context_file() {
  local test_name="$1"
  local runner_temp="$2"
  local context_file="${runner_temp}/pr-context.json"

  if [[ -f "${context_file}" ]]; then
    echo "FAIL: ${test_name} — pr-context.json should not exist"
    return 1
  fi

  return 0
}

check_prefetch_failed() {
  local test_name="$1"
  local runner_temp="$2"

  check_context_file_exists "${test_name}" "${runner_temp}" || return 1

  local context_file="${runner_temp}/pr-context.json"
  local status
  status=$(jq -r '.prefetch_status' "${context_file}")
  if [[ "${status}" != "failed" ]]; then
    echo "FAIL: ${test_name} — prefetch_status='${status}', expected 'failed'"
    return 1
  fi

  return 0
}

# --- Test cases ---

# 1. Valid PR URL with working token → full prefetch succeeds.
run_test "pr-url-prefetch-ok" \
  "https://github.com/test-org/test-repo/pull/42" \
  0 \
  "check_prefetch_ok"

# 2. Issue URL → skips prefetch entirely, no context file.
run_test "issue-url-skips-prefetch" \
  "https://github.com/test-org/test-repo/issues/42" \
  0 \
  "check_no_context_file"

# 3. No GH_TOKEN → skips prefetch, exits 0.
run_test "no-token-skips-prefetch" \
  "https://github.com/test-org/test-repo/pull/42" \
  0 \
  "check_no_context_file" \
  ""

# 4. PR metadata fetch fails → graceful degradation, exits 0, writes
#    a context file with prefetch_status=failed.
run_test "pr-metadata-fails-graceful" \
  "https://github.com/test-org/test-repo/pull/42" \
  0 \
  "check_prefetch_failed" \
  "fake-token" \
  "" \
  "${COMMENTS_JSON}" \
  "${REVIEWS_JSON}" \
  "${RUNS_JSON}"

# 5. Comments fetch fails → prefetch still succeeds with empty comments.
check_comments_fallback() {
  local test_name="$1"
  local runner_temp="$2"

  check_context_file_exists "${test_name}" "${runner_temp}" || return 1

  local context_file="${runner_temp}/pr-context.json"
  local status
  status=$(jq -r '.prefetch_status' "${context_file}")
  if [[ "${status}" != "ok" ]]; then
    echo "FAIL: ${test_name} — prefetch_status='${status}', expected 'ok'"
    return 1
  fi

  # Comments should fall back to empty array.
  local comment_count
  comment_count=$(jq '.comments | length' "${context_file}")
  if [[ "${comment_count}" -ne 0 ]]; then
    echo "FAIL: ${test_name} — expected 0 comments (fallback), got ${comment_count}"
    return 1
  fi

  return 0
}

run_test "comments-fail-fallback" \
  "https://github.com/test-org/test-repo/pull/42" \
  0 \
  "check_comments_fallback" \
  "fake-token" \
  "${PR_JSON}" \
  "" \
  "${REVIEWS_JSON}" \
  "${RUNS_JSON}"

# 6. Invalid URL → rejected in validation, exits 1.
run_test "invalid-url-rejected" \
  "https://not-github.com/org/repo/pull/1" \
  1 \
  ""

# 7. On-demand trigger (RETRO_COMMENT set) still prefetches.
check_ondemand_prefetch() {
  local test_name="$1"
  local runner_temp="$2"

  check_context_file_exists "${test_name}" "${runner_temp}" || return 1

  local context_file="${runner_temp}/pr-context.json"
  local status
  status=$(jq -r '.prefetch_status' "${context_file}")
  if [[ "${status}" != "ok" ]]; then
    echo "FAIL: ${test_name} — prefetch_status='${status}', expected 'ok'"
    return 1
  fi

  return 0
}

# Run test 7 with RETRO_COMMENT manually since run_test clears it.
test_name="ondemand-trigger-prefetches"
mock_bin="$(build_mock "${PR_JSON}" "${COMMENTS_JSON}" "${REVIEWS_JSON}" "${RUNS_JSON}")"
runner_temp="${TMPDIR}/runner-temp-${test_name}"
github_output="${TMPDIR}/github-output-${test_name}"
mkdir -p "${runner_temp}"
: > "${github_output}"

exit_code=0
env \
  PATH="${mock_bin}:${PATH}" \
  ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42" \
  GH_TOKEN="fake-token" \
  RUNNER_TEMP="${runner_temp}" \
  GITHUB_OUTPUT="${github_output}" \
  RETRO_COMMENT="/retro please analyze this" \
  bash "${PRE_SCRIPT}" > "${TMPDIR}/stdout-${test_name}.log" 2>&1 || exit_code=$?

if [[ ${exit_code} -ne 0 ]]; then
  echo "FAIL: ${test_name} — expected exit 0, got ${exit_code}"
  cat "${TMPDIR}/stdout-${test_name}.log"
  FAILURES=$((FAILURES + 1))
elif ! check_ondemand_prefetch "${test_name}" "${runner_temp}"; then
  FAILURES=$((FAILURES + 1))
else
  echo "PASS: ${test_name}"
fi

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
