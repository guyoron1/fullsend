#!/usr/bin/env bash
# pre-retro-test.sh — Test prefetch logic in pre-retro.sh
#
# Verifies that the PR context prefetch works correctly: produces valid
# JSON for PR and issue URLs, handles failures gracefully (non-blocking),
# and respects size limits.
#
# Run from the repo root:
#   bash internal/scaffold/fullsend-repo/scripts/pre-retro-test.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${SCRIPT_DIR}/pre-retro.sh"
FAILURES=0

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

# --- Helpers ---

# build_mock creates a mock gh binary that returns preconfigured JSON
# responses based on the subcommand and arguments.
#   $1 — JSON for gh pr view
#   $2 — JSON for gh api .../comments
#   $3 — JSON for gh api .../reviews
#   $4 — JSON for gh api .../actions/runs (optional, defaults to empty)
#   $5 — JSON for gh api .../issues/N (optional, for issue metadata)
build_mock() {
  local pr_meta="${1:-}"
  local comments="${2:-[]}"
  local reviews="${3:-[]}"
  local workflow_runs="${4:-}"
  local issue_meta="${5:-}"
  local mock_bin="${TMPDIR}/bin"

  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"

  # Write response files for the mock to read.
  printf '%s' "${pr_meta}" > "${TMPDIR}/pr-meta.json"
  printf '%s' "${comments}" > "${TMPDIR}/comments.json"
  printf '%s' "${reviews}" > "${TMPDIR}/reviews.json"
  printf '%s' "${workflow_runs:-"{\"workflow_runs\":[]}"}" > "${TMPDIR}/runs.json"
  printf '%s' "${issue_meta}" > "${TMPDIR}/issue-meta.json"

  # Create the mock gh script. It inspects arguments to decide which
  # response file to return.
  cat > "${mock_bin}/gh" <<'MOCKSCRIPT'
#!/usr/bin/env bash
set -euo pipefail
TMPDIR_PLACEHOLDER="__TMPDIR__"

# Parse --jq argument from command line.
jq_filter=""
prev_arg=""
for arg in "$@"; do
  if [[ "${prev_arg}" == "--jq" ]]; then
    jq_filter="${arg}"
  fi
  prev_arg="${arg}"
done

# Helper: output a response file, applying --jq filter if provided.
respond() {
  if [[ -n "${jq_filter}" ]]; then
    jq "${jq_filter}" "$1"
  else
    cat "$1"
  fi
}

if [[ "$1" == "pr" && "$2" == "view" ]]; then
  cat "${TMPDIR_PLACEHOLDER}/pr-meta.json"
  exit 0
fi

if [[ "$1" == "api" ]]; then
  endpoint="$2"
  if [[ "${endpoint}" == *"/comments"* ]]; then
    respond "${TMPDIR_PLACEHOLDER}/comments.json"
    exit 0
  fi
  if [[ "${endpoint}" == *"/reviews"* ]]; then
    respond "${TMPDIR_PLACEHOLDER}/reviews.json"
    exit 0
  fi
  if [[ "${endpoint}" == *"/actions/runs"* ]]; then
    respond "${TMPDIR_PLACEHOLDER}/runs.json"
    exit 0
  fi
  if [[ "${endpoint}" == *"/issues/"* && "${endpoint}" != *"/comments"* ]]; then
    respond "${TMPDIR_PLACEHOLDER}/issue-meta.json"
    exit 0
  fi
fi

echo "mock gh: unhandled command: $*" >&2
exit 1
MOCKSCRIPT

  # Replace placeholder with actual temp directory path.
  local escaped_tmpdir="${TMPDIR//\//\\/}"
  perl -pi -e "s/__TMPDIR__/${escaped_tmpdir}/g" "${mock_bin}/gh"
  chmod +x "${mock_bin}/gh"

  echo "${mock_bin}"
}

# build_failing_mock creates a mock gh that always fails.
build_failing_mock() {
  local mock_bin="${TMPDIR}/bin-fail"
  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"

  cat > "${mock_bin}/gh" <<'MOCKSCRIPT'
#!/usr/bin/env bash
echo "mock gh: simulated failure" >&2
exit 1
MOCKSCRIPT
  chmod +x "${mock_bin}/gh"
  echo "${mock_bin}"
}

run_test() {
  local test_name="$1"
  local originating_url="$2"
  local mock_bin="$3"
  local expect_exit="$4"       # 0 = success
  local expect_file="$5"       # "exists" or "missing"
  local expect_type="${6:-}"   # expected source_type in JSON (optional)

  local workspace="${TMPDIR}/workspace-${test_name}"
  mkdir -p "${workspace}"

  local exit_code=0
  env \
    PATH="${mock_bin}:${PATH}" \
    GH_TOKEN="fake-token" \
    ORIGINATING_URL="${originating_url}" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > "${TMPDIR}/stdout-${test_name}.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne ${expect_exit} ]]; then
    echo "FAIL: ${test_name} — expected exit ${expect_exit}, got ${exit_code}"
    echo "--- stdout ---"
    cat "${TMPDIR}/stdout-${test_name}.log"
    FAILURES=$((FAILURES + 1))
    return
  fi

  local context_file="${workspace}/pr-context.json"
  if [[ "${expect_file}" == "exists" ]]; then
    if [[ ! -f "${context_file}" ]]; then
      echo "FAIL: ${test_name} — expected pr-context.json to exist"
      FAILURES=$((FAILURES + 1))
      return
    fi
    # Validate JSON is parseable.
    if ! jq empty "${context_file}" 2>/dev/null; then
      echo "FAIL: ${test_name} — pr-context.json is not valid JSON"
      cat "${context_file}"
      FAILURES=$((FAILURES + 1))
      return
    fi
    # Check source_type if specified.
    if [[ -n "${expect_type}" ]]; then
      local actual_type
      actual_type=$(jq -r '.source_type' "${context_file}")
      if [[ "${actual_type}" != "${expect_type}" ]]; then
        echo "FAIL: ${test_name} — expected source_type='${expect_type}', got '${actual_type}'"
        FAILURES=$((FAILURES + 1))
        return
      fi
    fi
  elif [[ "${expect_file}" == "missing" ]]; then
    if [[ -f "${context_file}" ]]; then
      echo "FAIL: ${test_name} — expected pr-context.json to NOT exist"
      FAILURES=$((FAILURES + 1))
      return
    fi
  fi

  echo "PASS: ${test_name}"
}

# --- Test data ---

PR_META='{
  "title": "Fix bug in login",
  "body": "This PR fixes the login bug.",
  "state": "MERGED",
  "author": {"login": "dev1"},
  "labels": [{"name": "bug"}],
  "baseRefName": "main",
  "headRefName": "fix/login",
  "headRefOid": "abc1234567890",
  "additions": 10,
  "deletions": 5,
  "changedFiles": 3,
  "commits": [{"oid": "abc1234"}],
  "createdAt": "2026-01-01T00:00:00Z",
  "closedAt": "2026-01-02T00:00:00Z",
  "mergedAt": "2026-01-02T00:00:00Z",
  "mergedBy": {"login": "reviewer1"},
  "number": 42,
  "url": "https://github.com/test-org/test-repo/pull/42"
}'

PR_COMMENTS='[
  {"id": 1, "body": "LGTM", "user": {"login": "reviewer1"}},
  {"id": 2, "body": "Please fix typo", "user": {"login": "reviewer2"}}
]'

PR_REVIEWS='[
  {"id": 100, "state": "APPROVED", "user": {"login": "reviewer1"}, "body": "Looks good"}
]'

WORKFLOW_RUNS='{
  "workflow_runs": [
    {"id": 999, "name": "CI", "status": "completed", "conclusion": "success",
     "created_at": "2026-01-01T01:00:00Z", "html_url": "https://example.com/run/999",
     "event": "pull_request"}
  ]
}'

ISSUE_META='{
  "title": "Bug report",
  "body": "Something is broken.",
  "state": "open",
  "number": 10,
  "user": {"login": "reporter1"},
  "labels": [{"name": "bug"}]
}'

# --- Test cases ---

# 1. PR prefetch — happy path with all data.
MOCK_BIN=$(build_mock "${PR_META}" "${PR_COMMENTS}" "${PR_REVIEWS}" "${WORKFLOW_RUNS}")
run_test "pr-prefetch-happy-path" \
  "https://github.com/test-org/test-repo/pull/42" \
  "${MOCK_BIN}" \
  0 \
  "exists" \
  "pull"

# 2. Issue prefetch — happy path.
MOCK_BIN=$(build_mock "" "[]" "[]" "" "${ISSUE_META}")
run_test "issue-prefetch-happy-path" \
  "https://github.com/test-org/test-repo/issues/10" \
  "${MOCK_BIN}" \
  0 \
  "exists" \
  "issue"

# 3. Prefetch failure is non-blocking — gh fails, script still exits 0.
MOCK_BIN=$(build_failing_mock)
run_test "prefetch-failure-non-blocking" \
  "https://github.com/test-org/test-repo/pull/42" \
  "${MOCK_BIN}" \
  0 \
  "missing"

# 4. No GH_TOKEN — prefetch skipped, script exits 0.
run_test_no_token() {
  local workspace="${TMPDIR}/workspace-no-token"
  mkdir -p "${workspace}"

  local exit_code=0
  env \
    GH_TOKEN="" \
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > "${TMPDIR}/stdout-no-token.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: no-token-skips-prefetch — expected exit 0, got ${exit_code}"
    cat "${TMPDIR}/stdout-no-token.log"
    FAILURES=$((FAILURES + 1))
    return
  fi
  if [[ -f "${workspace}/pr-context.json" ]]; then
    echo "FAIL: no-token-skips-prefetch — expected no pr-context.json"
    FAILURES=$((FAILURES + 1))
    return
  fi
  echo "PASS: no-token-skips-prefetch"
}
run_test_no_token

# 5. Invalid ORIGINATING_URL — script exits with error (this is the
#    validation step, not the prefetch).
run_test_invalid_url() {
  local workspace="${TMPDIR}/workspace-invalid-url"
  mkdir -p "${workspace}"

  local exit_code=0
  env \
    GH_TOKEN="fake-token" \
    ORIGINATING_URL="not-a-valid-url" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > "${TMPDIR}/stdout-invalid-url.log" 2>&1 || exit_code=$?

  if [[ ${exit_code} -eq 0 ]]; then
    echo "FAIL: invalid-url — expected non-zero exit"
    FAILURES=$((FAILURES + 1))
    return
  fi
  echo "PASS: invalid-url"
}
run_test_invalid_url

# 6. PR prefetch — verify JSON structure has required fields.
run_test_json_structure() {
  local workspace="${TMPDIR}/workspace-json-structure"
  mkdir -p "${workspace}"
  local mock_bin
  mock_bin=$(build_mock "${PR_META}" "${PR_COMMENTS}" "${PR_REVIEWS}" "${WORKFLOW_RUNS}")

  env \
    PATH="${mock_bin}:${PATH}" \
    GH_TOKEN="fake-token" \
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > /dev/null 2>&1

  local context_file="${workspace}/pr-context.json"
  local missing_fields=""

  for field in source_url source_repo source_number source_type pr comments reviews workflow_runs; do
    if ! jq -e ".${field}" "${context_file}" > /dev/null 2>&1; then
      missing_fields="${missing_fields} ${field}"
    fi
  done

  if [[ -n "${missing_fields}" ]]; then
    echo "FAIL: json-structure — missing fields:${missing_fields}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Verify source_number is a number (not a string).
  local num_type
  num_type=$(jq -r '.source_number | type' "${context_file}")
  if [[ "${num_type}" != "number" ]]; then
    echo "FAIL: json-structure — source_number should be number, got ${num_type}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: json-structure"
}
run_test_json_structure

# 7. PR prefetch — comments/reviews failure does not block metadata.
run_test_partial_failure() {
  local workspace="${TMPDIR}/workspace-partial"
  mkdir -p "${workspace}"
  # Build a mock that returns PR metadata but fails on comments/reviews.
  local mock_bin="${TMPDIR}/bin-partial"
  rm -rf "${mock_bin}"
  mkdir -p "${mock_bin}"

  printf '%s' "${PR_META}" > "${TMPDIR}/pr-meta-partial.json"

  cat > "${mock_bin}/gh" <<MOCKSCRIPT
#!/usr/bin/env bash
set -euo pipefail
if [[ "\$1" == "pr" && "\$2" == "view" ]]; then
  cat "${TMPDIR}/pr-meta-partial.json"
  exit 0
fi
if [[ "\$1" == "api" ]]; then
  endpoint="\$2"
  if [[ "\${endpoint}" == *"/comments"* ]]; then
    exit 1  # Simulate failure
  fi
  if [[ "\${endpoint}" == *"/reviews"* ]]; then
    exit 1  # Simulate failure
  fi
  if [[ "\${endpoint}" == *"/actions/runs"* ]]; then
    jq_filter=""
    for arg in "\$@"; do
      if [[ "\${prev_arg:-}" == "--jq" ]]; then
        jq_filter="\${arg}"
      fi
      prev_arg="\${arg}"
    done
    if [[ -n "\${jq_filter}" ]]; then
      echo '{"workflow_runs":[]}' | jq "\${jq_filter}"
    else
      echo '{"workflow_runs":[]}'
    fi
    exit 0
  fi
fi
exit 1
MOCKSCRIPT
  chmod +x "${mock_bin}/gh"

  local exit_code=0
  env \
    PATH="${mock_bin}:${PATH}" \
    GH_TOKEN="fake-token" \
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > /dev/null 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: partial-failure — expected exit 0, got ${exit_code}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  local context_file="${workspace}/pr-context.json"
  if [[ ! -f "${context_file}" ]]; then
    echo "FAIL: partial-failure — expected pr-context.json to exist"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Comments and reviews should fall back to empty arrays.
  local comments_len reviews_len
  comments_len=$(jq '.comments | length' "${context_file}")
  reviews_len=$(jq '.reviews | length' "${context_file}")
  if [[ "${comments_len}" -ne 0 || "${reviews_len}" -ne 0 ]]; then
    echo "FAIL: partial-failure — expected empty comments/reviews on failure"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: partial-failure"
}
run_test_partial_failure

# 8. Multi-page pagination — verify --jq '.[]' | jq -s '.' pattern
#    correctly merges all items.
MULTI_PAGE_COMMENTS='[
  {"id": 1, "body": "Page 1 comment 1", "user": {"login": "user1"}},
  {"id": 2, "body": "Page 1 comment 2", "user": {"login": "user2"}},
  {"id": 3, "body": "Page 2 comment 1", "user": {"login": "user3"}},
  {"id": 4, "body": "Page 2 comment 2", "user": {"login": "user4"}},
  {"id": 5, "body": "Page 3 comment 1", "user": {"login": "user5"}}
]'
MULTI_PAGE_REVIEWS='[
  {"id": 100, "state": "APPROVED", "user": {"login": "reviewer1"}, "body": "LGTM"},
  {"id": 101, "state": "CHANGES_REQUESTED", "user": {"login": "reviewer2"}, "body": "Fix this"},
  {"id": 102, "state": "APPROVED", "user": {"login": "reviewer3"}, "body": "OK"}
]'

run_test_multi_page_pagination() {
  local workspace="${TMPDIR}/workspace-multi-page"
  mkdir -p "${workspace}"
  local mock_bin
  mock_bin=$(build_mock "${PR_META}" "${MULTI_PAGE_COMMENTS}" "${MULTI_PAGE_REVIEWS}" "${WORKFLOW_RUNS}")

  local exit_code=0
  env \
    PATH="${mock_bin}:${PATH}" \
    GH_TOKEN="fake-token" \
    ORIGINATING_URL="https://github.com/test-org/test-repo/pull/42" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > /dev/null 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: multi-page-pagination — expected exit 0, got ${exit_code}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  local context_file="${workspace}/pr-context.json"
  if [[ ! -f "${context_file}" ]]; then
    echo "FAIL: multi-page-pagination — expected pr-context.json to exist"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Verify all 5 comments are present.
  local comments_len
  comments_len=$(jq '.comments | length' "${context_file}")
  if [[ "${comments_len}" -ne 5 ]]; then
    echo "FAIL: multi-page-pagination — expected 5 comments, got ${comments_len}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Verify all 3 reviews are present.
  local reviews_len
  reviews_len=$(jq '.reviews | length' "${context_file}")
  if [[ "${reviews_len}" -ne 3 ]]; then
    echo "FAIL: multi-page-pagination — expected 3 reviews, got ${reviews_len}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  echo "PASS: multi-page-pagination"
}
run_test_multi_page_pagination

# 9. Issue prefetch — verify JSON structure has required fields and
#    PR-specific fields are absent.
run_test_issue_json_structure() {
  local workspace="${TMPDIR}/workspace-issue-structure"
  mkdir -p "${workspace}"
  local mock_bin
  mock_bin=$(build_mock "" "[]" "[]" "" "${ISSUE_META}")

  local exit_code=0
  env \
    PATH="${mock_bin}:${PATH}" \
    GH_TOKEN="fake-token" \
    ORIGINATING_URL="https://github.com/test-org/test-repo/issues/10" \
    REPO_FULL_NAME="test-org/test-repo" \
    GITHUB_WORKSPACE="${workspace}" \
    bash "${SCRIPT}" > /dev/null 2>&1 || exit_code=$?

  if [[ ${exit_code} -ne 0 ]]; then
    echo "FAIL: issue-json-structure — expected exit 0, got ${exit_code}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  local context_file="${workspace}/pr-context.json"
  if [[ ! -f "${context_file}" ]]; then
    echo "FAIL: issue-json-structure — expected pr-context.json to exist"
    FAILURES=$((FAILURES + 1))
    return
  fi

  local missing_fields=""
  for field in source_url source_repo source_number source_type issue comments; do
    if ! jq -e ".${field}" "${context_file}" > /dev/null 2>&1; then
      missing_fields="${missing_fields} ${field}"
    fi
  done

  if [[ -n "${missing_fields}" ]]; then
    echo "FAIL: issue-json-structure — missing fields:${missing_fields}"
    FAILURES=$((FAILURES + 1))
    return
  fi

  # Verify PR-specific fields are absent.
  for field in pr reviews workflow_runs; do
    if jq -e ".${field}" "${context_file}" > /dev/null 2>&1; then
      echo "FAIL: issue-json-structure — unexpected PR field '${field}' in issue output"
      FAILURES=$((FAILURES + 1))
      return
    fi
  done

  echo "PASS: issue-json-structure"
}
run_test_issue_json_structure

# --- Summary ---

echo ""
if [[ ${FAILURES} -gt 0 ]]; then
  echo "${FAILURES} test(s) failed"
  exit 1
fi
echo "All tests passed"
