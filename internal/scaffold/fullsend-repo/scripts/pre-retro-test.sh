#!/usr/bin/env bash
# pre-retro-test.sh — Test pre-retro.sh validation and PR context prefetch.
#
# Uses a mock gh command to simulate API responses without hitting GitHub.
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Mock data ---

# PR metadata returned by "gh pr view --json ..."
PR_META='{"title":"Test PR","state":"MERGED","author":{"login":"dev1"},"labels":[],"mergedBy":{"login":"merger"},"baseRefName":"main","headRefName":"feature","additions":10,"deletions":5,"changedFiles":3,"commits":[{"oid":"abc123"}],"createdAt":"2026-01-01T00:00:00Z","closedAt":"2026-01-02T00:00:00Z","mergedAt":"2026-01-02T00:00:00Z"}'

# NDJSON comments returned by "gh api .../comments --paginate --jq '.[]'"
# shellcheck disable=SC2016
COMMENTS_NDJSON='{"id":1,"body":"comment one","user":{"login":"dev1"}}
{"id":2,"body":"comment two","user":{"login":"dev2"}}'

# NDJSON reviews returned by "gh api .../reviews --paginate --jq '.[]'"
REVIEWS_NDJSON='{"id":10,"state":"APPROVED","user":{"login":"reviewer1"}}'

# JSON array returned by "gh api .../actions/runs --jq '.workflow_runs'"
RUNS_JSON='[{"id":100,"name":"CI","status":"completed","conclusion":"success"}]'

# --- Helpers ---

# build_mock creates a mock gh binary that returns preconfigured responses.
# The mock reads response data and exit codes from files in the data dir,
# routing by subcommand and URL pattern.
#
# Arguments (all optional, defaults produce a fully successful mock):
#   $1 — pr view output (JSON object)
#   $2 — pr view exit code
#   $3 — comments output (NDJSON, one object per line)
#   $4 — comments exit code
#   $5 — reviews output (NDJSON)
#   $6 — reviews exit code
#   $7 — runs output (JSON array)
#   $8 — runs exit code
build_mock() {
  local pr_view="${1:-${PR_META}}"
  local pr_view_exit="${2:-0}"
  local comments="${3:-${COMMENTS_NDJSON}}"
  local comments_exit="${4:-0}"
  local reviews="${5:-${REVIEWS_NDJSON}}"
  local reviews_exit="${6:-0}"
  local runs="${7:-${RUNS_JSON}}"
  local runs_exit="${8:-0}"

  local mock_bin="${TMPDIR}/bin"
  local data_dir="${TMPDIR}/mock-data"

  rm -rf "${mock_bin}" "${data_dir}"
  mkdir -p "${mock_bin}" "${data_dir}"

  printf '%s' "${pr_view}" > "${data_dir}/pr-view-output"
  printf '%s' "${pr_view_exit}" > "${data_dir}/pr-view-exit"
  printf '%s' "${comments}" > "${data_dir}/comments-output"
  printf '%s' "${comments_exit}" > "${data_dir}/comments-exit"
  printf '%s' "${reviews}" > "${data_dir}/reviews-output"
  printf '%s' "${reviews_exit}" > "${data_dir}/reviews-exit"
  printf '%s' "${runs}" > "${data_dir}/runs-output"
  printf '%s' "${runs_exit}" > "${data_dir}/runs-exit"

  cat > "${mock_bin}/gh" <<'MOCKEOF'
#!/usr/bin/env bash
DATA="__DATA_DIR__"

if [[ "$1" == "pr" && "$2" == "view" ]]; then
  ec=$(cat "${DATA}/pr-view-exit")
  if [[ "${ec}" -eq 0 ]]; then
    cat "${DATA}/pr-view-output"
  fi
  exit "${ec}"
fi

if [[ "$1" == "api" ]]; then
  url="$2"
  if [[ "${url}" == *"/comments"* ]]; then
    ec=$(cat "${DATA}/comments-exit")
    if [[ "${ec}" -eq 0 ]]; then
      cat "${DATA}/comments-output"
    fi
    exit "${ec}"
  fi
  if [[ "${url}" == *"/reviews"* ]]; then
    ec=$(cat "${DATA}/reviews-exit")
    if [[ "${ec}" -eq 0 ]]; then
      cat "${DATA}/reviews-output"
    fi
    exit "${ec}"
  fi
  if [[ "${url}" == *"/actions/runs"* ]]; then
    ec=$(cat "${DATA}/runs-exit")
    if [[ "${ec}" -eq 0 ]]; then
      cat "${DATA}/runs-output"
    fi
    exit "${ec}"
  fi
fi

exit 0
MOCKEOF

  local escaped="${data_dir//\//\\/}"
  perl -pi -e "s/__DATA_DIR__/${escaped}/g" "${mock_bin}/gh"
  chmod +x "${mock_bin}/gh"

  echo "${mock_bin}"
}

# run_test executes pre-retro.sh with a mock gh and validates the result.
#
# Arguments:
#   $1 — test name
#   $2 — expected exit code
#   $3 — ORIGINATING_URL
#   $4 — GH_TOKEN (empty string to unset)
#   $5 — expected prefetch_status in pr-context.json ("" to skip check)
#   $6 — expected stdout substring ("" to skip check)
#   $7 — RETRO_COMMENT (optional)
run_test() {
  local test_name="$1"
  local expect_exit="$2"
  local url="$3"
  local token="$4"
  local expect_status="$5"
  local expect_stdout="$6"
  local retro_comment="${7:-}"

  local mock_bin
  mock_bin="$(build_mock)"

  local runner_temp="${TMPDIR}/runner-temp"
  rm -rf "${runner_temp}"
  mkdir -p "${runner_temp}"

  # Build env array — always set GH_TOKEN (empty string unsets it).
  local env_cmd=(
    env
    PATH="${mock_bin}:${PATH}"
    ORIGINATING_URL="${url}"
    RUNNER_TEMP="${runner_temp}"
    "GH_TOKEN=${token}"
  )
  if [[ -n "${retro_comment}" ]]; then
    env_cmd+=("RETRO_COMMENT=${retro_comment}")
  fi

  local exit_code=0
  "${env_cmd[@]}" bash "${SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  # Check exit code.
  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    echo "--- stdout ---"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Check expected stdout substring.
  if [[ -n "${expect_stdout}" ]]; then
    if ! grep -qF "${expect_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
      echo "FAIL: ${test_name} — expected stdout containing '${expect_stdout}'"
      echo "--- stdout ---"
      cat "${TMPDIR}/stdout.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  # Check pr-context.json prefetch_status.
  if [[ -n "${expect_status}" ]]; then
    local context_file="${runner_temp}/pr-context.json"
    if [[ ! -f "${context_file}" ]]; then
      echo "FAIL: ${test_name} — pr-context.json not found"
      echo "--- stdout ---"
      cat "${TMPDIR}/stdout.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
    local actual_status
    actual_status=$(jq -r '.prefetch_status' "${context_file}" 2>/dev/null || echo "PARSE_ERROR")
    if [[ "${actual_status}" != "${expect_status}" ]]; then
      echo "FAIL: ${test_name} — expected prefetch_status '${expect_status}', got '${actual_status}'"
      echo "--- pr-context.json ---"
      cat "${context_file}"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# run_test_with_mock is like run_test but accepts custom mock parameters.
# Arguments 1-7 are the same as run_test. Arguments 8+ are passed to
# build_mock to override default mock responses.
run_test_with_mock() {
  local test_name="$1"
  local expect_exit="$2"
  local url="$3"
  local token="$4"
  local expect_status="$5"
  local expect_stdout="$6"
  local retro_comment="${7:-}"
  shift 7

  # Remaining args go to build_mock.
  local mock_bin
  mock_bin="$(build_mock "$@")"

  local runner_temp="${TMPDIR}/runner-temp"
  rm -rf "${runner_temp}"
  mkdir -p "${runner_temp}"

  local env_cmd=(
    env
    PATH="${mock_bin}:${PATH}"
    ORIGINATING_URL="${url}"
    RUNNER_TEMP="${runner_temp}"
    "GH_TOKEN=${token}"
  )
  if [[ -n "${retro_comment}" ]]; then
    env_cmd+=("RETRO_COMMENT=${retro_comment}")
  fi

  local exit_code=0
  "${env_cmd[@]}" bash "${SCRIPT}" > "${TMPDIR}/stdout.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    echo "--- stdout ---"
    cat "${TMPDIR}/stdout.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  if [[ -n "${expect_stdout}" ]]; then
    if ! grep -qF "${expect_stdout}" "${TMPDIR}/stdout.log" 2>/dev/null; then
      echo "FAIL: ${test_name} — expected stdout containing '${expect_stdout}'"
      echo "--- stdout ---"
      cat "${TMPDIR}/stdout.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  if [[ -n "${expect_status}" ]]; then
    local context_file="${runner_temp}/pr-context.json"
    if [[ ! -f "${context_file}" ]]; then
      echo "FAIL: ${test_name} — pr-context.json not found"
      echo "--- stdout ---"
      cat "${TMPDIR}/stdout.log"
      FAILURES=$((FAILURES + 1))
      return
    fi
    local actual_status
    actual_status=$(jq -r '.prefetch_status' "${context_file}" 2>/dev/null || echo "PARSE_ERROR")
    if [[ "${actual_status}" != "${expect_status}" ]]; then
      echo "FAIL: ${test_name} — expected prefetch_status '${expect_status}', got '${actual_status}'"
      echo "--- pr-context.json ---"
      cat "${context_file}"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# --- Test cases ---

# 1. Valid PR URL with all APIs succeeding — full prefetch.
run_test "pr-url-prefetch-ok" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "fake-token" \
  "full" \
  "PR context written to"

# 2. Issue URL — skips prefetch entirely.
run_test "issue-url-skips-prefetch" \
  0 \
  "https://github.com/test-org/test-repo/issues/42" \
  "fake-token" \
  "" \
  "skipping PR context prefetch"

# 3. No GH_TOKEN — skips prefetch.
run_test "no-token-skips-prefetch" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "" \
  "" \
  "GH_TOKEN not set"

# 4. Invalid URL — exits 1.
run_test "invalid-url-rejected" \
  1 \
  "https://not-github.com/org/repo/pull/1" \
  "fake-token" \
  "" \
  "does not match expected pattern"

# 5. PR metadata fetch fails — partial status with empty metadata.
run_test_with_mock "pr-metadata-fails-graceful" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "fake-token" \
  "partial" \
  "Failed to fetch PR metadata" \
  "" \
  "" 1 \
  "${COMMENTS_NDJSON}" 0 \
  "${REVIEWS_NDJSON}" 0 \
  "${RUNS_JSON}" 0

# 6. Comments fetch fails — empty array fallback, partial status.
run_test_with_mock "comments-fail-fallback" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "fake-token" \
  "partial" \
  "Failed to fetch PR comments" \
  "" \
  "${PR_META}" 0 \
  "" 1 \
  "${REVIEWS_NDJSON}" 0 \
  "${RUNS_JSON}" 0

# 7. Reviews fetch fails — empty array fallback, partial status.
run_test_with_mock "reviews-fail-fallback" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "fake-token" \
  "partial" \
  "Failed to fetch PR reviews" \
  "" \
  "${PR_META}" 0 \
  "${COMMENTS_NDJSON}" 0 \
  "" 1 \
  "${RUNS_JSON}" 0

# 8. Workflow runs fetch fails — empty array fallback, partial status.
run_test_with_mock "runs-fail-fallback" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "fake-token" \
  "partial" \
  "Failed to fetch workflow runs" \
  "" \
  "${PR_META}" 0 \
  "${COMMENTS_NDJSON}" 0 \
  "${REVIEWS_NDJSON}" 0 \
  "" 1

# 9. On-demand trigger with RETRO_COMMENT — still prefetches.
run_test "ondemand-trigger-prefetches" \
  0 \
  "https://github.com/test-org/test-repo/pull/42" \
  "fake-token" \
  "full" \
  "Retro triggered on-demand with comment." \
  "focus on the review agent"

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
